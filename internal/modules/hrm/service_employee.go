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
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// sensitiveFields lists the encrypted columns of hrm.employees, in API order.
var sensitiveFields = []string{"national_id", "social_insurance_no", "tax_code", "bank_account"}

func (s *Service) can(ctx context.Context, id int64, action record.Action) (bool, error) {
	e, err := store.New(platform.DBFrom(ctx)).GetEmployee(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Papers on the profile (an ID card) show what the sensitive fields hold.
	action, files := filesAction(action)
	if files {
		if ok, err := s.allowed(ctx, PermSensitive, e.OrgUnitID); err != nil || !ok {
			return false, err
		}
	}
	switch action {
	case record.View, record.Export:
		return s.allowed(ctx, PermView, e.OrgUnitID)
	case record.Edit:
		return s.allowed(ctx, PermEdit, e.OrgUnitID)
	}
	return false, nil
}

// Employees lists the employees within the actor's view scope.
func (s *Service) Employees(ctx context.Context, f EmployeeFilter) (EmployeeList, error) {
	out := EmployeeList{Items: []EmployeeListItem{}}
	sc, err := s.d.IAM.Scope(ctx, "hrm", PermView)
	if err != nil || !sc.Any() {
		return out, err
	}
	tz, err := s.d.Setting.Get(ctx, setting.Timezone)
	if err != nil {
		return out, err
	}
	today, err := store.New(platform.DBFrom(ctx)).Today(ctx, tz)
	if err != nil {
		return out, err
	}
	rows, err := store.New(platform.DBFrom(ctx)).ListEmployees(ctx, store.ListEmployeesParams{
		AllUnits: sc.All, Units: sc.Units, Q: f.Q, OrgUnitID: pgtype.Int8{Int64: f.OrgUnitID, Valid: f.OrgUnitID != 0},
		Status: f.Status, ManagerOf: pgtype.Int8{Int64: f.ManagerOf, Valid: f.ManagerOf != 0}, Tz: tz, Sort: f.Sort, Lim: int32(f.PageSize), Off: int32((f.Page - 1) * f.PageSize),
	})
	for _, r := range rows {
		out.Total = r.Total
		out.Items = append(out.Items, EmployeeListItem{
			ID: r.ID, Code: r.Code, FullName: r.FullName, OrgUnitID: r.OrgUnitID, OrgUnitName: r.OrgUnitName,
			Email: platform.TextPtr(r.Email), Phone: platform.TextPtr(r.Phone), HireDate: *platform.DatePtr(r.HireDate), TerminationDate: platform.DatePtr(r.TerminationDate),
			Status: status(r.TerminationDate, today),
		})
	}
	return out, err
}

// EmployeeActions lists what the actor may do with employees before picking one: create
// (needs edit somewhere, and the product enabled), view_sensitive (to fill sensitive
// fields when creating) and import_leave_balances. The frontend shows the create button
// and fields from this.
func (s *Service) EmployeeActions(ctx context.Context) ([]string, error) {
	out := []string{}
	edit, err := s.d.IAM.Scope(ctx, "hrm", PermEdit)
	if err != nil {
		return nil, err
	}
	if edit.Any() && platform.ProductGate(ctx, "hrm", platform.ClassWrite) == nil {
		out = append(out, "create")
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
	return out, nil
}

// Employee returns one employee; outside the actor's scope it does not exist.
func (s *Service) Employee(ctx context.Context, id int64) (Employee, error) {
	e, err := s.visible(ctx, id)
	if err != nil {
		return Employee{}, err
	}
	actions, err := s.d.Record.AllowedActions(ctx, employeeType, id, record.View, record.Edit)
	if err != nil {
		return Employee{}, err
	}
	if ok, err := s.allowed(ctx, PermSensitive, e.OrgUnitID); err != nil {
		return Employee{}, err
	} else if ok {
		actions = append(actions, "view_sensitive")
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
	today, err := s.today(ctx)
	if err != nil {
		return Employee{}, err
	}
	return Employee{
		ID: e.ID,
		EmployeeFields: EmployeeFields{
			Code: e.Code, FullName: e.FullName, DateOfBirth: platform.DatePtr(e.DateOfBirth), Gender: platform.TextPtr(e.Gender),
			Phone: platform.TextPtr(e.Phone), Email: platform.TextPtr(e.Email), Address: platform.TextPtr(e.Address), OrgUnitID: e.OrgUnitID,
			ManagerID: platform.Int8Ptr(e.ManagerID), UserLogin: platform.TextPtr(e.UserLogin), HireDate: *platform.DatePtr(e.HireDate),
			TerminationDate: platform.DatePtr(e.TerminationDate),
		},
		OrgUnitName: e.OrgUnitName, ManagerCode: platform.TextPtr(e.ManagerCode), ManagerName: platform.TextPtr(e.ManagerName), Status: status(e.TerminationDate, today),
		Sensitive: SensitivePresence{
			NationalID: e.NationalID != nil, SocialInsuranceNo: e.SocialInsuranceNo != nil,
			TaxCode: e.TaxCode != nil, BankAccount: e.BankAccount != nil,
		},
		AllowedActions: actions,
	}, nil
}

func (s *Service) visible(ctx context.Context, id int64) (store.GetEmployeeRow, error) {
	e, err := store.New(platform.DBFrom(ctx)).GetEmployee(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return e, platform.ErrNotFound
	}
	if err != nil {
		return e, err
	}
	if ok, err := s.allowed(ctx, PermView, e.OrgUnitID); err != nil || !ok {
		return e, platform.OrErr(err, platform.ErrNotFound)
	}
	return e, nil
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
func (s *Service) CreateEmployee(ctx context.Context, in EmployeeInput) (int64, error) {
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		row, err := s.prepare(ctx, store.HrmEmployee{}, in)
		if err != nil {
			return err
		}
		q := store.New(platform.DBFrom(ctx))
		id, err = q.CreateEmployee(ctx, store.CreateEmployeeParams{
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
	return platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		if _, err := s.visible(ctx, id); err != nil {
			return err
		}
		old, err := q.LockEmployee(ctx, id)
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
			// ponytail: one lock for all manager changes; per-tree locks if HR edits get busy.
			if err := q.LockManagerTree(ctx); err != nil {
				return err
			}
			chain, err := q.ManagerChain(ctx, row.ManagerID.Int64)
			if err != nil {
				return err
			}
			if row.ManagerID.Int64 == id {
				return ErrManagerSelf
			}
			if slices.Contains(chain, id) {
				m, err := q.GetEmployee(ctx, row.ManagerID.Int64)
				if err != nil {
					return err
				}
				return &platform.Error{Status: ErrManagerCycle.Status, Code: ErrManagerCycle.Code, Params: map[string]any{"name": m.Code + " · " + m.FullName}}
			}
		}
		if err := q.UpdateEmployee(ctx, store.UpdateEmployeeParams(row)); err != nil {
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
	row := store.HrmEmployee{
		ID: old.ID, Code: in.Code, FullName: in.FullName, DateOfBirth: platform.NullDate(in.DateOfBirth), Gender: platform.NullText(in.Gender),
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

func (s *Service) today(ctx context.Context) (pgtype.Date, error) {
	tz, err := s.d.Setting.Get(ctx, setting.Timezone)
	if err != nil {
		return pgtype.Date{}, err
	}
	return store.New(platform.DBFrom(ctx)).Today(ctx, tz)
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
	switch pgErr.ConstraintName {
	case "employees_code_key":
		return ErrCodeTaken
	case "employees_user_id_key":
		return ErrUserAlreadyLinked
	case "employees_manager_id_fkey", "employees_check":
		return ErrManagerNotFound
	case "employees_org_unit_id_fkey":
		return platform.ErrNotFound
	}
	return err
}
