package hrm

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

const leaveBalanceImport = "hrm.leave_balance"

func (s *Service) registerDataIO() {
	s.d.DataIO.RegisterImport(dataio.Import{Kind: timesheetType, Product: "hrm", Header: s.timesheetHeader, Run: s.importTimesheet})
	s.d.DataIO.RegisterExport(dataio.Export{Kind: timesheetType, Product: "hrm", Run: s.exportTimesheet,
		Check: func(ctx context.Context, params json.RawMessage) error {
			_, err := s.sheetTimesheet(ctx, params, record.Export)
			return err
		}})
	s.d.DataIO.RegisterExport(dataio.Export{Kind: payrollType, Product: "hrm", Run: s.exportPayroll,
		Check: func(ctx context.Context, params json.RawMessage) error {
			_, err := s.exportablePayroll(ctx, params)
			return err
		}})
	s.d.DataIO.RegisterImport(dataio.Import{Kind: leaveBalanceImport, Product: "hrm", Header: s.leaveBalanceHeader, Run: s.importLeaveBalances})
}

func (s *Service) locale(ctx context.Context) (string, error) {
	me, err := s.d.IAM.Me(ctx)
	return me.Locale, err
}

// sheetTimesheet reads the params of a timesheet import or export. The requester named
// the timesheet, so one they may no longer view is a permission error, not a missing one.
func (s *Service) sheetTimesheet(ctx context.Context, params json.RawMessage, action record.Action) (store.GetTimesheetRow, error) {
	var p TimesheetParams
	if err := json.Unmarshal(params, &p); err != nil || p.TimesheetID == 0 {
		return store.GetTimesheetRow{}, dataio.ErrInvalidParams
	}
	t, err := s.visibleTimesheet(ctx, p.TimesheetID)
	if errors.Is(err, platform.ErrNotFound) {
		return t, platform.ErrForbidden
	}
	if err != nil {
		return t, err
	}
	if ok, err := s.d.Record.Can(ctx, timesheetType, t.ID, action); err != nil || !ok {
		return t, platform.OrErr(err, platform.ErrForbidden)
	}
	return t, nil
}

func (s *Service) timesheetHeader(ctx context.Context, params json.RawMessage) ([]string, error) {
	t, err := s.sheetTimesheet(ctx, params, record.View)
	if err != nil {
		return nil, err
	}
	return s.timesheetColumns(ctx, t)
}

// timesheetColumns is Mã NV | Họ tên | 01/03 … 31/03: dated, so a file of another month never fits.
func (s *Service) timesheetColumns(ctx context.Context, t store.GetTimesheetRow) ([]string, error) {
	loc, err := s.locale(ctx)
	if err != nil {
		return nil, err
	}
	h := []string{platform.Translate(loc, "hrm.import.header.code", nil), platform.Translate(loc, "hrm.import.header.name", nil)}
	for _, date := range periodDays(t) {
		h = append(h, date.Format("02/01"))
	}
	return h, nil
}

// periodDays lists every day of the timesheet's period.
func periodDays(t store.GetTimesheetRow) []time.Time {
	var out []time.Time
	for d := t.PeriodStart.Time; !d.After(t.PeriodEnd.Time); d = d.AddDate(0, 0, 1) {
		out = append(out, d)
	}
	return out
}

// exportTimesheet writes the grid in the import's layout, so the file is also the form to fill in.
func (s *Service) exportTimesheet(ctx context.Context, params json.RawMessage) (dataio.Sheet, error) {
	t, err := s.sheetTimesheet(ctx, params, record.Export)
	if err != nil {
		return dataio.Sheet{}, err
	}
	header, err := s.timesheetColumns(ctx, t)
	if err != nil {
		return dataio.Sheet{}, err
	}
	d, err := s.d.Record.Get(ctx, record.Ref{Type: timesheetType, ID: t.ID})
	if err != nil {
		return dataio.Sheet{}, err
	}
	emps, err := s.timesheetEmployees(ctx, t)
	if err != nil {
		return dataio.Sheet{}, err
	}
	lines, err := timesheetLines(ctx, t.ID)
	if err != nil {
		return dataio.Sheet{}, err
	}
	days := map[TimesheetLine]decimal.Decimal{}
	for _, l := range lines {
		days[TimesheetLine{EmployeeID: l.EmployeeID, Date: l.Date}] = decimal.RequireFromString(l.Days)
	}
	rows := make([][]any, len(emps))
	for i, e := range emps {
		row := make([]any, len(header))
		row[0], row[1] = e.Code, e.FullName
		for c, date := range periodDays(t) {
			if v, ok := days[TimesheetLine{EmployeeID: e.ID, Date: date.Format(time.DateOnly)}]; ok {
				row[2+c] = v
			}
		}
		rows[i] = row
	}
	return dataio.Sheet{Name: d.Number, Header: header, Rows: rows}, nil
}

// importTimesheet replaces every line of a draft timesheet with the file's, at its current version.
func (s *Service) importTimesheet(ctx context.Context, params json.RawMessage, rows [][]string) ([]dataio.RowError, error) {
	t, err := s.sheetTimesheet(ctx, params, record.Edit)
	if err != nil {
		return nil, err
	}
	d, err := s.d.Record.Get(ctx, record.Ref{Type: timesheetType, ID: t.ID})
	if err != nil {
		return nil, err
	}
	emps, err := s.timesheetEmployees(ctx, t)
	if err != nil {
		return nil, err
	}
	grid := map[string]TimesheetEmployee{}
	for _, e := range emps {
		grid[strings.ToLower(e.Code)] = e
	}
	dates := periodDays(t)
	var errs []dataio.RowError
	var lines []TimesheetLine
	seen := map[int64]bool{}
	for i, row := range rows {
		if row == nil {
			continue
		}
		code := strings.TrimSpace(row[0])
		e, ok := grid[strings.ToLower(code)]
		switch {
		case code == "":
			errs = append(errs, dataio.RowError{Row: i, Code: "code_required"})
			continue
		case !ok:
			// Unknown and elsewhere read the same, so the file cannot probe other units.
			errs = append(errs, dataio.RowError{Row: i, Code: "employee_not_in_unit", Params: map[string]any{"code": code}})
			continue
		case seen[e.ID]:
			errs = append(errs, dataio.RowError{Row: i, Code: "duplicate_employee", Params: map[string]any{"code": code}})
			continue
		}
		seen[e.ID] = true
		for c, cell := range row[2:] {
			v, ok := cellNumber(cell)
			if ok && v.IsZero() {
				continue
			}
			date := dates[c].Format(time.DateOnly)
			if code := lineProblem(e, date, v, ok); code != "" {
				errs = append(errs, dataio.RowError{Row: i, Code: code, Params: map[string]any{"day": dates[c].Day(), "value": cell}})
				continue
			}
			lines = append(lines, TimesheetLine{EmployeeID: e.ID, Date: date, Days: v.String()})
		}
	}
	if len(errs) > 0 {
		return errs, nil
	}
	return nil, s.SaveTimesheet(ctx, t.ID, TimesheetUpdate{Version: d.Version, Month: t.PeriodStart.Time.Format("2006-01"), Lines: lines})
}

func (s *Service) leaveBalanceHeader(ctx context.Context, _ json.RawMessage) ([]string, error) {
	loc, err := s.locale(ctx)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, k := range []string{"code", "year", "days", "reason"} {
		out = append(out, platform.Translate(loc, "hrm.import.header."+k, nil))
	}
	return out, nil
}

// importLeaveBalances adds each row's days to an employee's leave balance of a year,
// with a reason; the file may not leave any balance below zero.
func (s *Service) importLeaveBalances(ctx context.Context, _ json.RawMessage, rows [][]string) ([]dataio.RowError, error) {
	sc, err := s.d.IAM.Scope(ctx, "hrm", PermBalanceAdjust)
	if err != nil {
		return nil, err
	}
	if !sc.Any() {
		return nil, platform.ErrForbidden
	}
	known, err := s.knownCodes(ctx, rows)
	if err != nil {
		return nil, err
	}
	type adjustment struct {
		row      int
		employee int64
		code     string
		year     int32
		in       BalanceAdjustment
	}
	type key struct {
		employee int64
		year     int32
	}
	var errs []dataio.RowError
	var adjs []adjustment
	sums, last := map[key]decimal.Decimal{}, map[key]adjustment{}
	for i, row := range rows {
		if row == nil {
			continue
		}
		code := strings.TrimSpace(row[0])
		e, ok := known[strings.ToLower(code)]
		// Outside the actor's scope an employee does not exist, as everywhere else.
		if !ok || !sc.Has(e.OrgUnitID) {
			errs = append(errs, dataio.RowError{Row: i, Code: "unknown_employee", Params: map[string]any{"code": code}})
			continue
		}
		year, err := strconv.Atoi(strings.TrimSpace(row[1]))
		if err != nil || year < 2000 || year > 2100 {
			errs = append(errs, dataio.RowError{Row: i, Code: "invalid_year", Params: map[string]any{"value": row[1]}})
			continue
		}
		days, ok := cellNumber(row[2])
		if !ok || !days.Mul(decimal.NewFromInt(2)).IsInteger() || days.Abs().GreaterThan(decimal.NewFromInt(999)) {
			errs = append(errs, dataio.RowError{Row: i, Code: "invalid_leave_days", Params: map[string]any{"value": row[2]}})
			continue
		}
		reason := strings.TrimSpace(row[3])
		if reason == "" {
			errs = append(errs, dataio.RowError{Row: i, Code: "reason_required"})
			continue
		}
		a := adjustment{row: i, employee: e.ID, code: e.Code, year: int32(year), in: BalanceAdjustment{Delta: days.String(), Reason: reason}}
		adjs = append(adjs, a)
		k := key{e.ID, a.year}
		sums[k], last[k] = sums[k].Add(days), a
	}
	q := store.New(platform.DBFrom(ctx))
	for k, sum := range sums {
		balance, err := q.LeaveBalanceDays(ctx, store.LeaveBalanceDaysParams{EmployeeID: k.employee, Year: k.year})
		if err != nil {
			return nil, err
		}
		if decimal.RequireFromString(balance).Add(sum).IsNegative() {
			a := last[k]
			errs = append(errs, dataio.RowError{Row: a.row, Code: "balance_negative", Params: map[string]any{"code": a.code, "year": a.year}})
		}
	}
	if len(errs) > 0 {
		return errs, nil
	}
	// Additions first: the file's end result is checked above, and no balance may dip below zero on the way.
	slices.SortStableFunc(adjs, func(a, b adjustment) int {
		return cmp.Compare(decimal.RequireFromString(b.in.Delta).Sign(), decimal.RequireFromString(a.in.Delta).Sign())
	})
	for _, a := range adjs {
		err := s.AdjustLeaveBalance(ctx, a.employee, a.year, a.in)
		// A leave approved since the balances were read above: still the row's fault, reported as such.
		if errors.Is(err, ErrNegativeBalance) {
			return []dataio.RowError{{Row: a.row, Code: "balance_negative", Params: map[string]any{"code": a.code, "year": a.year}}}, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// knownCodes looks up the employees named in the first column, by lower-cased code.
func (s *Service) knownCodes(ctx context.Context, rows [][]string) (map[string]store.EmployeesByCodesRow, error) {
	codes := make([]string, 0, len(rows))
	for _, r := range rows {
		codes = append(codes, strings.ToLower(strings.TrimSpace(r[0])))
	}
	found, err := store.New(platform.DBFrom(ctx)).EmployeesByCodes(ctx, codes)
	out := make(map[string]store.EmployeesByCodesRow, len(found))
	for _, e := range found {
		out[strings.ToLower(e.Code)] = e
	}
	return out, err
}

// cellNumber reads a number cell; blank is zero, and a decimal comma is accepted.
func cellNumber(cell string) (decimal.Decimal, bool) {
	cell = strings.ReplaceAll(strings.TrimSpace(cell), ",", ".")
	if cell == "" {
		return decimal.Zero, true
	}
	return record.ParseNumber(cell)
}
