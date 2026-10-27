package hrm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// The personal space: what the actor's own employee record holds. "Own" is the employee
// linked to the actor's account; no endpoint takes an employee id.

// me is the actor's own employee record.
func me(ctx context.Context) (store.EmployeeByUserRow, error) {
	actor, _ := platform.ActorFrom(ctx)
	self, err := store.EmployeeByUser.Run(ctx, pgtype.Int8{Int64: actor, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return self, ErrNotAnEmployee
	}
	return self, err
}

// MyProfile returns the actor's ordinary fields and the contract in force today.
func (s *Service) MyProfile(ctx context.Context) (MyProfile, error) {
	self, err := me(ctx)
	if err != nil {
		return MyProfile{}, err
	}
	e, err := store.GetEmployee.Run(ctx, self.ID)
	if err != nil {
		return MyProfile{}, err
	}
	today, err := s.d.Setting.Today(ctx)
	if err != nil {
		return MyProfile{}, err
	}
	out := MyProfile{EmployeeID: e.ID, Code: e.Code, FullName: e.FullName, DateOfBirth: platform.DatePtr(e.DateOfBirth),
		Gender: platform.TextPtr(e.Gender), Phone: platform.TextPtr(e.Phone), Email: platform.TextPtr(e.Email), Address: platform.TextPtr(e.Address),
		OrgUnitName: e.OrgUnitName, ManagerName: platform.TextPtr(e.ManagerName), HireDate: *platform.DatePtr(e.HireDate),
		TerminationDate: platform.DatePtr(e.TerminationDate)}
	c, err := store.CurrentContract.Run(ctx, store.CurrentContractParams{EmployeeID: e.ID, At: today, AllUnits: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err == nil {
		out.Contract = &MyContract{TypeName: c.ContractTypeName, StartDate: *platform.DatePtr(c.StartDate), EndDate: platform.DatePtr(c.EndDate)}
	}
	return out, err
}

// myDayKinds names the calendar's day kinds as the personal space shows them.
var myDayKinds = map[string]string{"weekday": "work", "weekly_off": "rest", "holiday": "holiday"}

// MyDays returns the actor's month (YYYY-MM): worked days from its timesheets that are not
// cancelled, posted leave and overtime, and the calendar of their legal entity.
func (s *Service) MyDays(ctx context.Context, month string) (MyDays, error) {
	self, err := me(ctx)
	if err != nil {
		return MyDays{}, err
	}
	start, end, err := platform.MonthPeriod(month)
	if err != nil {
		return MyDays{}, err
	}
	le, err := s.d.IAM.LegalEntityOf(ctx, self.OrgUnitID)
	if err != nil {
		return MyDays{}, err
	}
	// A leave request stays within one year, so the year's calendar places all of it.
	year := start.Time.Year()
	cal, err := workCalendar(ctx, le, fmt.Sprintf("%d-01-01", year), fmt.Sprintf("%d-12-31", year))
	if err != nil {
		return MyDays{}, err
	}
	out := MyDays{Month: month, StandardDays: cal.standardDays(start.Time, end.Time), Days: []MyDay{}}
	at := map[string]*MyDay{}
	for d := start.Time; !d.After(end.Time); d = d.AddDate(0, 0, 1) {
		out.Days = append(out.Days, MyDay{Date: d.Format(time.DateOnly), Kind: myDayKinds[cal.dayKind(d)]})
	}
	for i := range out.Days {
		at[out.Days[i].Date] = &out.Days[i]
	}

	worked, err := store.MyTimesheetLines.Run(ctx, store.MyTimesheetLinesParams{EmployeeID: self.ID, PeriodStart: start})
	if err != nil {
		return out, err
	}
	for _, w := range worked {
		d := at[*platform.DatePtr(w.Date)]
		if d == nil {
			continue
		}
		v := decimal.RequireFromString(w.Days).String()
		d.Worked, d.WorkedClosed = &v, w.Status == "posted"
	}

	leaves, err := store.MyLeaves.Run(ctx, store.MyLeavesParams{EmployeeID: self.ID, FromDate: start, ToDate: end})
	if err != nil {
		return out, err
	}
	for _, l := range leaves {
		for date, days := range leaveAllocation(cal, *platform.DatePtr(l.StartDate), *platform.DatePtr(l.EndDate), decimal.RequireFromString(l.Days)) {
			// ponytail: one leave per day shown; a second half-day request on the same date is left out.
			if d := at[date]; d != nil && d.Leave == nil {
				d.Leave = &MyDayLeave{TypeName: l.TypeName, Days: days.String(), Paid: l.Paid}
			}
		}
	}

	ot, err := store.MyOvertime.Run(ctx, store.MyOvertimeParams{EmployeeID: self.ID, FromDate: start, ToDate: end})
	if err != nil {
		return out, err
	}
	for _, o := range ot {
		d := at[*platform.DatePtr(o.Date)]
		sum := decimal.RequireFromString(o.Hours)
		if d.OvertimeHours != nil {
			sum = sum.Add(decimal.RequireFromString(*d.OvertimeHours))
		}
		v := sum.String()
		d.OvertimeHours = &v
	}
	return out, nil
}

// MyPayslips lists the actor's payslips of posted payrolls, newest first; the read is audited.
func (s *Service) MyPayslips(ctx context.Context) ([]MyPayslipSummary, error) {
	out := []MyPayslipSummary{}
	err := platform.InTx(ctx, func(ctx context.Context) error {
		self, err := me(ctx)
		if err != nil {
			return err
		}
		if err := s.d.Audit.RecordFor(ctx, "hrm.own_payslips_viewed", audit.Ref{Type: employeeType, ID: self.ID}, nil); err != nil {
			return err
		}
		rows, err := store.MyPostedPayrollLines.Run(ctx, self.ID)
		if err != nil {
			return err
		}
		for _, r := range rows {
			l, err := openLine(ctx, r.Data)
			if err != nil {
				return err
			}
			out = append(out, MyPayslipSummary{PayrollID: r.ID, PeriodStart: *platform.DatePtr(r.PeriodStart), Net: l.Result.Net})
		}
		return nil
	})
	return out, err
}

// MyPayslip returns the actor's part of a posted payroll; the read is audited on the payroll.
func (s *Service) MyPayslip(ctx context.Context, payrollID int64) (MyPayslip, error) {
	var out MyPayslip
	err := platform.InTx(ctx, func(ctx context.Context) error {
		self, err := me(ctx)
		if err != nil {
			return err
		}
		r, err := store.MyPayrollLine.Run(ctx, store.MyPayrollLineParams{EmployeeID: self.ID, PayrollID: payrollID})
		if errors.Is(err, pgx.ErrNoRows) {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := s.d.Audit.RecordFor(ctx, "hrm.own_payslip_viewed", audit.Ref{Type: payrollType, ID: payrollID}, map[string]any{"employee_id": self.ID}); err != nil {
			return err
		}
		l, err := openLine(ctx, r.Data)
		if err != nil {
			return err
		}
		adjustments, err := s.adjustments(ctx, payrollID)
		if err != nil {
			return err
		}
		out = MyPayslip{PayrollID: payrollID, Number: r.Number, PeriodStart: *platform.DatePtr(r.PeriodStart), StandardDays: l.Input.StandardDays,
			PaidDays: l.Result.PaidDays.String(), OvertimeHours: l.Result.OvertimeHours.String(), Amounts: l.Result.PayrollAmounts,
			Adjustments: []PayrollAdjustmentInput{}}
		for _, a := range adjustments {
			if a.EmployeeID == self.ID {
				out.Adjustments = append(out.Adjustments, a.PayrollAdjustmentInput)
			}
		}
		return nil
	})
	return out, err
}
