package hrm

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

const termsAAD = "hrm.contracts.terms"

func (s *Service) contractTypeOptions(ctx context.Context) ([]record.Option, error) {
	rows, err := store.New(platform.DBFrom(ctx)).ListContractTypes(ctx)
	out := make([]record.Option, len(rows))
	for i, r := range rows {
		out[i] = record.Option{Value: strconv.FormatInt(r.ID, 10), Label: r.Name}
	}
	return out, err
}

// ContractTypes lists every kind of contract, inactive ones included.
func (s *Service) ContractTypes(ctx context.Context) ([]ContractType, error) {
	rows, err := store.New(platform.DBFrom(ctx)).ListContractTypes(ctx)
	out := make([]ContractType, len(rows))
	for i, r := range rows {
		out[i] = ContractType{ID: r.ID, ContractTypeInput: ContractTypeInput{Name: r.Name, FixedTerm: r.FixedTerm, Active: r.Active}}
	}
	return out, err
}

// SaveContractType creates a kind of contract (id 0) or replaces one. Kinds are tenant-wide.
func (s *Service) SaveContractType(ctx context.Context, id int64, in ContractTypeInput) (int64, error) {
	if sc, err := s.d.IAM.Scope(ctx, "hrm", PermContractTypeManage); err != nil || !sc.All {
		return 0, platform.OrErr(err, platform.ErrForbidden)
	}
	err := platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		var err error
		if id == 0 {
			id, err = q.CreateContractType(ctx, store.CreateContractTypeParams{Name: in.Name, FixedTerm: in.FixedTerm, Active: in.Active})
		} else {
			var n int64
			n, err = q.UpdateContractType(ctx, store.UpdateContractTypeParams{ID: id, Name: in.Name, FixedTerm: in.FixedTerm, Active: in.Active})
			if err == nil && n == 0 {
				return platform.ErrNotFound
			}
		}
		if isCheck(err, "contract_types_name_key") {
			return ErrContractTypeTaken
		}
		if err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "hrm.contract_type_saved", map[string]any{"id": id, "name": in.Name, "fixed_term": in.FixedTerm, "active": in.Active})
	})
	return id, err
}

// canWriteContract: writing a contract means entering its money terms.
func (s *Service) canWriteContract(ctx context.Context, unit int64) (bool, error) {
	if ok, err := s.allowed(ctx, PermContractEdit, unit); err != nil || !ok {
		return false, err
	}
	return s.allowed(ctx, PermSalaryView, unit)
}

// canContract goes by the contract's org unit. Sending and cancelling need no salary
// permission, since neither shows nor changes the terms; its attachments do.
func (s *Service) canContract(ctx context.Context, id int64, action record.Action) (bool, error) {
	r, err := store.New(platform.DBFrom(ctx)).GetContract(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// The signed scan shows the salary, and so does the print. Attaching also once posted:
	// contracts are often signed after they are entered.
	if action == record.Print {
		action = record.ViewFiles
	}
	action, files := filesAction(action)
	if files {
		if ok, err := s.allowed(ctx, PermSalaryView, r.OrgUnitID); err != nil || !ok {
			return false, err
		}
	}
	switch action {
	case record.View, record.Export:
		return s.allowed(ctx, PermContractView, r.OrgUnitID)
	case record.Edit:
		return s.canWriteContract(ctx, r.OrgUnitID)
	case record.Post, record.Cancel:
		return s.allowed(ctx, PermContractEdit, r.OrgUnitID)
	}
	return false, nil
}

// contractSubjects is the account of the employee, who never approves their own contract.
func (s *Service) contractSubjects(ctx context.Context, id int64) ([]int64, error) {
	r, err := store.New(platform.DBFrom(ctx)).GetContract(ctx, id)
	if err != nil || !r.EmployeeUserID.Valid {
		return nil, err
	}
	return []int64{r.EmployeeUserID.Int64}, nil
}

// contractTransition keeps posted originals of an employee from overlapping and appendices
// tied to a posted original. It locks the employee row, so the posts and cancels of one
// employee's contracts run one at a time.
func (s *Service) contractTransition(ctx context.Context, d record.Doc, from record.Status) error {
	posted, ok := payrollEffect(d, from)
	if !ok {
		return nil
	}
	q := store.New(platform.DBFrom(ctx))
	r, err := q.GetContract(ctx, d.ID)
	if err != nil {
		return err
	}
	appendix := r.ParentID.Valid
	// Affected range: an original runs to its end date or forever, an appendix to its original's end.
	end := platform.DatePtr(r.EndDate)
	if appendix {
		end = platform.DatePtr(r.ParentEnd)
	}
	if err := checkPayrollPeriods(ctx, d.LegalEntityID, *platform.DatePtr(r.StartDate), end); err != nil {
		return err
	}
	if _, err := q.LockEmployee(ctx, r.EmployeeID); err != nil {
		return err
	}
	switch {
	case posted && appendix:
		// Read again under the lock: the original may have just been cancelled.
		if r, err = q.GetContract(ctx, d.ID); err != nil {
			return err
		}
		if r.ParentStatus.String != string(record.Posted) {
			return ErrContractParentDraft
		}
		if !within(r.StartDate, r.ParentStart, r.ParentEnd) {
			return ErrAppendixDate
		}
	case posted:
		overlaps, err := q.ContractOverlaps(ctx, store.ContractOverlapsParams{EmployeeID: r.EmployeeID, ID: r.ID, StartDate: r.StartDate, EndDate: r.EndDate})
		if err != nil {
			return err
		}
		if overlaps {
			return ErrContractOverlaps
		}
	case !appendix:
		has, err := q.HasPostedAppendix(ctx, pgtype.Int8{Int64: r.ID, Valid: true})
		if err != nil {
			return err
		}
		if has {
			return ErrContractHasAppendices
		}
	}
	return nil
}

func within(d, start, end pgtype.Date) bool {
	return !d.Time.Before(start.Time) && (!end.Valid || !d.Time.After(end.Time))
}

// contractAt returns the money terms in force for an employee at date (YYYY-MM-DD):
// those of the posted original covering it, or of its latest posted appendix effective
// by then. ok is false when no posted contract covers the date. The caller checks
// hrm.salary.view; the read is audited on the contract the terms came from.
func (s *Service) contractAt(ctx context.Context, employeeID int64, date string) (t ContractTerms, ok bool, err error) {
	err = platform.InTx(ctx, func(ctx context.Context) error {
		r, err := store.New(platform.DBFrom(ctx)).ContractTermsAt(ctx, store.ContractTermsAtParams{EmployeeID: employeeID, At: platform.NullDate(&date)})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := s.d.Audit.RecordFor(ctx, "hrm.contract_terms_viewed", audit.Ref{Type: contractType, ID: r.ID}, nil); err != nil {
			return err
		}
		t, err = openTerms(ctx, r.Terms)
		ok = err == nil
		return err
	})
	return t, ok, err
}

// EmployeeContracts lists an employee's contracts in the actor's scope, each original
// followed by its appendices, without their terms.
func (s *Service) EmployeeContracts(ctx context.Context, employeeID int64) ([]ContractListItem, error) {
	e, err := s.visible(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	sc, err := s.d.IAM.Scope(ctx, "hrm", PermContractView)
	if err != nil {
		return nil, err
	}
	// An appendix is a new contract of the employee, so it needs what creating one needs.
	write, err := s.canWriteContract(ctx, e.OrgUnitID)
	if err != nil {
		return nil, err
	}
	write = write && platform.ProductGate(ctx, "hrm", platform.ClassWrite) == nil
	rows, err := store.New(platform.DBFrom(ctx)).ListEmployeeContracts(ctx, store.ListEmployeeContractsParams{EmployeeID: employeeID, AllUnits: sc.All, Units: sc.Units})
	out := make([]ContractListItem, len(rows))
	numbers := map[int64]string{}
	for _, r := range rows {
		numbers[r.ID] = r.Number
	}
	for i, r := range rows {
		out[i] = ContractListItem{
			ID: r.ID, Number: r.Number, Status: r.Status, EmployeeID: e.ID, EmployeeCode: e.Code, EmployeeName: e.FullName,
			ParentID: platform.Int8Ptr(r.ParentID), ContractTypeName: r.ContractTypeName,
			StartDate: *platform.DatePtr(r.StartDate), EndDate: platform.DatePtr(r.EndDate), AllowedActions: []string{},
		}
		if n, ok := numbers[r.ParentID.Int64]; r.ParentID.Valid && ok {
			out[i].ParentNumber = &n
		}
		if write && !r.ParentID.Valid && r.Status != string(record.Cancelled) {
			out[i].AllowedActions = append(out[i].AllowedActions, "add_appendix")
		}
	}
	return out, err
}

// Contracts lists every contract and appendix in the actor's scope, without their terms.
func (s *Service) Contracts(ctx context.Context, f ContractFilter) (ContractList, error) {
	out := ContractList{Items: []ContractListItem{}}
	sc, err := s.d.IAM.Scope(ctx, "hrm", PermContractView)
	if err != nil || !sc.Any() {
		return out, err
	}
	today, err := s.today(ctx)
	if err != nil {
		return out, err
	}
	rows, err := store.New(platform.DBFrom(ctx)).ListContracts(ctx, store.ListContractsParams{
		AllUnits: sc.All, Units: sc.Units, Status: f.Status,
		ContractTypeID: pgtype.Int8{Int64: f.ContractTypeID, Valid: f.ContractTypeID != 0},
		OrgUnitID:      pgtype.Int8{Int64: f.OrgUnitID, Valid: f.OrgUnitID != 0},
		EmployeeID:     pgtype.Int8{Int64: f.EmployeeID, Valid: f.EmployeeID != 0},
		Expiring:       f.Expiring, Today: today,
		Sort: f.Sort, Lim: int32(f.PageSize), Off: int32((f.Page - 1) * f.PageSize),
	})
	for _, r := range rows {
		out.Total = r.Total
		out.Items = append(out.Items, ContractListItem{
			ID: r.ID, Number: r.Number, Status: r.Status, EmployeeID: r.EmployeeID, EmployeeCode: r.EmployeeCode,
			EmployeeName: r.EmployeeName, ParentID: platform.Int8Ptr(r.ParentID), ParentNumber: platform.TextPtr(r.ParentNumber),
			ContractTypeName: r.ContractTypeName, StartDate: *platform.DatePtr(r.StartDate), EndDate: platform.DatePtr(r.EndDate),
			AllowedActions: []string{},
		})
	}
	return out, err
}

// Contract returns one contract; one the actor may not view does not exist. The terms
// come only with hrm.salary.view, and returning them is audited.
func (s *Service) Contract(ctx context.Context, id int64) (Contract, error) {
	var out Contract
	err := platform.InTx(ctx, func(ctx context.Context) error {
		r, err := s.visibleContract(ctx, id)
		if err != nil {
			return err
		}
		ref := record.Ref{Type: contractType, ID: id}
		d, err := s.d.Record.Get(ctx, ref)
		if err != nil {
			return err
		}
		actions, err := s.d.Record.DocumentActions(ctx, d)
		if err != nil {
			return err
		}
		// Printing changes nothing, so a locked period does not hold it back.
		print, err := s.d.Record.AllowedActions(ctx, contractType, id, record.Print)
		if err != nil {
			return err
		}
		actions = append(actions, print...)
		out = Contract{
			ID: id, Number: d.Number, Status: string(d.Status), Version: d.Version, EmployeeID: r.EmployeeID,
			EmployeeCode: r.EmployeeCode, EmployeeName: r.EmployeeName, ParentID: platform.Int8Ptr(r.ParentID),
			ParentNumber: platform.TextPtr(r.ParentNumber), ContractTypeID: r.ContractTypeID, ContractTypeName: r.ContractTypeName,
			StartDate: *platform.DatePtr(r.StartDate), EndDate: platform.DatePtr(r.EndDate), AllowedActions: actions,
		}
		if ok, err := s.allowed(ctx, PermSalaryView, r.OrgUnitID); err != nil || !ok {
			return err
		}
		if err := s.d.Audit.RecordFor(ctx, "hrm.contract_terms_viewed", audit.Ref(ref), nil); err != nil {
			return err
		}
		t, err := openTerms(ctx, r.Terms)
		out.Terms = &t
		return err
	})
	return out, err
}

// CreateContract adds a draft contract, or an appendix when ParentID is set.
func (s *Service) CreateContract(ctx context.Context, in NewContract) (int64, error) {
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		e, err := q.GetEmployee(ctx, in.EmployeeID)
		if errors.Is(err, pgx.ErrNoRows) {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		if ok, err := s.canWriteContract(ctx, e.OrgUnitID); err != nil || !ok {
			return platform.OrErr(err, platform.ErrForbidden)
		}
		var parent *store.GetContractRow
		if in.ParentID != nil {
			p, err := q.GetContract(ctx, *in.ParentID)
			if errors.Is(err, pgx.ErrNoRows) || err == nil && (p.EmployeeID != e.ID || p.ParentID.Valid) {
				return ErrContractParent
			}
			if err != nil {
				return err
			}
			parent = &p
		}
		h, row, typeName, err := s.contractHeader(ctx, e.OrgUnitID, parent, &in.ContractFields)
		if err != nil {
			return err
		}
		d, err := s.d.Record.Create(ctx, contractType, h)
		if err != nil {
			return err
		}
		id = d.ID
		row.ID, row.EmployeeID, row.ParentID = id, e.ID, platform.NullInt8(in.ParentID)
		if err := q.CreateContract(ctx, row); err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "hrm.contract_created", audit.Ref{Type: contractType, ID: id}, contractChanges(nil, "", &in.ContractFields, typeName))
	})
	return id, err
}

// UpdateContract replaces the fields of a draft at the version the actor saw.
func (s *Service) UpdateContract(ctx context.Context, id int64, in ContractUpdate) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		old, err := s.visibleContract(ctx, id)
		if err != nil {
			return err
		}
		var parent *store.GetContractRow
		if old.ParentID.Valid {
			p, err := q.GetContract(ctx, old.ParentID.Int64)
			if err != nil {
				return err
			}
			parent = &p
		}
		h, row, typeName, err := s.contractHeader(ctx, old.OrgUnitID, parent, &in.ContractFields)
		if err != nil {
			return err
		}
		// Edit checks the actor may write the contract, salary included, before the old terms are opened.
		if _, err := s.d.Record.Edit(ctx, record.Ref{Type: contractType, ID: id}, in.Version, h); err != nil {
			return err
		}
		if err := q.UpdateContract(ctx, store.UpdateContractParams{
			ID: id, ContractTypeID: row.ContractTypeID, StartDate: row.StartDate, EndDate: row.EndDate, Terms: row.Terms,
		}); err != nil {
			return err
		}
		terms, err := openTerms(ctx, old.Terms)
		if err != nil {
			return err
		}
		before := ContractFields{ContractTypeID: old.ContractTypeID, StartDate: *platform.DatePtr(old.StartDate), EndDate: platform.DatePtr(old.EndDate), Terms: terms}
		return s.d.Audit.RecordChanges(ctx, "hrm.contract_updated", audit.Ref{Type: contractType, ID: id}, contractChanges(&before, old.ContractTypeName, &in.ContractFields, typeName))
	})
}

// DeleteContract removes a draft at the version the actor saw; an original with appendices stays.
func (s *Service) DeleteContract(ctx context.Context, id int64, version int32) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.visibleContract(ctx, id); err != nil {
			return err
		}
		if err := s.d.Record.Delete(ctx, record.Ref{Type: contractType, ID: id}, version); err != nil {
			return err
		}
		err := store.New(platform.DBFrom(ctx)).DeleteContract(ctx, id)
		if isCheck(err, "contracts_parent_id_fkey") {
			return ErrContractHasAppendices
		}
		return err
	})
}

func (s *Service) visibleContract(ctx context.Context, id int64) (store.GetContractRow, error) {
	if ok, err := s.d.Record.Can(ctx, contractType, id, record.View); err != nil || !ok {
		return store.GetContractRow{}, platform.OrErr(err, platform.ErrNotFound)
	}
	return store.New(platform.DBFrom(ctx)).GetContract(ctx, id)
}

// contractHeader checks the fields and builds the record header and the row. An appendix
// (parent set) takes its original's kind and has no end date; in is updated to match.
func (s *Service) contractHeader(ctx context.Context, unit int64, parent *store.GetContractRow, in *ContractFields) (h record.Header, row store.CreateContractParams, typeName string, err error) {
	start := platform.NullDate(&in.StartDate)
	if parent != nil {
		in.ContractTypeID, in.EndDate = parent.ContractTypeID, nil
		if !within(start, parent.StartDate, parent.EndDate) {
			return h, row, "", ErrAppendixDate
		}
	} else if in.EndDate != nil && *in.EndDate < in.StartDate {
		return h, row, "", ErrContractDates
	}
	if in.Terms.Salary < 0 {
		return h, row, "", ErrContractTerms
	}
	if in.Terms.Lines == nil {
		in.Terms.Lines = []ContractLine{}
	}
	for _, l := range in.Terms.Lines {
		// A salary allowance is always taxable; only a support may be tax-free.
		if l.Amount < 0 || strings.TrimSpace(l.Name) == "" || l.Kind != "allowance" && l.Kind != "support" || l.Kind == "allowance" && !l.Taxable {
			return h, row, "", ErrContractTerms
		}
	}
	t, err := store.New(platform.DBFrom(ctx)).GetContractType(ctx, in.ContractTypeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return h, row, "", platform.ErrNotFound
	}
	if err != nil {
		return h, row, "", err
	}
	// An appendix keeps its original's kind even after the kind stops being offered.
	if !t.Active && parent == nil {
		return h, row, "", ErrContractTypeInactive
	}
	switch {
	case parent == nil && t.FixedTerm && in.EndDate == nil:
		return h, row, "", ErrContractEndRequired
	case parent == nil && !t.FixedTerm && in.EndDate != nil:
		return h, row, "", ErrContractEndForbidden
	}
	raw, err := json.Marshal(in.Terms)
	if err != nil {
		return h, row, "", err
	}
	row = store.CreateContractParams{ContractTypeID: t.ID, StartDate: start, EndDate: platform.NullDate(in.EndDate), Terms: platform.Encrypt(ctx, termsAAD, raw)}
	h = record.Header{Date: in.StartDate, OrgUnitID: unit, Fields: map[string]string{"contract_kind": strconv.FormatInt(t.ID, 10)}}
	return h, row, t.Name, nil
}

// contractChanges names the kind rather than its id; the terms are stored encrypted.
func contractChanges(old *ContractFields, oldType string, n *ContractFields, newType string) []audit.Change {
	var o ContractFields
	var oldTerms *ContractTerms
	if old != nil {
		o, oldTerms = *old, &old.Terms
	}
	return []audit.Change{
		{Field: "contract_type", Old: oldType, New: newType},
		{Field: "start_date", Old: o.StartDate, New: n.StartDate},
		{Field: "end_date", Old: o.EndDate, New: n.EndDate},
		{Field: "terms", Old: oldTerms, New: &n.Terms, Sensitive: true},
	}
}

func openTerms(ctx context.Context, sealed []byte) (ContractTerms, error) {
	var t ContractTerms
	raw, err := platform.Decrypt(ctx, termsAAD, sealed)
	if err != nil {
		return t, err
	}
	return t, json.Unmarshal(raw, &t)
}
