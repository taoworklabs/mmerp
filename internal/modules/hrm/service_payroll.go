package hrm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

const (
	linesAAD       = "hrm.payroll_lines.data"
	adjustmentsAAD = "hrm.payroll_adjustments.data"
)

// payrollComputeArgs computes a draft payroll at the version it was asked for, as its requester.
type payrollComputeArgs struct {
	ID      int64 `json:"id"`
	Version int32 `json:"version"`
}

func (payrollComputeArgs) Kind() string { return "hrm.payroll_compute" }

func (payrollComputeArgs) Spec() platform.JobSpec {
	return platform.JobSpec{Product: "hrm", Class: platform.ClassWrite, Notify: true}
}

// The requester waits for it: a few quick retries.
func (payrollComputeArgs) InsertOpts() river.InsertOpts { return river.InsertOpts{MaxAttempts: 3} }

type payrollWorker struct {
	river.WorkerDefaults[payrollComputeArgs]
	s *Service
}

func (w *payrollWorker) Work(ctx context.Context, j *river.Job[payrollComputeArgs]) error {
	return w.s.computePayroll(ctx, j)
}

// payrollLine is what a payroll line keeps, encrypted: the inputs read and the result.
type payrollLine struct {
	Input  payInput  `json:"input"`
	Result payResult `json:"result"`
}

// adjustment is the encrypted part of an adjustment.
type adjustment struct {
	Amount int64  `json:"amount"`
	Reason string `json:"reason"`
}

// canPayroll goes by the payroll's legal entity.
func (s *Service) canPayroll(ctx context.Context, id int64, action record.Action) (bool, error) {
	p, err := store.New(platform.DBFrom(ctx)).GetPayroll(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	switch action {
	case record.View, record.Export:
		return s.allowed(ctx, PermPayrollView, p.LegalEntityID)
	case record.Edit, record.Post, record.Cancel:
		return s.allowed(ctx, PermPayrollEdit, p.LegalEntityID)
	}
	return false, nil
}

// PayrollActions lists what the actor may do before picking a payroll: create.
func (s *Service) PayrollActions(ctx context.Context) ([]string, error) {
	out := []string{}
	edit, err := s.d.IAM.Scope(ctx, "hrm", PermPayrollEdit)
	if err != nil {
		return nil, err
	}
	// Creating computes, which reads salaries: both are needed at one same legal entity.
	salary, err := s.d.IAM.Scope(ctx, "hrm", PermSalaryView)
	both := edit.All && salary.Any() || salary.All && edit.Any() || slices.ContainsFunc(edit.Units, salary.Has)
	if both && platform.ProductGate(ctx, "hrm", platform.ClassWrite) == nil {
		out = append(out, "create")
	}
	return out, err
}

// CreatePayroll adds a draft payroll of a legal entity and month and queues its computation.
func (s *Service) CreatePayroll(ctx context.Context, in NewPayroll) (PayrollJob, error) {
	var out PayrollJob
	start, end, err := period(in.Month)
	if err != nil {
		return out, err
	}
	if err := s.checkLegalEntity(ctx, in.LegalEntityID); err != nil {
		return out, err
	}
	if err := s.require(ctx, PermPayrollEdit, in.LegalEntityID); err != nil {
		return out, err
	}
	if err := s.require(ctx, PermSalaryView, in.LegalEntityID); err != nil {
		return out, err
	}
	// Fail now rather than in the job when a timesheet or a parameter is missing.
	if _, err := s.readBasis(ctx, in.LegalEntityID, 0, start.Time, end.Time); err != nil {
		return out, err
	}
	err = platform.InTx(ctx, func(ctx context.Context) error {
		d, err := s.d.Record.Create(ctx, payrollType, record.Header{Date: *platform.DatePtr(end), OrgUnitID: in.LegalEntityID})
		if err != nil {
			return err
		}
		out.ID = d.ID
		if err := store.New(platform.DBFrom(ctx)).CreatePayroll(ctx, store.CreatePayrollParams{ID: d.ID, PeriodStart: start, PeriodEnd: end}); err != nil {
			return err
		}
		out.JobID, err = platform.Enqueue(ctx, payrollComputeArgs{ID: d.ID, Version: d.Version})
		return err
	})
	return out, err
}

// RecomputePayroll queues a computation of a draft at the version the actor saw.
func (s *Service) RecomputePayroll(ctx context.Context, id int64, version int32) (int64, error) {
	p, d, err := s.editablePayroll(ctx, id, version)
	if err != nil {
		return 0, err
	}
	if err := s.require(ctx, PermSalaryView, p.LegalEntityID); err != nil {
		return 0, err
	}
	return platform.Enqueue(ctx, payrollComputeArgs{ID: id, Version: d.Version})
}

// editablePayroll reads a draft the actor may edit, at the version they saw.
func (s *Service) editablePayroll(ctx context.Context, id int64, version int32) (store.GetPayrollRow, record.Doc, error) {
	p, err := s.visiblePayroll(ctx, id)
	if err != nil {
		return p, record.Doc{}, err
	}
	d, err := s.d.Record.Get(ctx, record.Ref{Type: payrollType, ID: id})
	if err != nil {
		return p, d, err
	}
	if ok, err := s.d.Record.Can(ctx, payrollType, id, record.Edit); err != nil || !ok {
		return p, d, platform.OrErr(err, platform.ErrForbidden)
	}
	switch {
	case d.Status != record.Draft:
		return p, d, record.ErrNotEditable
	case d.Version != version:
		return p, d, record.ErrVersionConflict
	}
	return p, d, nil
}

// computePayroll gathers the inputs and computes every employee's pay, in one transaction
// that also completes the job. It is an edit at the version asked for, so the total cost
// becomes the document's amount and a page showing the old figures is stale.
func (s *Service) computePayroll(ctx context.Context, j *river.Job[payrollComputeArgs]) error {
	ref := record.Ref{Type: payrollType, ID: j.Args.ID}
	return platform.InTx(ctx, func(ctx context.Context) error {
		p, err := s.visiblePayroll(ctx, ref.ID)
		if errors.Is(err, platform.ErrNotFound) {
			return platform.ErrForbidden
		}
		if err != nil {
			return err
		}
		// Checked before any salary is decrypted; record.Edit checks again under its locks.
		if ok, err := s.d.Record.Can(ctx, payrollType, ref.ID, record.Edit); err != nil || !ok {
			return platform.OrErr(err, platform.ErrForbidden)
		}
		if err := s.require(ctx, PermSalaryView, p.LegalEntityID); err != nil {
			return err
		}
		b, err := s.readBasis(ctx, p.LegalEntityID, ref.ID, p.PeriodStart.Time, p.PeriodEnd.Time)
		if err != nil {
			return err
		}
		in, err := s.payInputs(ctx, b, ref.ID)
		if err != nil {
			return err
		}
		prm, err := payParams(b.Params, b.Region)
		if err != nil {
			return err
		}
		res := calcPayroll(prm, platform.Rounding(b.Rounding), in)
		var cost int64
		for i := range b.Employees {
			cost += res[i].Cost
		}
		if _, err := s.d.Record.Edit(ctx, ref, j.Args.Version, record.Header{Date: b.End, OrgUnitID: p.LegalEntityID, Amount: &cost}); err != nil {
			return err
		}
		q := store.New(platform.DBFrom(ctx))
		if err := q.DeletePayrollLines(ctx, ref.ID); err != nil {
			return err
		}
		for i, e := range b.Employees {
			raw, err := json.Marshal(payrollLine{Input: in[i], Result: res[i]})
			if err != nil {
				return err
			}
			if err := q.InsertPayrollLine(ctx, store.InsertPayrollLineParams{PayrollID: ref.ID, EmployeeID: e.ID, OrgUnitID: e.OrgUnitID,
				Data: platform.Encrypt(ctx, linesAAD, raw)}); err != nil {
				return err
			}
		}
		src := store.InsertPayrollSourcesParams{PayrollID: ref.ID}
		for _, x := range b.Sources {
			src.DocTypes, src.DocIds, src.Versions = append(src.DocTypes, x.DocType), append(src.DocIds, x.DocID), append(src.Versions, x.Version)
		}
		if err := q.DeletePayrollSources(ctx, ref.ID); err != nil {
			return err
		}
		if err := q.InsertPayrollSources(ctx, src); err != nil {
			return err
		}
		hash, err := b.hash()
		if err != nil {
			return err
		}
		if err := q.SavePayrollResult(ctx, store.SavePayrollResultParams{ID: ref.ID, InputsHash: hash}); err != nil {
			return err
		}
		if err := s.d.Audit.RecordFor(ctx, "hrm.payroll_computed", audit.Ref(ref), map[string]any{"employees": len(b.Employees), "sources": len(b.Sources)}); err != nil {
			return err
		}
		return platform.CompleteJob(ctx, j, nil)
	})
}

// sourcesChanged reports whether the payroll's sources differ from those it was computed from.
func (s *Service) sourcesChanged(ctx context.Context, p store.GetPayrollRow) (bool, error) {
	b, err := s.readBasis(ctx, p.LegalEntityID, p.ID, p.PeriodStart.Time, p.PeriodEnd.Time)
	if err != nil {
		// A timesheet cancelled since, for one, is a changed source.
		if e, ok := errors.AsType[*platform.Error](err); ok && e.Code == "payroll_timesheets_missing" {
			return true, nil
		}
		return false, err
	}
	hash, err := b.hash()
	return !bytes.Equal(hash, p.InputsHash), err
}

// payrollBeforeSubmit sends only a payroll computed from its current sources.
func (s *Service) payrollBeforeSubmit(ctx context.Context, d record.Doc) error {
	p, err := store.New(platform.DBFrom(ctx)).GetPayroll(ctx, d.ID)
	if err != nil {
		return err
	}
	if !p.ComputedAt.Valid {
		return ErrPayrollNotComputed
	}
	changed, err := s.sourcesChanged(ctx, p)
	if err == nil && changed {
		return ErrPayrollSourceChanged
	}
	return err
}

// payrollTransition closes the period when a payroll is posted and reopens it when the
// payroll holding it is cancelled. It holds the legal entity's payroll lock FOR UPDATE, so
// no source changes until commit, and checks the sources again under it.
func (s *Service) payrollTransition(ctx context.Context, d record.Doc, from record.Status) error {
	posted, ok := payrollEffect(d, from)
	if !ok {
		return nil
	}
	q := store.New(platform.DBFrom(ctx))
	p, err := q.GetPayroll(ctx, d.ID)
	if err != nil {
		return err
	}
	le := p.LegalEntityID
	if err := q.EnsurePayrollLock(ctx, le); err != nil {
		return err
	}
	if err := q.LockPayrollLock(ctx, le); err != nil {
		return err
	}
	if err := q.EnsurePayrollPeriod(ctx, store.EnsurePayrollPeriodParams{LegalEntityID: le, PeriodStart: p.PeriodStart, PeriodEnd: p.PeriodEnd}); err != nil {
		return err
	}
	holder, err := q.PostedPayroll(ctx, store.PostedPayrollParams{LegalEntityID: le, PeriodStart: p.PeriodStart})
	if err != nil {
		return err
	}
	ref := record.Ref{Type: payrollType, ID: d.ID}
	set := func(id pgtype.Int8) error {
		return q.SetPostedPayroll(ctx, store.SetPostedPayrollParams{LegalEntityID: le, PeriodStart: p.PeriodStart, PostedPayrollID: id})
	}
	if !posted {
		if holder.Int64 != d.ID {
			return ErrPayrollNotHolding
		}
		if err := set(pgtype.Int8{}); err != nil {
			return err
		}
		return s.d.Posting.Void(ctx, ref)
	}
	if holder.Valid {
		return ErrPayrollAlreadyPosted
	}
	if !p.ComputedAt.Valid {
		return ErrPayrollNotComputed
	}
	if changed, err := s.sourcesChanged(ctx, p); err != nil || changed {
		return platform.OrErr(err, ErrPayrollSourceChanged)
	}
	lines, err := s.payrollLines(ctx, d.ID)
	if err != nil {
		return err
	}
	if err := s.d.Posting.Record(ctx, ref, le, d.Date, postingLines(le, lines)); err != nil {
		return err
	}
	return set(pgtype.Int8{Int64: d.ID, Valid: true})
}

// postingLines sums the pay by department, and what is owed to the state by legal entity:
// never one person's amount on its own.
func postingLines(le int64, lines []PayrollLine) []posting.Line {
	type key struct {
		kind posting.Kind
		unit int64
	}
	sums := map[key]int64{}
	var order []key
	add := func(k posting.Kind, unit, amount int64) {
		kk := key{k, unit}
		if _, ok := sums[kk]; !ok {
			order = append(order, kk)
		}
		sums[kk] += amount
	}
	for _, l := range lines {
		add(posting.SalaryExpense, l.OrgUnitID, l.Gross)
		add(posting.EmployerInsuranceExpense, l.OrgUnitID, l.employerInsurance())
		add(posting.SalaryPayable, l.OrgUnitID, l.Net)
		add(posting.InsurancePayable, le, l.employeeInsurance()+l.employerInsurance())
		add(posting.PitPayable, le, l.IncomeTax)
	}
	out := []posting.Line{}
	for _, k := range order {
		if sums[k] != 0 {
			out = append(out, posting.Line{Kind: k.kind, OrgUnitID: k.unit, Amount: sums[k]})
		}
	}
	return out
}

// payrollLines decrypts every employee's pay of a payroll, by employee code.
func (s *Service) payrollLines(ctx context.Context, id int64) ([]PayrollLine, error) {
	rows, err := store.New(platform.DBFrom(ctx)).PayrollLines(ctx, id)
	out := make([]PayrollLine, len(rows))
	for i, r := range rows {
		raw, err := platform.Decrypt(ctx, linesAAD, r.Data)
		if err != nil {
			return nil, err
		}
		var l payrollLine
		if err := json.Unmarshal(raw, &l); err != nil {
			return nil, err
		}
		out[i] = PayrollLine{EmployeeID: r.EmployeeID, EmployeeCode: r.Code, EmployeeName: r.FullName, OrgUnitID: r.OrgUnitID,
			StandardDays: l.Input.StandardDays, PaidDays: l.Result.PaidDays.String(), OvertimeHours: l.Result.OvertimeHours.String(),
			Warnings: l.Result.Warnings, PayrollAmounts: l.Result.PayrollAmounts}
		if out[i].Warnings == nil {
			out[i].Warnings = []string{}
		}
	}
	return out, err
}

// adjustments decrypts the adjustments of a payroll.
func (s *Service) adjustments(ctx context.Context, id int64) ([]PayrollAdjustment, error) {
	rows, err := store.New(platform.DBFrom(ctx)).PayrollAdjustments(ctx, id)
	out := make([]PayrollAdjustment, len(rows))
	for i, r := range rows {
		raw, err := platform.Decrypt(ctx, adjustmentsAAD, r.Data)
		if err != nil {
			return nil, err
		}
		var a adjustment
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, err
		}
		out[i] = PayrollAdjustment{EmployeeCode: r.Code, EmployeeName: r.FullName, PayrollAdjustmentInput: PayrollAdjustmentInput{
			EmployeeID: r.EmployeeID, Amount: a.Amount, SourcePeriod: (*platform.DatePtr(r.SourcePeriod))[:7], Reason: a.Reason}}
	}
	return out, err
}

// SavePayrollAdjustments replaces the adjustments of a draft at the version the actor saw and
// queues a new computation; until it is done the payroll counts as not computed.
func (s *Service) SavePayrollAdjustments(ctx context.Context, id int64, in PayrollAdjustments) (int64, error) {
	var job int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		p, err := s.visiblePayroll(ctx, id)
		if err != nil {
			return err
		}
		ref := record.Ref{Type: payrollType, ID: id}
		old, err := s.d.Record.Get(ctx, ref)
		if err != nil {
			return err
		}
		d, err := s.d.Record.Edit(ctx, ref, in.Version, record.Header{Date: old.Date, OrgUnitID: old.OrgUnitID, Amount: old.Amount})
		if err != nil {
			return err
		}
		if err := s.require(ctx, PermSalaryView, p.LegalEntityID); err != nil {
			return err
		}
		before, err := s.adjustments(ctx, id)
		if err != nil {
			return err
		}
		q := store.New(platform.DBFrom(ctx))
		if err := q.DeletePayrollAdjustments(ctx, id); err != nil {
			return err
		}
		for _, a := range in.Items {
			start, _, err := period(a.SourcePeriod)
			if err != nil || a.Amount == 0 || !start.Time.Before(p.PeriodStart.Time) {
				return ErrAdjustment
			}
			// Only an employee of the payroll's legal entity: their pay is read into it.
			e, err := q.GetEmployee(ctx, a.EmployeeID)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAdjustment
			}
			if err != nil {
				return err
			}
			if le, err := s.d.IAM.LegalEntityOf(ctx, e.OrgUnitID); err != nil || le != p.LegalEntityID {
				return platform.OrErr(err, ErrAdjustment)
			}
			raw, err := json.Marshal(adjustment{Amount: a.Amount, Reason: a.Reason})
			if err != nil {
				return err
			}
			if err := q.InsertPayrollAdjustment(ctx, store.InsertPayrollAdjustmentParams{PayrollID: id, EmployeeID: a.EmployeeID, SourcePeriod: start,
				Data: platform.Encrypt(ctx, adjustmentsAAD, raw)}); err != nil {
				return err
			}
		}
		if err := q.ClearPayrollComputed(ctx, id); err != nil {
			return err
		}
		oldItems := make([]PayrollAdjustmentInput, len(before))
		for i, a := range before {
			oldItems[i] = a.PayrollAdjustmentInput
		}
		if err := s.d.Audit.RecordChanges(ctx, "hrm.payroll_adjusted", audit.Ref(ref), []audit.Change{
			{Field: "adjustments", Old: oldItems, New: in.Items, Sensitive: true},
		}); err != nil {
			return err
		}
		job, err = platform.Enqueue(ctx, payrollComputeArgs{ID: id, Version: d.Version})
		return err
	})
	return job, err
}

// Payrolls lists the payrolls of the legal entities in the actor's scope, latest period first.
func (s *Service) Payrolls(ctx context.Context, f PayrollFilter) (PayrollList, error) {
	out := PayrollList{Items: []PayrollListItem{}}
	sc, err := s.d.IAM.Scope(ctx, "hrm", PermPayrollView)
	if err != nil || !sc.Any() {
		return out, err
	}
	var start pgtype.Date
	if f.Month != "" {
		if start, _, err = period(f.Month); err != nil {
			return out, err
		}
	}
	rows, err := store.New(platform.DBFrom(ctx)).ListPayrolls(ctx, store.ListPayrollsParams{
		AllUnits: sc.All, Units: sc.Units, Status: f.Status, LegalEntityID: pgtype.Int8{Int64: f.LegalEntityID, Valid: f.LegalEntityID != 0},
		PeriodStart: start, Lim: int32(f.PageSize), Off: int32((f.Page - 1) * f.PageSize),
	})
	for _, r := range rows {
		out.Total = r.Total
		out.Items = append(out.Items, PayrollListItem{ID: r.ID, Number: r.Number, Status: r.Status, LegalEntityID: r.LegalEntityID,
			LegalEntityName: r.LegalEntityName, PeriodStart: *platform.DatePtr(r.PeriodStart), PeriodEnd: *platform.DatePtr(r.PeriodEnd),
			ComputedAt: instant(r.ComputedAt)})
	}
	return out, err
}

// Payroll returns a payroll with its department totals; one the actor may not view does not
// exist. Each employee's pay and the adjustments come only with hrm.salary.view, audited.
func (s *Service) Payroll(ctx context.Context, id int64) (Payroll, error) {
	var out Payroll
	err := platform.InTx(ctx, func(ctx context.Context) error {
		p, err := s.visiblePayroll(ctx, id)
		if err != nil {
			return err
		}
		ref := record.Ref{Type: payrollType, ID: id}
		d, err := s.d.Record.Get(ctx, ref)
		if err != nil {
			return err
		}
		actions, err := s.d.Record.DocumentActions(ctx, d)
		if err != nil {
			return err
		}
		salary, err := s.allowed(ctx, PermSalaryView, p.LegalEntityID)
		if err != nil {
			return err
		}
		if salary && slices.Contains(actions, "edit") {
			actions = append(actions, "compute", "adjust")
		}
		if ok, err := s.d.Record.Can(ctx, payrollType, id, record.Export); err != nil {
			return err
		} else if ok && salary && p.ComputedAt.Valid {
			actions = append(actions, "export")
		}
		out = Payroll{ID: id, Number: d.Number, Status: string(d.Status), Version: d.Version, LegalEntityID: p.LegalEntityID,
			LegalEntityName: p.LegalEntityName, PeriodStart: *platform.DatePtr(p.PeriodStart), PeriodEnd: *platform.DatePtr(p.PeriodEnd),
			ComputedAt: instant(p.ComputedAt), AllowedActions: actions}
		if p.ComputedAt.Valid && (d.Status == record.Draft || d.Status == record.PendingApproval) {
			if out.SourcesChanged, err = s.sourcesChanged(ctx, p); err != nil {
				return err
			}
		}
		lines, err := s.payrollLines(ctx, id)
		if err != nil {
			return err
		}
		if out.Totals, err = payrollTotals(ctx, lines); err != nil || !salary {
			return err
		}
		if err := s.d.Audit.RecordFor(ctx, "hrm.payroll_lines_viewed", audit.Ref(ref), nil); err != nil {
			return err
		}
		out.Lines = lines
		out.Adjustments, err = s.adjustments(ctx, id)
		return err
	})
	return out, err
}

// payrollTotals sums the decrypted lines by department, in department id order.
func payrollTotals(ctx context.Context, lines []PayrollLine) ([]PayrollTotal, error) {
	byUnit := map[int64]*PayrollTotal{}
	for _, l := range lines {
		t, ok := byUnit[l.OrgUnitID]
		if !ok {
			t = &PayrollTotal{OrgUnitID: l.OrgUnitID}
			byUnit[l.OrgUnitID] = t
		}
		t.Employees++
		t.add(l.PayrollAmounts)
	}
	out := []PayrollTotal{}
	for _, k := range slices.Sorted(maps.Keys(byUnit)) {
		out = append(out, *byUnit[k])
	}
	ids := make([]int64, len(out))
	for i, t := range out {
		ids[i] = t.OrgUnitID
	}
	names, err := store.New(platform.DBFrom(ctx)).OrgUnitNames(ctx, ids)
	for i := range out {
		for _, n := range names {
			if n.ID == out[i].OrgUnitID {
				out[i].OrgUnitName = n.Name
			}
		}
	}
	return out, err
}

// DeletePayroll removes a draft at the version the actor saw; its lines go with it.
func (s *Service) DeletePayroll(ctx context.Context, id int64, version int32) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.visiblePayroll(ctx, id); err != nil {
			return err
		}
		if err := s.d.Record.Delete(ctx, record.Ref{Type: payrollType, ID: id}, version); err != nil {
			return err
		}
		return store.New(platform.DBFrom(ctx)).DeletePayroll(ctx, id)
	})
}

func (s *Service) visiblePayroll(ctx context.Context, id int64) (store.GetPayrollRow, error) {
	if ok, err := s.d.Record.Can(ctx, payrollType, id, record.View); err != nil || !ok {
		return store.GetPayrollRow{}, platform.OrErr(err, platform.ErrNotFound)
	}
	return store.New(platform.DBFrom(ctx)).GetPayroll(ctx, id)
}

func instant(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	return new(t.Time.UTC().Format(time.RFC3339))
}

// payrollColumns are the money columns of the export, in order.
var payrollColumns = []string{"earned", "overtime", "unused_leave", "adjustment", "gross", "exempt", "insurance_base",
	"social_insurance", "health_insurance", "unemployment_insurance", "personal_deduction", "dependent_deduction", "taxable",
	"income_tax", "net", "employer_social_insurance", "employer_health_insurance", "employer_unemployment_insurance", "employer_union_fee", "cost"}

func (a PayrollAmounts) columns() []any {
	return []any{a.Earned, a.Overtime, a.UnusedLeave, a.Adjustment, a.Gross, a.Exempt, a.InsuranceBase, a.SocialInsurance,
		a.HealthInsurance, a.UnemploymentIns, a.PersonalDeduction, a.DependentDeduction, a.Taxable, a.IncomeTax, a.Net,
		a.EmployerSocial, a.EmployerHealth, a.EmployerUnemploy, a.EmployerUnion, a.Cost}
}

// exportablePayroll reads the params of a payroll export the actor may run: it needs
// hrm.salary.view.
func (s *Service) exportablePayroll(ctx context.Context, params json.RawMessage) (store.GetPayrollRow, error) {
	var in PayrollParams
	if err := json.Unmarshal(params, &in); err != nil || in.PayrollID == 0 {
		return store.GetPayrollRow{}, dataio.ErrInvalidParams
	}
	p, err := s.visiblePayroll(ctx, in.PayrollID)
	if errors.Is(err, platform.ErrNotFound) {
		return p, platform.ErrForbidden
	}
	if err != nil {
		return p, err
	}
	if ok, err := s.d.Record.Can(ctx, payrollType, p.ID, record.Export); err != nil || !ok {
		return p, platform.OrErr(err, platform.ErrForbidden)
	}
	return p, s.require(ctx, PermSalaryView, p.LegalEntityID)
}

// exportPayroll writes one row per employee with every amount, department by department,
// each followed by its total, then the grand total.
func (s *Service) exportPayroll(ctx context.Context, params json.RawMessage) (dataio.Sheet, error) {
	p, err := s.exportablePayroll(ctx, params)
	if err != nil {
		return dataio.Sheet{}, err
	}
	if !p.ComputedAt.Valid {
		return dataio.Sheet{}, ErrPayrollNotComputed
	}
	d, err := s.d.Record.Get(ctx, record.Ref{Type: payrollType, ID: p.ID})
	if err != nil {
		return dataio.Sheet{}, err
	}
	loc, err := s.locale(ctx)
	if err != nil {
		return dataio.Sheet{}, err
	}
	tr := func(k string, params map[string]any) string {
		return platform.Translate(loc, "hrm.export.payroll."+k, params)
	}
	header := []string{tr("code", nil), tr("name", nil), tr("org_unit", nil), tr("standard_days", nil), tr("paid_days", nil), tr("overtime_hours", nil)}
	for _, c := range payrollColumns {
		header = append(header, tr(c, nil))
	}
	lines, err := s.payrollLines(ctx, p.ID)
	if err != nil {
		return dataio.Sheet{}, err
	}
	totals, err := payrollTotals(ctx, lines)
	if err != nil {
		return dataio.Sheet{}, err
	}
	var rows [][]any
	var all PayrollAmounts
	for _, t := range totals {
		for _, l := range lines {
			if l.OrgUnitID != t.OrgUnitID {
				continue
			}
			paid, _ := record.ParseNumber(l.PaidDays)
			hours, _ := record.ParseNumber(l.OvertimeHours)
			rows = append(rows, append([]any{l.EmployeeCode, l.EmployeeName, t.OrgUnitName, int64(l.StandardDays), paid, hours}, l.columns()...))
		}
		rows = append(rows, append([]any{"", tr("org_unit_total", map[string]any{"name": t.OrgUnitName}), t.OrgUnitName, "", "", ""}, t.columns()...))
		all.add(t.PayrollAmounts)
	}
	rows = append(rows, append([]any{"", tr("total", nil), "", "", "", ""}, all.columns()...))
	return dataio.Sheet{Name: d.Number, Header: header, Rows: rows}, nil
}
