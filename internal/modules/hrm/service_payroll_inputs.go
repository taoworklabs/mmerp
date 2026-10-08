package hrm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// payrollSource is a posted document a payroll was computed from, at its version.
type payrollSource struct {
	DocType string `json:"doc_type"`
	DocID   int64  `json:"doc_id"`
	Version int32  `json:"version"`
}

type basisEmployee struct {
	ID        int64   `json:"id"`
	Code      string  `json:"-"`
	Name      string  `json:"-"`
	OrgUnitID int64   `json:"org_unit_id"`
	Hire      string  `json:"hire"`
	Leave     *string `json:"leave"`
	// Digest of the dependents as stored: any change to them changes it, without decrypting.
	Dependents []byte `json:"dependents"`
	// Unused leave of an employee leaving during the period.
	LeaveBalance string `json:"leave_balance"`
}

type basisLeave struct {
	EmployeeID int64           `json:"employee_id"`
	Start      string          `json:"start"`
	End        string          `json:"end"`
	Days       decimal.Decimal `json:"days"`
	Paid       bool            `json:"paid"`
}

type basisOvertime struct {
	EmployeeID int64 `json:"employee_id"`
	overtimeDay
}

// payrollBasis is every non-sensitive input of a payroll of one legal entity and period. Its
// hash tells whether the sources changed, and it is read without decrypting any salary.
type payrollBasis struct {
	LegalEntity int64                 `json:"legal_entity"`
	Start       string                `json:"start"`
	End         string                `json:"end"`
	Rounding    string                `json:"rounding"`
	Region      string                `json:"region"`
	Params      map[string]LegalParam `json:"params"`
	Calendar    calendar              `json:"calendar"`
	Employees   []basisEmployee       `json:"employees"`
	Sources     []payrollSource       `json:"sources"`
	Worked      map[string]string     `json:"worked"` // "<employee>:<date>" → days
	Leaves      []basisLeave          `json:"leaves"`
	Overtime    []basisOvertime       `json:"overtime"`
}

func (b payrollBasis) hash() ([]byte, error) {
	raw, err := json.Marshal(b)
	sum := sha256.Sum256(raw)
	return sum[:], err
}

func dayOf(s string) time.Time {
	t, _ := time.Parse(time.DateOnly, s)
	return t
}

// readBasis gathers a payroll's non-sensitive inputs; payrollID adds the employees of its
// adjustments (0 for none). Every org unit of the employees needs a posted timesheet.
func (s *Service) readBasis(ctx context.Context, legalEntity, payrollID int64, start, end time.Time) (payrollBasis, error) {
	q := store.New(platform.DBFrom(ctx))
	ps, pe := start.Format(time.DateOnly), end.Format(time.DateOnly)
	prevStart := start.AddDate(0, -1, 0)
	yearStart := time.Date(start.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	b := payrollBasis{LegalEntity: legalEntity, Start: ps, End: pe, Worked: map[string]string{}}
	var err error
	if b.Rounding, err = s.d.Setting.GetFor(ctx, legalEntity, setting.Rounding); err != nil {
		return b, err
	}
	if b.Region, err = s.d.Setting.GetFor(ctx, legalEntity, SettingWageRegion); err != nil {
		return b, err
	}
	if b.Params, err = legalParamsAt(ctx, pe); err != nil {
		return b, err
	}
	if b.Calendar, err = workCalendar(ctx, legalEntity, prevStart.Format(time.DateOnly), pe); err != nil {
		return b, err
	}
	pgStart, pgEnd, pgPrev, pgYear := platform.NullDate(&ps), platform.NullDate(&pe), pgDate(prevStart), pgDate(yearStart)

	emps, err := q.PayrollEmployees(ctx, store.PayrollEmployeesParams{PeriodStart: pgStart, PeriodEnd: pgEnd, LegalEntityID: legalEntity, PayrollID: payrollID})
	if err != nil {
		return b, err
	}
	ids := make([]int64, len(emps))
	units := []int64{}
	for i, e := range emps {
		ids[i] = e.ID
		b.Employees = append(b.Employees, basisEmployee{ID: e.ID, Code: e.Code, Name: e.FullName, OrgUnitID: e.OrgUnitID,
			Hire: *platform.DatePtr(e.HireDate), Leave: platform.DatePtr(e.TerminationDate)})
		if !slices.Contains(units, e.OrgUnitID) {
			units = append(units, e.OrgUnitID)
		}
	}
	if err := missingTimesheets(ctx, units, pgStart); err != nil {
		return b, err
	}

	deps, err := q.PayrollDependents(ctx, ids)
	if err != nil {
		return b, err
	}
	digests := map[int64][]byte{}
	for _, d := range deps {
		sum := sha256.Sum256(append(digests[d.EmployeeID], d.Data...))
		digests[d.EmployeeID] = sum[:]
	}
	balances, err := q.PayrollLeaveBalances(ctx, store.PayrollLeaveBalancesParams{Ids: ids, Year: int32(start.Year())})
	if err != nil {
		return b, err
	}
	for i := range b.Employees {
		e := &b.Employees[i]
		e.Dependents = digests[e.ID]
		if e.Leave != nil && *e.Leave <= pe {
			for _, bal := range balances {
				if bal.EmployeeID == e.ID {
					e.LeaveBalance = bal.Days
				}
			}
		}
	}

	docs, err := q.PayrollSourceDocs(ctx, store.PayrollSourceDocsParams{LegalEntityID: legalEntity, PeriodStart: pgStart, PeriodEnd: pgEnd, PrevStart: pgPrev, YearStart: pgYear})
	if err != nil {
		return b, err
	}
	for _, d := range docs {
		b.Sources = append(b.Sources, payrollSource{DocType: d.DocType, DocID: d.ID, Version: d.Version})
	}
	worked, err := q.PayrollWorkedDays(ctx, store.PayrollWorkedDaysParams{PeriodStart: pgStart, Ids: ids})
	if err != nil {
		return b, err
	}
	for _, w := range worked {
		b.Worked[fmt.Sprintf("%d:%s", w.EmployeeID, *platform.DatePtr(w.Date))] = w.Days
	}
	leaves, err := q.PayrollLeaves(ctx, store.PayrollLeavesParams{Ids: ids, PeriodStart: pgStart, PeriodEnd: pgEnd})
	if err != nil {
		return b, err
	}
	for _, l := range leaves {
		b.Leaves = append(b.Leaves, basisLeave{EmployeeID: l.EmployeeID, Start: *platform.DatePtr(l.StartDate), End: *platform.DatePtr(l.EndDate),
			Days: decimal.RequireFromString(l.Days), Paid: l.Paid})
	}
	ots, err := q.PayrollOvertime(ctx, store.PayrollOvertimeParams{Ids: ids, YearStart: pgYear, PeriodEnd: pgEnd})
	for _, o := range ots {
		b.Overtime = append(b.Overtime, basisOvertime{EmployeeID: o.EmployeeID, overtimeDay: overtimeDay{Date: *platform.DatePtr(o.Date), DayKind: o.DayKind,
			DayHours: decimal.RequireFromString(o.DayHours), NightHours: decimal.RequireFromString(o.NightHours)}})
	}
	return b, err
}

// missingTimesheets refuses a payroll while an org unit of its employees has no posted timesheet.
func missingTimesheets(ctx context.Context, units []int64, start pgtype.Date) error {
	q := store.New(platform.DBFrom(ctx))
	posted, err := q.PostedTimesheetUnits(ctx, start)
	if err != nil {
		return err
	}
	var missing []int64
	for _, u := range units {
		if !slices.Contains(posted, u) {
			missing = append(missing, u)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	rows, err := q.OrgUnitNames(ctx, missing)
	if err != nil {
		return err
	}
	list := make([]map[string]any, len(rows))
	for i, r := range rows {
		list[i] = map[string]any{"id": r.ID, "name": r.Name}
	}
	return &platform.Error{Status: http.StatusConflict, Code: "payroll_timesheets_missing", Params: map[string]any{"org_units": list}}
}

func pgDate(t time.Time) pgtype.Date { return pgtype.Date{Time: t, Valid: true} }

// payInputs decrypts what the basis cannot hold (contract terms, dependents, adjustments)
// and builds each employee's input, in the basis's employee order. Each contract whose
// terms are read is audited.
func (s *Service) payInputs(ctx context.Context, b payrollBasis, payrollID int64) ([]payInput, error) {
	q := store.New(platform.DBFrom(ctx))
	start, end := dayOf(b.Start), dayOf(b.End)
	prevStart, prevEnd := start.AddDate(0, -1, 0), start.AddDate(0, 0, -1)
	ids := make([]int64, len(b.Employees))
	for i, e := range b.Employees {
		ids[i] = e.ID
	}

	rows, err := q.PayrollContracts(ctx, store.PayrollContractsParams{LegalEntityID: b.LegalEntity, Ids: ids, PeriodEnd: platform.NullDate(&b.End), PrevStart: platform.NullDate(new(prevStart.Format(time.DateOnly)))})
	if err != nil {
		return nil, err
	}
	contracts := map[int64][]payContract{}
	for _, r := range rows {
		if err := s.d.Audit.RecordFor(ctx, "hrm.contract_terms_viewed", audit.Ref{Type: contractType, ID: r.ID}, nil); err != nil {
			return nil, err
		}
		t, err := openTerms(ctx, r.Terms)
		if err != nil {
			return nil, err
		}
		contracts[r.EmployeeID] = append(contracts[r.EmployeeID], payContract{ID: r.ID, Original: !r.ParentID.Valid, Parent: r.ParentID.Int64,
			Start: *platform.DatePtr(r.StartDate), End: platform.DatePtr(r.EndDate), Terms: t})
	}

	deps, err := q.PayrollDependents(ctx, ids)
	if err != nil {
		return nil, err
	}
	month := b.Start[:7]
	dependents := map[int64]int{}
	for _, d := range deps {
		dep, err := openDependent(ctx, d.Data)
		if err != nil {
			return nil, err
		}
		if dep.DeductionFrom != nil && *dep.DeductionFrom <= month && (dep.DeductionTo == nil || month <= *dep.DeductionTo) {
			dependents[d.EmployeeID]++
		}
	}
	adjustments, err := s.adjustments(ctx, payrollID)
	if err != nil {
		return nil, err
	}

	cal := b.Calendar
	std := cal.standardDays(start, end)
	out := make([]payInput, len(b.Employees))
	for i, e := range b.Employees {
		x := payInput{StandardDays: std, Dependents: dependents[e.ID], Overtime: []overtimeDay{}}
		for _, a := range adjustments {
			if a.EmployeeID == e.ID {
				x.Adjustment += a.Amount
			}
		}
		cs := contracts[e.ID]
		leaveDays := leaveByDay(b, e.ID)
		from, to := start, end
		if h := dayOf(e.Hire); h.After(from) {
			from = h
		}
		if e.Leave != nil && dayOf(*e.Leave).Before(to) {
			to = dayOf(*e.Leave)
		}
		paidTotal := decimal.Zero
		for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
			c := termsAt(cs, d.Format(time.DateOnly))
			if c == nil {
				continue
			}
			day := d.Format(time.DateOnly)
			paid := decimal.Zero
			switch {
			case cal.weeklyOff(d):
			case cal.holiday(d):
				paid = one
			default:
				w, _ := record.ParseNumber(b.Worked[fmt.Sprintf("%d:%s", e.ID, day)])
				paid = decimal.Min(one, w.Add(leaveDays[day]))
			}
			if n := len(x.Segments); n > 0 && x.Segments[n-1].contract == c.ID {
				x.Segments[n-1].To, x.Segments[n-1].PaidDays = day, x.Segments[n-1].PaidDays.Add(paid)
			} else {
				x.Segments = append(x.Segments, paySegment{From: day, To: day, Terms: c.Terms, PaidDays: paid, contract: c.ID})
			}
			paidTotal = paidTotal.Add(paid)
		}
		// A month with more working days than the standard (capped at 26) pays at most the
		// standard: the excess comes off the last segments.
		for excess := paidTotal.Sub(decimal.NewFromInt(int64(std))); excess.IsPositive(); {
			s := &x.Segments[len(x.Segments)-1]
			for i := len(x.Segments) - 1; i >= 0 && !s.PaidDays.IsPositive(); i-- {
				s = &x.Segments[i]
			}
			take := decimal.Min(excess, s.PaidDays)
			s.PaidDays, excess, paidTotal = s.PaidDays.Sub(take), excess.Sub(take), paidTotal.Sub(take)
		}
		x.UnpaidDays = decimal.NewFromInt(int64(std)).Sub(paidTotal)
		if c := termsAt(cs, to.Format(time.DateOnly)); c != nil {
			x.InsuredTerms = c.Terms
		} else if n := len(x.Segments); n > 0 {
			x.InsuredTerms = x.Segments[n-1].Terms
		}
		for _, o := range b.Overtime {
			if o.EmployeeID != e.ID {
				continue
			}
			if o.Date < b.Start {
				x.OvertimeHoursBefore = x.OvertimeHoursBefore.Add(o.DayHours).Add(o.NightHours)
			} else {
				x.Overtime = append(x.Overtime, o.overtimeDay)
			}
		}
		if days, ok := record.ParseNumber(e.LeaveBalance); ok && days.IsPositive() {
			c := termsAt(cs, prevEnd.Format(time.DateOnly))
			if c == nil {
				c = termsAt(cs, to.Format(time.DateOnly))
			}
			if c != nil {
				x.UnusedLeave = &unusedLeave{Salary: c.Terms.Salary, StandardDays: cal.standardDays(prevStart, prevEnd), Days: days}
			}
		}
		out[i] = x
	}
	return out, nil
}

// payContract is a posted contract or appendix with its terms; an appendix runs to its original's end.
type payContract struct {
	ID       int64
	Original bool
	Parent   int64
	Start    string
	End      *string
	Terms    ContractTerms
}

// termsAt is the contract whose terms are in force at day: the latest posted appendix of the
// original covering it, or the original itself.
func termsAt(cs []payContract, day string) *payContract {
	var orig *payContract
	for i, c := range cs {
		if c.Original && c.Start <= day && (c.End == nil || day <= *c.End) {
			orig = &cs[i]
		}
	}
	if orig == nil {
		return nil
	}
	out := orig
	for i, c := range cs {
		if !c.Original && c.Parent == orig.ID && c.Start <= day && c.Start >= out.Start {
			out = &cs[i]
		}
	}
	return out
}

// leaveByDay spreads each paid leave of an employee over its working days in date order,
// a day at most per day, and returns the paid leave days of each date.
func leaveByDay(b payrollBasis, employee int64) map[string]decimal.Decimal {
	out := map[string]decimal.Decimal{}
	for _, l := range b.Leaves {
		if l.EmployeeID != employee || !l.Paid {
			continue
		}
		left := l.Days
		for d := dayOf(l.Start); !d.After(dayOf(l.End)) && left.IsPositive(); d = d.AddDate(0, 0, 1) {
			if b.Calendar.weeklyOff(d) || b.Calendar.holiday(d) {
				continue
			}
			take := decimal.Min(one, left)
			out[d.Format(time.DateOnly)] = out[d.Format(time.DateOnly)].Add(take)
			left = left.Sub(take)
		}
	}
	return out
}

// payParams turns the parameters in force into what the calculation uses, with the
// minimum wage of region (1–4).
func payParams(p map[string]LegalParam, region string) (params, error) {
	num := func(k string) decimal.Decimal { d, _ := record.ParseNumber(p[k].Value); return d }
	out := params{
		BaseSalary: num("base_salary"), MinWage: num("min_wage_region_" + region),
		SIEmployee: num("si_employee"), HIEmployee: num("hi_employee"), UIEmployee: num("ui_employee"),
		SIEmployer: num("si_employer"), HIEmployer: num("hi_employer"), UIEmployer: num("ui_employer"), UnionEmployer: num("union_employer"),
		CapMultiplier: num("insurance_cap_multiplier"), Personal: num("personal_deduction"), Dependent: num("dependent_deduction"),
		OTMonth: num("ot_exempt_hours_month"), OTYear: num("ot_exempt_hours_year"), HoursPerDay: num("ot_hours_per_day"),
		SkipDays: num("insurance_skip_days"),
	}
	b, ok := parseBrackets(p["pit_brackets"].Value)
	if !ok || out.HoursPerDay.IsZero() {
		return out, fmt.Errorf("hrm: stored legal parameters are malformed")
	}
	out.Brackets = b
	return out, nil
}

// legalParamsAt reads each key's version in force at date; a missing key is an error
// naming it, since no payroll can be computed without it.
func legalParamsAt(ctx context.Context, date string) (map[string]LegalParam, error) {
	rows, err := store.New(platform.DBFrom(ctx)).LegalParamsAt(ctx, platform.NullDate(&date))
	if err != nil {
		return nil, err
	}
	out := map[string]LegalParam{}
	for _, r := range rows {
		out[r.Key] = LegalParam{Key: r.Key, EffectiveFrom: *platform.DatePtr(r.EffectiveFrom), Value: r.Value}
	}
	for _, k := range legalParamKeys {
		if _, ok := out[k]; !ok {
			return nil, &platform.Error{Status: http.StatusUnprocessableEntity, Code: "legal_param_missing", Params: map[string]any{"key": k, "date": date}}
		}
	}
	return out, nil
}
