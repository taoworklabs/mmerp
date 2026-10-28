package hrm

import (
	"context"
	"errors"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// sensitiveFields lists the encrypted columns of hrm.employees, in API order.
var sensitiveFields = []string{"national_id", "social_insurance_no", "tax_code", "bank_account"}

// employeeUnit locates an employee for record: the org unit the profile belongs to.
func employeeUnit(ctx context.Context, id int64) (int64, error) {
	e, err := store.GetEmployee.Run(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, platform.ErrNotFound
	}
	return e.OrgUnitID, err
}

// employeeRefine: a profile's files also need the sensitive permission.
func (s *Service) employeeRefine(ctx context.Context, id int64, action record.Action) (bool, error) {
	if action != record.ViewFiles && action != record.Attach {
		return true, nil
	}
	unit, err := employeeUnit(ctx, id)
	if err != nil {
		return false, err
	}
	return s.allowed(ctx, PermSensitive, unit)
}

// Employees lists the employees within the actor's view scope.
func (s *Service) Employees(ctx context.Context, f EmployeeFilter) (EmployeeList, error) {
	out := EmployeeList{Items: []EmployeeListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "hrm", PermView)
	if err != nil {
		return out, err
	}
	today, err := s.d.Setting.Today(ctx)
	if err != nil {
		return out, err
	}
	page, err := employeeList.Run(ctx, employeeQuery(sc, f, today))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, EmployeeListItem{
			ID: r.ID, Code: r.Code, FullName: r.FullName, OrgUnitID: r.OrgUnitID, OrgUnitName: r.OrgUnitName,
			Email: platform.TextPtr(r.Email), Phone: platform.TextPtr(r.Phone), HireDate: *platform.DatePtr(r.HireDate), TerminationDate: platform.DatePtr(r.TerminationDate),
			Status: status(r.TerminationDate, today),
		})
	}
	return out, nil
}

// EmployeeActions lists what the actor may do with employees before picking one: create
// (needs edit somewhere, and the product enabled), view_sensitive (to fill sensitive
// fields when creating), import_leave_balances and import_employees. The frontend shows the create button
// and fields from this.
func (s *Service) EmployeeActions(ctx context.Context) ([]string, error) {
	out, err := s.d.Record.TypeActions(ctx, employeeType)
	if err != nil {
		return nil, err
	}
	sens, err := s.d.IAM.Scope(ctx, "hrm", PermSensitive)
	if err != nil {
		return nil, err
	}
	if sens.Any() {
		out = append(out, "view_sensitive")
	}
	adjust, err := s.d.IAM.Scope(ctx, "hrm", PermBalanceAdjust)
	if err != nil {
		return nil, err
	}
	if adjust.Any() && platform.ProductGate(ctx, "hrm", platform.ClassWrite) == nil {
		out = append(out, "import_leave_balances")
	}
	// "create" already means edit somewhere with the product enabled.
	if slices.Contains(out, "create") {
		out = append(out, "import_employees")
	}
	return out, nil
}

// Employee returns one employee; outside the actor's scope it does not exist.
func (s *Service) Employee(ctx context.Context, id int64) (Employee, error) {
	e, err := s.visible(ctx, id)
	if err != nil {
		return Employee{}, err
	}
	actions, err := s.d.Record.Actions(ctx, record.Doc{Ref: record.Ref{Type: employeeType, ID: id}})
	if err != nil {
		return Employee{}, err
	}
	var tails SensitiveTails
	if ok, err := s.allowed(ctx, PermSensitive, e.OrgUnitID); err != nil {
		return Employee{}, err
	} else if ok {
		actions = append(actions, "view_sensitive")
		sealed := sealedOf(store.HrmEmployee{NationalID: e.NationalID, SocialInsuranceNo: e.SocialInsuranceNo, TaxCode: e.TaxCode, BankAccount: e.BankAccount})
		for field, out := range map[string]**string{"national_id": &tails.NationalID, "social_insurance_no": &tails.SocialInsuranceNo, "tax_code": &tails.TaxCode, "bank_account": &tails.BankAccount} {
			v, err := open(ctx, field, sealed[field])
			if err != nil {
				return Employee{}, err
			}
			*out = digitTail(v)
		}
	}
	if ok, err := s.allowed(ctx, PermBalanceAdjust, e.OrgUnitID); err != nil {
		return Employee{}, err
	} else if ok && platform.ProductGate(ctx, "hrm", platform.ClassWrite) == nil {
		actions = append(actions, "adjust_leave_balance")
	}
	if ok, err := s.allowed(ctx, PermContractView, e.OrgUnitID); err != nil {
		return Employee{}, err
	} else if ok {
		actions = append(actions, "view_contracts")
	}
	if ok, err := s.canWriteContract(ctx, e.OrgUnitID); err != nil {
		return Employee{}, err
	} else if ok && platform.ProductGate(ctx, "hrm", platform.ClassWrite) == nil {
		actions = append(actions, "create_contract")
	}
	today, err := s.d.Setting.Today(ctx)
	if err != nil {
		return Employee{}, err
	}
	var current *CurrentContract
	if slices.Contains(actions, "view_contracts") {
		sc, err := s.d.IAM.Scope(ctx, "hrm", PermContractView)
		if err != nil {
			return Employee{}, err
		}
		r, err := store.CurrentContract.Run(ctx, store.CurrentContractParams{EmployeeID: id, At: today, AllUnits: sc.All, Units: sc.Units})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return Employee{}, err
		}
		if err == nil {
			current = &CurrentContract{ID: r.ID, Number: r.Number, ContractType: r.ContractTypeName, StartDate: *platform.DatePtr(r.StartDate), EndDate: platform.DatePtr(r.EndDate)}
		}
	}
	return Employee{
		ID: e.ID, Code: e.Code,
		EmployeeFields: EmployeeFields{
			FullName: e.FullName, DateOfBirth: platform.DatePtr(e.DateOfBirth), Gender: platform.TextPtr(e.Gender),
			Phone: platform.TextPtr(e.Phone), Email: platform.TextPtr(e.Email), Address: platform.TextPtr(e.Address), OrgUnitID: e.OrgUnitID,
			ManagerID: platform.Int8Ptr(e.ManagerID), UserLogin: platform.TextPtr(e.UserLogin), HireDate: *platform.DatePtr(e.HireDate),
			TerminationDate: platform.DatePtr(e.TerminationDate),
		},
		OrgUnitName: e.OrgUnitName, ManagerCode: platform.TextPtr(e.ManagerCode), ManagerName: platform.TextPtr(e.ManagerName), Status: status(e.TerminationDate, today),
		Sensitive: SensitivePresence{
			NationalID: e.NationalID != nil, SocialInsuranceNo: e.SocialInsuranceNo != nil,
			TaxCode: e.TaxCode != nil, BankAccount: e.BankAccount != nil,
		},
		SensitiveTail:   tails,
		CurrentContract: current,
		AllowedActions:  actions,
	}, nil
}

// visible returns an employee the actor may view; one they may not does not exist.
func (s *Service) visible(ctx context.Context, id int64) (store.GetEmployeeRow, error) {
	if err := s.d.Record.Visible(ctx, record.Ref{Type: employeeType, ID: id}, record.View); err != nil {
		return store.GetEmployeeRow{}, err
	}
	return store.GetEmployee.Run(ctx, id)
}

// RevealSensitive returns one sensitive field. The read is audited before the value
// leaves, and a failed audit fails the read.
func (s *Service) RevealSensitive(ctx context.Context, id int64, field string) (*string, error) {
	var out *string
	err := platform.InTx(ctx, func(ctx context.Context) error {
		e, err := s.visible(ctx, id)
		if err != nil {
			return err
		}
		if err := s.require(ctx, PermSensitive, e.OrgUnitID); err != nil {
			return err
		}
		if err := s.d.Audit.RecordFor(ctx, "hrm.employee_sensitive_viewed", audit.Ref{Type: employeeType, ID: id}, map[string]any{"field": field}); err != nil {
			return err
		}
		out, err = open(ctx, field, sealedOf(store.HrmEmployee{NationalID: e.NationalID, SocialInsuranceNo: e.SocialInsuranceNo, TaxCode: e.TaxCode, BankAccount: e.BankAccount})[field])
		return err
	})
	return out, err
}

// CreateEmployee adds an employee in an org unit where the actor may edit.
func (s *Service) CreateEmployee(ctx context.Context, in NewEmployee) (int64, error) {
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		row, err := s.prepare(ctx, store.HrmEmployee{}, in.EmployeeInput)
		if err != nil {
			return err
		}
		if row.Code, err = s.d.Numbering.Code(ctx, employeeType, in.Code, employeeCodeTaken); err != nil {
			return err
		}
		id, err = store.CreateEmployee.Run(ctx, store.CreateEmployeeParams{
			Code: row.Code, FullName: row.FullName, DateOfBirth: row.DateOfBirth, Gender: row.Gender, Phone: row.Phone,
			Email: row.Email, Address: row.Address, OrgUnitID: row.OrgUnitID, ManagerID: row.ManagerID, UserID: row.UserID,
			HireDate: row.HireDate, TerminationDate: row.TerminationDate, NationalID: row.NationalID,
			SocialInsuranceNo: row.SocialInsuranceNo, TaxCode: row.TaxCode, BankAccount: row.BankAccount,
		})
		if err != nil {
			return writeErr(err)
		}
		row.ID = id
		return s.recordChanges(ctx, "hrm.employee_created", id, store.HrmEmployee{}, row)
	})
	return id, err
}

// UpdateEmployee replaces the plain fields and the sensitive fields listed in in.
func (s *Service) UpdateEmployee(ctx context.Context, id int64, in EmployeeInput) error {
	return s.updateEmployee(ctx, id, in, true)
}

// updateEmployee is UpdateEmployee; without checkCycle the caller checks the manager tree itself.
func (s *Service) updateEmployee(ctx context.Context, id int64, in EmployeeInput, checkCycle bool) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.visible(ctx, id); err != nil {
			return err
		}
		old, err := store.LockEmployee.Run(ctx, id)
		if err != nil {
			return err
		}
		if err := s.require(ctx, PermEdit, old.OrgUnitID); err != nil {
			return err
		}
		row, err := s.prepare(ctx, old, in)
		if err != nil {
			return err
		}
		if row.ManagerID.Valid && row.ManagerID != old.ManagerID {
			if row.ManagerID.Int64 == id {
				return ErrManagerSelf
			}
		}
		if checkCycle && row.ManagerID.Valid && row.ManagerID != old.ManagerID {
			// ponytail: one lock for all manager changes; per-tree locks if HR edits get busy.
			if err := store.LockManagerTree.Run(ctx); err != nil {
				return err
			}
			chain, err := store.ManagerChain.Run(ctx, row.ManagerID.Int64)
			if err != nil {
				return err
			}
			if slices.Contains(chain, id) {
				m, err := store.GetEmployee.Run(ctx, row.ManagerID.Int64)
				if err != nil {
					return err
				}
				return &platform.Error{Status: ErrManagerCycle.Status, Code: ErrManagerCycle.Code, Params: map[string]any{"name": m.Code + " · " + m.FullName}}
			}
		}
		if err := store.UpdateEmployee.Run(ctx, store.UpdateEmployeeParams{
			ID: row.ID, FullName: row.FullName, DateOfBirth: row.DateOfBirth, Gender: row.Gender, Phone: row.Phone, Email: row.Email,
			Address: row.Address, OrgUnitID: row.OrgUnitID, ManagerID: row.ManagerID, UserID: row.UserID, HireDate: row.HireDate,
			TerminationDate: row.TerminationDate, NationalID: row.NationalID, SocialInsuranceNo: row.SocialInsuranceNo, TaxCode: row.TaxCode,
			BankAccount: row.BankAccount,
		}); err != nil {
			return writeErr(err)
		}
		return s.recordChanges(ctx, "hrm.employee_updated", id, old, row)
	})
}

// prepare checks permissions for writing in over old and builds the new row.
func (s *Service) prepare(ctx context.Context, old store.HrmEmployee, in EmployeeInput) (store.HrmEmployee, error) {
	if err := s.require(ctx, PermEdit, in.OrgUnitID); err != nil {
		return old, err
	}
	if in.TerminationDate != nil && *in.TerminationDate < in.HireDate {
		return old, ErrTerminationBeforeHire
	}
	// Only a termination changes the date once the employee exists.
	if was, now := platform.DatePtr(old.TerminationDate), in.TerminationDate; old.ID != 0 && ((was == nil) != (now == nil) || was != nil && *was != *now) {
		return old, ErrTerminationViaDocument
	}
	row := store.HrmEmployee{
		ID: old.ID, Code: old.Code, FullName: in.FullName, DateOfBirth: platform.NullDate(in.DateOfBirth), Gender: platform.NullText(in.Gender),
		Phone: platform.NullText(in.Phone), Email: platform.NullText(in.Email), Address: platform.NullText(in.Address), OrgUnitID: in.OrgUnitID,
		ManagerID: platform.NullInt8(in.ManagerID), HireDate: platform.NullDate(&in.HireDate), TerminationDate: platform.NullDate(in.TerminationDate),
		NationalID: old.NationalID, SocialInsuranceNo: old.SocialInsuranceNo, TaxCode: old.TaxCode, BankAccount: old.BankAccount,
	}
	// A manager outside the actor's view scope is treated as missing, so ids cannot be probed.
	if in.ManagerID != nil && (old.ID == 0 || !old.ManagerID.Valid || old.ManagerID.Int64 != *in.ManagerID) {
		if _, err := s.visible(ctx, *in.ManagerID); errors.Is(err, platform.ErrNotFound) {
			return old, ErrManagerNotFound
		} else if err != nil {
			return old, err
		}
	}
	if in.UserLogin != nil {
		uid, err := s.d.IAM.UserIDByLogin(ctx, *in.UserLogin)
		if errors.Is(err, platform.ErrNotFound) {
			return old, ErrUserNotFound
		}
		if err != nil {
			return old, err
		}
		row.UserID = pgtype.Int8{Int64: uid, Valid: true}
	}
	// Moving into a unit where the actor sees sensitive data from one where they do not would grant them access.
	if old.ID != 0 && old.OrgUnitID != in.OrgUnitID {
		atNew, err := s.allowed(ctx, PermSensitive, in.OrgUnitID)
		if err != nil {
			return old, err
		}
		if atNew {
			if err := s.require(ctx, PermSensitive, old.OrgUnitID); err != nil {
				return old, err
			}
		}
	}
	if v := in.Sensitive; v != nil {
		// Both the old and the new unit must allow it, so a move cannot launder access.
		var was []int64
		if old.ID != 0 {
			was = append(was, old.OrgUnitID)
		}
		if err := s.require(ctx, PermSensitive, in.OrgUnitID, was...); err != nil {
			return old, err
		}
		for field, val := range map[string]*string{"national_id": v.NationalID, "social_insurance_no": v.SocialInsuranceNo, "tax_code": v.TaxCode, "bank_account": v.BankAccount} {
			if val != nil {
				*sealedRef(&row, field) = seal(ctx, field, *val)
			}
		}
	}
	return row, nil
}

func (s *Service) recordChanges(ctx context.Context, action string, id int64, old, row store.HrmEmployee) error {
	var oldUnit any = old.OrgUnitID
	if old.ID == 0 {
		oldUnit = nil
	}
	changes := []audit.Change{
		{Field: "code", Old: old.Code, New: row.Code},
		{Field: "full_name", Old: old.FullName, New: row.FullName},
		{Field: "date_of_birth", Old: platform.DatePtr(old.DateOfBirth), New: platform.DatePtr(row.DateOfBirth)},
		{Field: "gender", Old: platform.TextPtr(old.Gender), New: platform.TextPtr(row.Gender)},
		{Field: "phone", Old: platform.TextPtr(old.Phone), New: platform.TextPtr(row.Phone)},
		{Field: "email", Old: platform.TextPtr(old.Email), New: platform.TextPtr(row.Email)},
		{Field: "address", Old: platform.TextPtr(old.Address), New: platform.TextPtr(row.Address)},
		{Field: "org_unit_id", Old: oldUnit, New: row.OrgUnitID},
		{Field: "manager_id", Old: platform.Int8Ptr(old.ManagerID), New: platform.Int8Ptr(row.ManagerID)},
		{Field: "user_id", Old: platform.Int8Ptr(old.UserID), New: platform.Int8Ptr(row.UserID)},
		{Field: "hire_date", Old: platform.DatePtr(old.HireDate), New: platform.DatePtr(row.HireDate)},
		{Field: "termination_date", Old: platform.DatePtr(old.TerminationDate), New: platform.DatePtr(row.TerminationDate)},
	}
	before, after := sealedOf(old), sealedOf(row)
	for _, f := range sensitiveFields {
		if slices.Equal(before[f], after[f]) {
			continue
		}
		o, err := open(ctx, f, before[f])
		if err != nil {
			return err
		}
		n, err := open(ctx, f, after[f])
		if err != nil {
			return err
		}
		changes = append(changes, audit.Change{Field: f, Old: o, New: n, Sensitive: true})
	}
	return s.d.Audit.RecordChanges(ctx, action, audit.Ref{Type: employeeType, ID: row.ID}, changes)
}

func sealedOf(e store.HrmEmployee) map[string][]byte {
	return map[string][]byte{"national_id": e.NationalID, "social_insurance_no": e.SocialInsuranceNo, "tax_code": e.TaxCode, "bank_account": e.BankAccount}
}

func sealedRef(e *store.HrmEmployee, field string) *[]byte {
	return map[string]*[]byte{"national_id": &e.NationalID, "social_insurance_no": &e.SocialInsuranceNo, "tax_code": &e.TaxCode, "bank_account": &e.BankAccount}[field]
}

// seal encrypts a sensitive column value; "" clears it.
func seal(ctx context.Context, field, v string) []byte {
	if v == "" {
		return nil
	}
	return platform.Encrypt(ctx, "hrm.employees."+field, []byte(v))
}

func open(ctx context.Context, field string, sealed []byte) (*string, error) {
	if sealed == nil {
		return nil, nil
	}
	pt, err := platform.Decrypt(ctx, "hrm.employees."+field, sealed)
	v := string(pt)
	return &v, err
}

// digitTail is the last four digits of v when it holds at least eight, so a short value is never
// mostly disclosed; nil otherwise.
func digitTail(v *string) *string {
	if v == nil {
		return nil
	}
	var digits []rune
	for _, r := range *v {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	if len(digits) < 8 {
		return nil
	}
	tail := string(digits[len(digits)-4:])
	return &tail
}

// status: terminated once the termination date has passed in the tenant's time zone.
func status(termination, today pgtype.Date) string {
	if termination.Valid && termination.Time.Before(today.Time) {
		return "terminated"
	}
	return "active"
}

func writeErr(err error) error {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return err
	}
	// Joined, so a caller can still tell that the transaction is now aborted.
	switch pgErr.ConstraintName {
	case "employees_code_key":
		return errors.Join(ErrCodeTaken, err)
	case "employees_user_id_key":
		return errors.Join(ErrUserAlreadyLinked, err)
	case "employees_manager_id_fkey", "employees_check":
		return errors.Join(ErrManagerNotFound, err)
	case "employees_org_unit_id_fkey":
		return errors.Join(platform.ErrNotFound, err)
	}
	return err
}

// employeeCodeTaken reports whether a code is used, for numbering to skip it.
func employeeCodeTaken(ctx context.Context, code string) (bool, error) {
	return store.EmployeeCodeTaken.Run(ctx, code)
}
