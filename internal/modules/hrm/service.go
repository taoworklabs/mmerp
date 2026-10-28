// Package hrm owns employee records, dependents, contracts, leave and overtime requests,
// leave balances, timesheets and the payroll lock.
package hrm

import (
	"context"
	"embed"
	"io/fs"
	"slices"
	"sync"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
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
	data.RegisterConstraints(data.Constraints{
		"hrm.leave_types_name_key":           ErrLeaveTypeTaken,
		"hrm.contract_types_name_key":        ErrContractTypeTaken,
		"hrm.timesheets_org_unit_period_key": ErrTimesheetExists,
	})
	d.Setting.RegisterLegalEntityKey("hrm", SettingWageRegion, "1", []string{"1", "2", "3", "4"})
	d.IAM.RegisterRoles("hrm", map[string][]string{
		"hr": {PermView, PermEdit, PermLeaveView, PermLeaveEdit, PermOvertimeView, PermOvertimeEdit,
			PermContractView, PermContractEdit, PermContractTypeManage, PermTimesheetView, PermTimesheetEdit,
			PermTerminationView, PermTerminationEdit},
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
	d.Numbering.RegisterSeries(employeeType, "hrm", "NV")
	d.Record.Register(record.Type{Code: employeeType, Product: "hrm", Kind: record.Catalog,
		// Papers on a profile (an ID card) show what the sensitive fields hold.
		Perms:  record.Perms{View: PermView, Edit: PermEdit, ViewFiles: PermView, Attach: PermEdit},
		Locate: employeeUnit, Refine: s.employeeRefine,
		HasData: func(ctx context.Context) (bool, error) { return store.AnyEmployee.Run(ctx) }})
	// A profile's viewer sees neither leave balances nor what sensitive data was read or changed;
	// on an employee, ViewFiles is view plus the sensitive permission.
	d.Record.RestrictHistory("hrm.leave_balance_adjusted", "")
	for _, a := range []string{"hrm.employee_sensitive_viewed", "hrm.employee_sensitive_exported", "hrm.dependents_viewed", "hrm.dependent_saved", "hrm.dependent_deleted"} {
		d.Record.RestrictHistory(a, record.ViewFiles)
	}
	d.Record.Register(record.Type{
		Code: leaveType, Product: "hrm", Kind: record.Document, NumberPrefix: "NP",
		Fields: []record.Field{
			{Key: "days", Kind: record.Number, Label: "hrm.leave.days"},
			{Key: "leave_type", Kind: record.Choice, Label: "hrm.leave.type", Options: s.leaveTypeOptions},
		},
		// Attachments (a medical certificate) go with the request, asking nothing more.
		Perms: record.Perms{View: PermLeaveView, Edit: PermLeaveEdit, ViewFiles: PermLeaveView, Attach: PermLeaveEdit},
		Grant: s.leaveGrant, SubjectCreates: true,
		OnTransition: s.leaveTransition, BeforeSubmit: s.leaveBeforeSubmit, Approvers: s.leaveApprovers, Subjects: s.leaveSubjects,
		OnDelete: deleteLeave,
	})
	d.Record.Register(record.Type{
		Code: overtimeType, Product: "hrm", Kind: record.Document, NumberPrefix: "TC",
		Fields: []record.Field{
			{Key: "hours", Kind: record.Number, Label: "hrm.overtime.hours"},
			{Key: "day_kind", Kind: record.Choice, Label: "hrm.overtime.day_kind", Options: s.dayKindOptions},
		},
		// Attachments (a signed overtime sheet) go with the request, asking nothing more.
		Perms: record.Perms{View: PermOvertimeView, Edit: PermOvertimeEdit, ViewFiles: PermOvertimeView, Attach: PermOvertimeEdit},
		Grant: s.overtimeGrant, SubjectCreates: true,
		OnTransition: s.overtimeTransition, Approvers: s.overtimeApprovers, Subjects: s.overtimeSubjects, OnDelete: deleteOvertime,
	})
	d.Record.Register(record.Type{
		Code: contractType, Product: "hrm", Kind: record.Document, NumberPrefix: "HD",
		Fields: []record.Field{{Key: "contract_kind", Kind: record.Choice, Label: "hrm.contract.kind", Options: s.contractTypeOptions}},
		// Sending and cancelling show and change no terms; writing, the signed scan and the
		// print do, so they need the salary permission too. Attaching also once posted:
		// contracts are often signed after they are entered.
		Perms:        record.Perms{View: PermContractView, Edit: PermContractEdit, ViewFiles: PermContractView, Attach: PermContractEdit},
		Refine:       s.needsSalary(contractType, record.Edit, record.ViewFiles, record.Attach, record.Print),
		OnTransition: s.contractTransition, Subjects: s.contractSubjects, OnDelete: deleteContract,
	})
	// No approval fields and no module-chosen approvers: timesheet rules go by role.
	d.Record.Register(record.Type{
		Code: timesheetType, Product: "hrm", Kind: record.Document, NumberPrefix: "BC",
		Perms: record.Perms{View: PermTimesheetView, Edit: PermTimesheetEdit}, OnTransition: s.timesheetTransition, OnDelete: deleteTimesheet,
	})
	// The legal entity is the org unit; approval rules go by role. No attachments: a file on
	// a payroll would carry per-person amounts its view permission does not cover. Payslips
	// and the export show each person's pay.
	d.Record.Register(record.Type{
		Code: payrollType, Product: "hrm", Kind: record.Document, NumberPrefix: "BL",
		Perms:  record.Perms{View: PermPayrollView, Edit: PermPayrollEdit},
		Refine: s.payrollRefine, OnTransition: s.payrollTransition, BeforeSubmit: s.payrollBeforeSubmit,
		OnDelete: deletePayroll,
	})
	// Papers (the signed decision, the resignation letter) carry no amount.
	d.Record.Register(record.Type{
		Code: terminationType, Product: "hrm", Kind: record.Document, NumberPrefix: "NV",
		Fields:       []record.Field{{Key: "reason", Kind: record.Choice, Label: "hrm.termination.reason", Options: s.reasonOptions}},
		Perms:        record.Perms{View: PermTerminationView, Edit: PermTerminationEdit, ViewFiles: PermTerminationView, Attach: PermTerminationEdit},
		OnTransition: s.terminationTransition, Subjects: s.terminationSubjects, OnDelete: deleteTermination,
	})
	s.registerDataIO()
	s.registerPrints()
	return s
}

// needsSalary refines a document type so that actions also need hrm.salary.view at the
// document's org unit.
func (s *Service) needsSalary(code string, actions ...record.Action) func(context.Context, int64, record.Action) (bool, error) {
	return func(ctx context.Context, id int64, a record.Action) (bool, error) {
		if !slices.Contains(actions, a) {
			return true, nil
		}
		d, err := s.d.Record.Get(ctx, record.Ref{Type: code, ID: id})
		if err != nil {
			return false, err
		}
		return s.allowed(ctx, PermSalaryView, d.OrgUnitID)
	}
}

// payrollRefine makes payslips and the export, which show each person's pay, need
// hrm.salary.view and a computed payroll: before the first computation there is nothing to show.
func (s *Service) payrollRefine(ctx context.Context, id int64, a record.Action) (bool, error) {
	if a != record.Print && a != record.Export {
		return true, nil
	}
	p, err := store.GetPayroll.Run(ctx, id)
	if err != nil || !p.ComputedAt.Valid {
		return false, err
	}
	return s.allowed(ctx, PermSalaryView, p.LegalEntityID)
}

func (s *Service) allowed(ctx context.Context, perm string, unit int64, more ...int64) (bool, error) {
	return s.d.IAM.Allowed(ctx, "hrm", perm, unit, more...)
}

func (s *Service) require(ctx context.Context, perm string, unit int64, more ...int64) error {
	return s.d.IAM.Require(ctx, "hrm", perm, unit, more...)
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
