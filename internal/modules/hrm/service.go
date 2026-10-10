// Package hrm owns employee records, dependents, contracts, leave and overtime requests,
// leave balances, timesheets and the payroll lock.
package hrm

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"sync"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

const (
	employeeType  = "hrm.employee"
	leaveType     = "hrm.leave_request"
	contractType  = "hrm.contract"
	overtimeType  = "hrm.overtime_request"
	timesheetType = "hrm.timesheet"
	payrollType   = "hrm.payroll"
)

type Service struct{ d Deps }

//go:embed i18n/*.json
var i18nFiles embed.FS

// loadTranslations adds the labels approval shows as is (day kinds) to the catalog, once.
var loadTranslations = sync.OnceValue(func() error {
	sub, err := fs.Sub(i18nFiles, "i18n")
	if err != nil {
		return err
	}
	return platform.LoadTranslations(sub)
})

// NewService registers HRM roles and record types with core.
func NewService(d Deps) *Service {
	if err := loadTranslations(); err != nil {
		panic(err) // embedded files: broken only by a bad build
	}
	s := &Service{d: d}
	d.Setting.RegisterLegalEntityKey(SettingWageRegion, "1", []string{"1", "2", "3", "4"})
	d.IAM.RegisterRoles("hrm", map[string][]string{
		"hr": {PermView, PermEdit, PermLeaveView, PermLeaveEdit, PermOvertimeView, PermOvertimeEdit,
			PermContractView, PermContractEdit, PermContractTypeManage, PermTimesheetView, PermTimesheetEdit},
		"viewer":           {PermView},
		"sensitive_viewer": {PermSensitive},
		"leave_admin":      {PermBalanceAdjust, PermLeaveTypeManage},
		"payroll":          {PermSalaryView, PermPayrollView, PermPayrollEdit, PermLegalParamManage, PermCalendarManage},
		// Department totals of payrolls, without any person's amounts (e.g. accounting).
		"payroll_viewer": {PermPayrollView},
		// Approval rules apply tenant-wide, so the role is granted only tenant-wide.
		"approval_admin": {PermApprovalManage},
	}, "approval_admin")
	// Sensitive fields and every person's pay need a role of their own, even for administrators.
	d.IAM.RegisterSensitive("hrm", PermSensitive, PermSalaryView)
	d.Record.Register(record.Type{Code: employeeType, Product: "hrm", Kind: record.Catalog, Can: s.can})
	// A profile's viewer sees neither leave balances nor what sensitive data was read or changed;
	// on an employee, ViewFiles is view plus the sensitive permission.
	d.Record.RestrictHistory("hrm.leave_balance_adjusted", "")
	for _, a := range []string{"hrm.employee_sensitive_viewed", "hrm.dependents_viewed", "hrm.dependent_saved", "hrm.dependent_deleted"} {
		d.Record.RestrictHistory(a, record.ViewFiles)
	}
	d.Record.Register(record.Type{
		Code: leaveType, Product: "hrm", Kind: record.Document, NumberPrefix: "NP",
		Fields: []record.Field{
			{Key: "days", Kind: record.Number, Label: "hrm.leave.days"},
			{Key: "leave_type", Kind: record.Choice, Label: "hrm.leave.type", Options: s.leaveTypeOptions},
		},
		Can: s.canLeave, OnTransition: s.leaveTransition, Approvers: s.leaveApprovers, Subjects: s.leaveSubjects,
	})
	d.Record.Register(record.Type{
		Code: overtimeType, Product: "hrm", Kind: record.Document, NumberPrefix: "TC",
		Fields: []record.Field{
			{Key: "hours", Kind: record.Number, Label: "hrm.overtime.hours"},
			{Key: "day_kind", Kind: record.Choice, Label: "hrm.overtime.day_kind", Options: s.dayKindOptions},
		},
		Can: s.canOvertime, OnTransition: s.overtimeTransition, Approvers: s.overtimeApprovers, Subjects: s.overtimeSubjects,
	})
	d.Record.Register(record.Type{
		Code: contractType, Product: "hrm", Kind: record.Document, NumberPrefix: "HD",
		Fields: []record.Field{{Key: "contract_kind", Kind: record.Choice, Label: "hrm.contract.kind", Options: s.contractTypeOptions}},
		Can:    s.canContract, OnTransition: s.contractTransition, Subjects: s.contractSubjects,
	})
	// No approval fields and no module-chosen approvers: timesheet rules go by role.
	d.Record.Register(record.Type{
		Code: timesheetType, Product: "hrm", Kind: record.Document, NumberPrefix: "BC",
		Can: s.canTimesheet, OnTransition: s.timesheetTransition,
	})
	// The legal entity is the org unit; approval rules go by role.
	d.Record.Register(record.Type{
		Code: payrollType, Product: "hrm", Kind: record.Document, NumberPrefix: "BL",
		Can: s.canPayroll, OnTransition: s.payrollTransition, BeforeSubmit: s.payrollBeforeSubmit,
	})
	s.registerDataIO()
	s.registerPrints()
	return s
}

// filesAction maps the attachment actions onto a record's own: seeing its files as
// viewing it, attaching as editing it, whatever its status. A type whose files may show
// more than the record asks for a further permission on top.
func filesAction(a record.Action) (base record.Action, files bool) {
	switch a {
	case record.ViewFiles:
		return record.View, true
	case record.Attach:
		return record.Edit, true
	}
	return a, false
}

func (s *Service) allowed(ctx context.Context, perm string, units ...int64) (bool, error) {
	sc, err := s.d.IAM.Scope(ctx, "hrm", perm)
	if err != nil {
		return false, err
	}
	for _, u := range units {
		if !sc.Has(u) {
			return false, nil
		}
	}
	return true, nil
}

func (s *Service) require(ctx context.Context, perm string, units ...int64) error {
	ok, err := s.allowed(ctx, perm, units...)
	if err == nil && !ok {
		return platform.ErrForbidden
	}
	return err
}

// payrollEffect tells whether a status change takes effect for payroll: posted is true
// into posted, false from posted to cancelled; ok is false for every other change.
func payrollEffect(d record.Doc, from record.Status) (posted, ok bool) {
	switch {
	case d.Status == record.Posted:
		return true, true
	case from == record.Posted && d.Status == record.Cancelled:
		return false, true
	}
	return false, false
}

func isCheck(err error, constraint string) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.ConstraintName == constraint
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
