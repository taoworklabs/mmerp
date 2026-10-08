package hrm

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// canTimesheet goes by the timesheet's org unit.
func (s *Service) canTimesheet(ctx context.Context, id int64, action record.Action) (bool, error) {
	t, err := store.New(platform.DBFrom(ctx)).GetTimesheet(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	switch action {
	case record.View, record.Export:
		return s.allowed(ctx, PermTimesheetView, t.OrgUnitID)
	case record.Edit, record.Post, record.Cancel:
		return s.allowed(ctx, PermTimesheetEdit, t.OrgUnitID)
	}
	return false, nil
}

// timesheetTransition keeps posting and cancelling out of closed payroll periods; a
// cancelled timesheet frees its unit and period for a new one.
func (s *Service) timesheetTransition(ctx context.Context, d record.Doc, from record.Status) error {
	if _, ok := payrollEffect(d, from); !ok {
		return nil
	}
	q := store.New(platform.DBFrom(ctx))
	t, err := q.GetTimesheet(ctx, d.ID)
	if err != nil {
		return err
	}
	if err := checkPayrollPeriods(ctx, d.LegalEntityID, *platform.DatePtr(t.PeriodStart), platform.DatePtr(t.PeriodEnd)); err != nil {
		return err
	}
	if d.Status == record.Cancelled {
		return q.DeactivateTimesheet(ctx, d.ID)
	}
	return nil
}

// TimesheetActions lists what the actor may do before picking a timesheet: create.
func (s *Service) TimesheetActions(ctx context.Context) ([]string, error) {
	out := []string{}
	sc, err := s.d.IAM.Scope(ctx, "hrm", PermTimesheetEdit)
	if err != nil {
		return nil, err
	}
	if sc.Any() && platform.ProductGate(ctx, "hrm", platform.ClassWrite) == nil {
		out = append(out, "create")
	}
	return out, nil
}

// Timesheets lists the timesheets of the org units in the actor's scope, latest period first.
func (s *Service) Timesheets(ctx context.Context, f TimesheetFilter) (TimesheetList, error) {
	out := TimesheetList{Items: []TimesheetListItem{}}
	sc, err := s.d.IAM.Scope(ctx, "hrm", PermTimesheetView)
	if err != nil {
		return out, err
	}
	var start pgtype.Date
	if f.Month != "" {
		if start, _, err = period(f.Month); err != nil {
			return out, err
		}
	}
	rows, err := store.New(platform.DBFrom(ctx)).ListTimesheets(ctx, store.ListTimesheetsParams{
		AllUnits: sc.All, Units: sc.Units, Status: f.Status, OrgUnitID: pgtype.Int8{Int64: f.OrgUnitID, Valid: f.OrgUnitID != 0},
		PeriodStart: start, Lim: int32(f.PageSize), Off: int32((f.Page - 1) * f.PageSize),
	})
	for _, r := range rows {
		out.Total = r.Total
		out.Items = append(out.Items, TimesheetListItem{
			ID: r.ID, Number: r.Number, Status: r.Status, OrgUnitID: r.OrgUnitID, OrgUnitName: r.OrgUnitName,
			PeriodStart: *platform.DatePtr(r.PeriodStart), PeriodEnd: *platform.DatePtr(r.PeriodEnd),
		})
	}
	return out, err
}

// Timesheet returns a timesheet with its grid; one the actor may not view does not exist.
func (s *Service) Timesheet(ctx context.Context, id int64) (Timesheet, error) {
	t, err := s.visibleTimesheet(ctx, id)
	if err != nil {
		return Timesheet{}, err
	}
	ref := record.Ref{Type: timesheetType, ID: id}
	d, err := s.d.Record.Get(ctx, ref)
	if err != nil {
		return Timesheet{}, err
	}
	actions, err := s.d.Record.DocumentActions(ctx, d)
	if err != nil {
		return Timesheet{}, err
	}
	if ok, err := s.d.Record.Can(ctx, timesheetType, id, record.Export); err != nil {
		return Timesheet{}, err
	} else if ok {
		actions = append(actions, "export")
	}
	// Importing replaces the lines of a draft: whoever may edit it may import into it.
	if slices.Contains(actions, "edit") {
		actions = append(actions, "import")
	}
	emps, err := s.timesheetEmployees(ctx, t)
	if err != nil {
		return Timesheet{}, err
	}
	lines, err := timesheetLines(ctx, id)
	if err != nil {
		return Timesheet{}, err
	}
	start, end := *platform.DatePtr(t.PeriodStart), *platform.DatePtr(t.PeriodEnd)
	cal, err := workCalendar(ctx, d.LegalEntityID, start, end)
	return Timesheet{
		ID: id, Number: d.Number, Status: string(d.Status), Version: d.Version, OrgUnitID: t.OrgUnitID, OrgUnitName: t.OrgUnitName,
		PeriodStart: start, PeriodEnd: end, Employees: emps, Lines: lines,
		OffDays: cal.offDays(t.PeriodStart.Time, t.PeriodEnd.Time), AllowedActions: actions,
	}, err
}

// CreateTimesheet adds an empty draft for an org unit and month.
func (s *Service) CreateTimesheet(ctx context.Context, in NewTimesheet) (int64, error) {
	start, end, err := period(in.Month)
	if err != nil {
		return 0, err
	}
	if err := s.require(ctx, PermTimesheetEdit, in.OrgUnitID); err != nil {
		return 0, err
	}
	var id int64
	err = platform.InTx(ctx, func(ctx context.Context) error {
		d, err := s.d.Record.Create(ctx, timesheetType, record.Header{Date: *platform.DatePtr(end), OrgUnitID: in.OrgUnitID})
		if err != nil {
			return err
		}
		id = d.ID
		err = store.New(platform.DBFrom(ctx)).CreateTimesheet(ctx, store.CreateTimesheetParams{ID: id, OrgUnitID: in.OrgUnitID, PeriodStart: start, PeriodEnd: end})
		if isCheck(err, "timesheets_org_unit_period_key") {
			return ErrTimesheetExists
		}
		return err
	})
	return id, err
}

// SaveTimesheet replaces a draft's period and all its lines, at the version the actor saw.
func (s *Service) SaveTimesheet(ctx context.Context, id int64, in TimesheetUpdate) error {
	start, end, err := period(in.Month)
	if err != nil {
		return err
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		t, err := s.visibleTimesheet(ctx, id)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, record.Ref{Type: timesheetType, ID: id}, in.Version,
			record.Header{Date: *platform.DatePtr(end), OrgUnitID: t.OrgUnitID}); err != nil {
			return err
		}
		t.PeriodStart, t.PeriodEnd = start, end
		emps, err := s.timesheetEmployees(ctx, t)
		if err != nil {
			return err
		}
		p := store.InsertTimesheetLinesParams{TimesheetID: id}
		if p.EmployeeIds, p.Dates, p.Days, err = checkLines(t, emps, in.Lines); err != nil {
			return err
		}
		q := store.New(platform.DBFrom(ctx))
		err = q.UpdateTimesheetPeriod(ctx, store.UpdateTimesheetPeriodParams{ID: id, PeriodStart: start, PeriodEnd: end})
		if isCheck(err, "timesheets_org_unit_period_key") {
			return ErrTimesheetExists
		}
		if err != nil {
			return err
		}
		if err := q.DeleteTimesheetLines(ctx, id); err != nil {
			return err
		}
		if err := q.InsertTimesheetLines(ctx, p); err != nil {
			return err
		}
		return s.d.Audit.RecordFor(ctx, "hrm.timesheet_saved", audit.Ref{Type: timesheetType, ID: id},
			map[string]any{"month": in.Month, "lines": len(in.Lines)})
	})
}

// DeleteTimesheet removes a draft at the version the actor saw; its lines go with it.
func (s *Service) DeleteTimesheet(ctx context.Context, id int64, version int32) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.visibleTimesheet(ctx, id); err != nil {
			return err
		}
		if err := s.d.Record.Delete(ctx, record.Ref{Type: timesheetType, ID: id}, version); err != nil {
			return err
		}
		return store.New(platform.DBFrom(ctx)).DeleteTimesheet(ctx, id)
	})
}

func (s *Service) visibleTimesheet(ctx context.Context, id int64) (store.GetTimesheetRow, error) {
	if ok, err := s.d.Record.Can(ctx, timesheetType, id, record.View); err != nil || !ok {
		return store.GetTimesheetRow{}, platform.OrErr(err, platform.ErrNotFound)
	}
	return store.New(platform.DBFrom(ctx)).GetTimesheet(ctx, id)
}

func (s *Service) timesheetEmployees(ctx context.Context, t store.GetTimesheetRow) ([]TimesheetEmployee, error) {
	rows, err := store.New(platform.DBFrom(ctx)).TimesheetEmployees(ctx, store.TimesheetEmployeesParams{
		OrgUnitID: t.OrgUnitID, PeriodStart: t.PeriodStart, PeriodEnd: t.PeriodEnd, TimesheetID: t.ID,
	})
	out := make([]TimesheetEmployee, len(rows))
	for i, r := range rows {
		out[i] = TimesheetEmployee{ID: r.ID, Code: r.Code, FullName: r.FullName, HireDate: *platform.DatePtr(r.HireDate), TerminationDate: platform.DatePtr(r.TerminationDate)}
	}
	return out, err
}

func timesheetLines(ctx context.Context, id int64) ([]TimesheetLine, error) {
	rows, err := store.New(platform.DBFrom(ctx)).TimesheetLines(ctx, id)
	out := make([]TimesheetLine, len(rows))
	for i, r := range rows {
		out[i] = TimesheetLine{EmployeeID: r.EmployeeID, Date: *platform.DatePtr(r.Date), Days: plain(r.Days)}
	}
	return out, err
}

var half, one = decimal.RequireFromString("0.5"), decimal.NewFromInt(1)

// lineProblem names what is wrong with e working days (ok: a number was read) on date
// of the period, or "" when nothing is; the API and the import share it.
func lineProblem(e TimesheetEmployee, date string, days decimal.Decimal, ok bool) string {
	switch {
	case !ok || (!days.Equal(half) && !days.Equal(one)):
		return "invalid_timesheet_days"
	case date < e.HireDate || (e.TerminationDate != nil && date > *e.TerminationDate):
		return "timesheet_date_outside_employment"
	}
	return ""
}

// checkLines keeps each line inside the period, on an employee of the grid while
// employed, at a half or a full day, once per employee and day.
func checkLines(t store.GetTimesheetRow, emps []TimesheetEmployee, lines []TimesheetLine) ([]int64, []pgtype.Date, []string, error) {
	byID := make(map[int64]TimesheetEmployee, len(emps))
	for _, e := range emps {
		byID[e.ID] = e
	}
	start, end := *platform.DatePtr(t.PeriodStart), *platform.DatePtr(t.PeriodEnd)
	seen := map[TimesheetLine]bool{}
	ids, dates, days := make([]int64, len(lines)), make([]pgtype.Date, len(lines)), make([]string, len(lines))
	for i, l := range lines {
		date, err := time.Parse(time.DateOnly, l.Date)
		if err != nil || l.Date < start || l.Date > end {
			return nil, nil, nil, errTimesheetLine("timesheet_date_outside_period", l)
		}
		e, ok := byID[l.EmployeeID]
		if !ok {
			return nil, nil, nil, errTimesheetLine("timesheet_employee_not_in_unit", l)
		}
		d, ok := record.ParseNumber(l.Days)
		if code := lineProblem(e, l.Date, d, ok); code != "" {
			return nil, nil, nil, errTimesheetLine(code, l)
		}
		key := TimesheetLine{EmployeeID: l.EmployeeID, Date: l.Date}
		if seen[key] {
			return nil, nil, nil, errTimesheetLine("timesheet_line_duplicate", l)
		}
		seen[key] = true
		ids[i], dates[i], days[i] = l.EmployeeID, pgtype.Date{Time: date, Valid: true}, d.String()
	}
	return ids, dates, days, nil
}

// period turns YYYY-MM into its first and last day: payroll periods are calendar months.
func period(month string) (start, end pgtype.Date, err error) {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return start, end, &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_request", Params: map[string]any{"fields": []string{"month"}}}
	}
	return pgtype.Date{Time: t, Valid: true}, pgtype.Date{Time: t.AddDate(0, 1, -1), Valid: true}, nil
}
