package hrm

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// SettingLeaveAccrual is monthly (the job credits annual leave) or manual (HR grants it).
const SettingLeaveAccrual = "hrm.leave_accrual"

// SettingCarryOverMonths is how many months of the new year last year's unused days last: 0, 3, 6 or 12.
const SettingCarryOverMonths = "hrm.leave_carry_over_months"

// Annual leave of a full year under normal conditions (Labour Code 2019, Article 113.1a).
const annualLeaveDays = 12

// monthCounted: the employee is employed on at least half of the month's working days
// (Decree 145/2020, Article 66.2, without subtracting unpaid leave inside the span).
func monthCounted(cal calendar, hire string, termination *string, year, month int) bool {
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1)
	from, to := first, last
	if h := dayOf(hire); h.After(from) {
		from = h
	}
	if termination != nil && dayOf(*termination).Before(to) {
		to = dayOf(*termination)
	}
	all := cal.workingDays(first, last)
	if all == 0 || to.Before(from) {
		return false
	}
	return 2*cal.workingDays(from, to) >= all
}

// seniorityDays is one more day per five full years since hire, counted at the month's end
// (Article 114).
func seniorityDays(hire string, year, month int) int {
	h := dayOf(hire)
	end := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	years := end.Year() - h.Year()
	if end.Before(h.AddDate(years, 0, 0)) {
		years--
	}
	return max(years, 0) / 5
}

// entitlementThrough is the year's entitlement earned by the end of month: (12 + seniority)/12
// per counted month (Decree 145/2020, Article 66.1), rounded down to a half day.
func entitlementThrough(cal calendar, hire string, termination *string, year, month int) decimal.Decimal {
	// Twelfths, summed as integers so 13/12 twelve times is exactly 13.
	twelfths := 0
	for m := 1; m <= month; m++ {
		if monthCounted(cal, hire, termination, year, m) {
			twelfths += annualLeaveDays + seniorityDays(hire, year, m)
		}
	}
	return decimal.NewFromInt(int64(twelfths / 6)).Div(decimal.NewFromInt(2))
}

// monthCredit is what month adds to the year's rounded entitlement.
func monthCredit(cal calendar, hire string, termination *string, year, month int) decimal.Decimal {
	return entitlementThrough(cal, hire, termination, year, month).Sub(entitlementThrough(cal, hire, termination, year, month-1))
}

type leaveAccrualArgs struct{}

func (leaveAccrualArgs) Kind() string           { return "hrm.leave_accrual" }
func (leaveAccrualArgs) Spec() platform.JobSpec { return platform.JobSpec{System: true} }

type leaveAccrualWorker struct {
	river.WorkerDefaults[leaveAccrualArgs]
	s *Service
}

func (w *leaveAccrualWorker) Work(ctx context.Context, _ *river.Job[leaveAccrualArgs]) error {
	today, err := w.s.d.Setting.Today(ctx)
	if err != nil {
		return err
	}
	return w.s.AccrueLeave(ctx, *platform.DatePtr(today))
}

// periodic looks for months to credit every hour and at start: a server off on the 1st
// credits what it missed at its next run.
var periodic = []*river.PeriodicJob{river.NewPeriodicJob(river.PeriodicInterval(time.Hour),
	func() (river.JobArgs, *river.InsertOpts) { return leaveAccrualArgs{}, nil }, &river.PeriodicJobOpts{RunOnStart: true})}

// AccrueLeave credits every month completed before today that the ledger has not decided
// yet, each employee in its own transaction. A failing employee does not stop the others;
// the next run retries them.
func (s *Service) AccrueLeave(ctx context.Context, today string) error {
	t := dayOf(today)
	emps, err := store.AccrualEmployees.Run(ctx, pgDate(time.Date(t.Year()-1, 1, 1, 0, 0, 0, 0, time.UTC)))
	if err != nil {
		return err
	}
	cals := map[[2]int64]calendar{}
	var errs []error
	for _, e := range emps {
		if err := platform.InTx(ctx, func(ctx context.Context) error { return s.accrueEmployee(ctx, e, t, cals) }); err != nil {
			errs = append(errs, fmt.Errorf("leave accrual of employee %d: %w", e.ID, err))
		}
	}
	return errors.Join(errs...)
}

func (s *Service) accrueEmployee(ctx context.Context, e store.AccrualEmployeesRow, today time.Time, cals map[[2]int64]calendar) error {
	legalEntity, err := s.d.IAM.LegalEntityOf(ctx, e.OrgUnitID)
	if err != nil {
		return err
	}
	mode, err := s.d.Setting.GetFor(ctx, legalEntity, SettingLeaveAccrual)
	if err != nil {
		return err
	}
	monthly := mode == "monthly"
	calOf := func(year int) (calendar, error) {
		k := [2]int64{legalEntity, int64(year)}
		if c, ok := cals[k]; ok {
			return c, nil
		}
		c, err := workCalendar(ctx, legalEntity, fmt.Sprintf("%d-01-01", year), fmt.Sprintf("%d-12-31", year))
		if err == nil {
			cals[k] = c
		}
		return c, err
	}
	year := today.Year()
	// Last year is caught up in January, or later when it was already accruing, so turning
	// accrual on in the autumn never back-fills the year before.
	last, err := store.LeaveAccrualMonths.Run(ctx, store.LeaveAccrualMonthsParams{EmployeeID: e.ID, Year: int32(year - 1)})
	if err != nil {
		return err
	}
	if monthly && (today.Month() == time.January || len(last) > 0) {
		cal, err := calOf(year - 1)
		if err != nil {
			return err
		}
		if err := s.accrueMonths(ctx, e, cal, year-1, 12, last); err != nil {
			return err
		}
	}
	// A leaver's balance is final once paid out: it is neither carried past the year they
	// left in nor expired after they left.
	termination := platform.DatePtr(e.TerminationDate)
	if termination == nil || dayOf(*termination).Year() >= year {
		if err := s.carryOver(ctx, e.ID, legalEntity, year); err != nil {
			return err
		}
	}
	if monthly {
		done, err := store.LeaveAccrualMonths.Run(ctx, store.LeaveAccrualMonthsParams{EmployeeID: e.ID, Year: int32(year)})
		if err != nil {
			return err
		}
		cal, err := calOf(year)
		if err != nil {
			return err
		}
		if err := s.accrueMonths(ctx, e, cal, year, int(today.Month())-1, done); err != nil {
			return err
		}
	}
	if termination == nil || !dayOf(*termination).Before(today) {
		return s.expireCarried(ctx, e.ID, year, today)
	}
	return nil
}

// carryOver moves what is left of last year into this year's carried days, lasting to the
// end of the setting's month (0: the end of last year, so they lapse at once). Last year is
// left at 0, so a rerun moves only what reached it since.
func (s *Service) carryOver(ctx context.Context, employee, legalEntity int64, year int) error {
	prev, err := store.LockLeaveBalance.Run(ctx, store.LockLeaveBalanceParams{EmployeeID: employee, Year: int32(year - 1)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	moved := decimal.RequireFromString(prev.Days)
	if !moved.IsPositive() {
		return nil
	}
	v, err := s.d.Setting.GetFor(ctx, legalEntity, SettingCarryOverMonths)
	if err != nil {
		return err
	}
	months, err := strconv.Atoi(v)
	if err != nil {
		return err
	}
	until := time.Date(year, time.Month(months)+1, 0, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
	if err := store.SetLeaveBalance.Run(ctx, store.SetLeaveBalanceParams{EmployeeID: employee, Year: int32(year - 1), Days: "0", Carried: "0", CarriedUntil: prev.CarriedUntil}); err != nil {
		return err
	}
	if err := store.EnsureLeaveBalance.Run(ctx, store.EnsureLeaveBalanceParams{EmployeeID: employee, Year: int32(year)}); err != nil {
		return err
	}
	cur, err := store.LockLeaveBalance.Run(ctx, store.LockLeaveBalanceParams{EmployeeID: employee, Year: int32(year)})
	if err != nil {
		return err
	}
	days := decimal.RequireFromString(cur.Days).Add(moved)
	carried := decimal.RequireFromString(cur.Carried).Add(moved)
	if err := store.SetLeaveBalance.Run(ctx, store.SetLeaveBalanceParams{EmployeeID: employee, Year: int32(year),
		Days: days.String(), Carried: carried.String(), CarriedUntil: platform.NullDate(&until)}); err != nil {
		return err
	}
	return s.d.Audit.RecordFor(ctx, "hrm.leave_carried_over", audit.Ref{Type: employeeType, ID: employee},
		map[string]any{"year": year, "days": moved.String(), "until": until, "balance": days.String()})
}

// expireCarried takes the carried days still unspent out of the balance once they have lapsed.
func (s *Service) expireCarried(ctx context.Context, employee int64, year int, today time.Time) error {
	cur, err := store.LockLeaveBalance.Run(ctx, store.LockLeaveBalanceParams{EmployeeID: employee, Year: int32(year)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	carried := decimal.RequireFromString(cur.Carried)
	if !carried.IsPositive() || !cur.CarriedUntil.Valid || !today.After(cur.CarriedUntil.Time) {
		return nil
	}
	days := decimal.RequireFromString(cur.Days).Sub(carried)
	if err := store.SetLeaveBalance.Run(ctx, store.SetLeaveBalanceParams{EmployeeID: employee, Year: int32(year),
		Days: days.String(), Carried: "0", CarriedUntil: cur.CarriedUntil}); err != nil {
		return err
	}
	return s.d.Audit.RecordFor(ctx, "hrm.leave_carry_expired", audit.Ref{Type: employeeType, ID: employee},
		map[string]any{"year": year, "days": carried.String(), "balance": days.String()})
}

// accrueMonths decides months 1 to through of year that are not done, within the employment.
func (s *Service) accrueMonths(ctx context.Context, e store.AccrualEmployeesRow, cal calendar, year, through int, done []int32) error {
	hire, termination := *platform.DatePtr(e.HireDate), platform.DatePtr(e.TerminationDate)
	ym := func(d string) int { t := dayOf(d); return t.Year()*12 + int(t.Month()) }
	for m := 1; m <= through; m++ {
		at := year*12 + m
		if slices.Contains(done, int32(m)) || at < ym(hire) || termination != nil && at > ym(*termination) {
			continue
		}
		credit := monthCredit(cal, hire, termination, year, m)
		n, err := store.InsertLeaveAccrual.Run(ctx, store.InsertLeaveAccrualParams{EmployeeID: e.ID, Year: int32(year), Month: int32(m), Days: credit.String()})
		if err != nil {
			return err
		}
		if n == 0 || !credit.IsPositive() {
			continue
		}
		if err := store.EnsureLeaveBalance.Run(ctx, store.EnsureLeaveBalanceParams{EmployeeID: e.ID, Year: int32(year)}); err != nil {
			return err
		}
		after, err := store.AddLeaveBalance.Run(ctx, store.AddLeaveBalanceParams{EmployeeID: e.ID, Year: int32(year), Delta: credit.String()})
		if err != nil {
			return err
		}
		if err := s.d.Audit.RecordFor(ctx, "hrm.leave_accrued", audit.Ref{Type: employeeType, ID: e.ID},
			map[string]any{"year": year, "month": m, "days": credit.String(), "balance": platform.PlainDecimal(after)}); err != nil {
			return err
		}
	}
	return nil
}

// earnedNotCredited is the leave earned by termination that the job has not credited yet,
// the termination month at least: the job credits a month once it has ended. It shares the
// job's arithmetic, so the two never disagree.
func (s *Service) earnedNotCredited(ctx context.Context, legalEntity, employee int64, hire, termination string) (decimal.Decimal, error) {
	mode, err := s.d.Setting.GetFor(ctx, legalEntity, SettingLeaveAccrual)
	if err != nil || mode != "monthly" {
		return decimal.Zero, err
	}
	t := dayOf(termination)
	cal, err := workCalendar(ctx, legalEntity, fmt.Sprintf("%d-01-01", t.Year()), fmt.Sprintf("%d-12-31", t.Year()))
	if err != nil {
		return decimal.Zero, err
	}
	credited, err := store.LeaveAccruedDays.Run(ctx, store.LeaveAccruedDaysParams{EmployeeID: employee, Year: int32(t.Year())})
	if err != nil {
		return decimal.Zero, err
	}
	left := entitlementThrough(cal, hire, &termination, t.Year(), int(t.Month())).Sub(decimal.RequireFromString(credited))
	return decimal.Max(left, decimal.Zero), nil
}
