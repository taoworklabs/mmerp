package hrm_test

import (
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/modules/hrm"
)

func TestWorkCalendar(t *testing.T) {
	f := newLeaveFixture(t)
	pay := f.ctxFor("pay", "payroll@*")

	c, err := f.hrm.WorkCalendar(f.hr, f.c, 2026)
	if err != nil || len(c.WorkWeeks) != 0 || len(c.AllowedActions) != 0 {
		t.Fatalf("hr sees %+v %v", c, err)
	}
	_, err = f.hrm.WorkCalendar(f.hr, f.a, 2026)
	wantErr(t, err, "not_a_legal_entity")
	wantErr(t, f.hrm.SaveHoliday(f.hr, f.c, hrm.Holiday{Date: "2026-04-30", Name: "Giải phóng miền Nam"}), "forbidden")

	f.check(f.hrm.SaveHoliday(pay, f.c, hrm.Holiday{Date: "2026-04-30", Name: "Giải phóng miền Nam"}))
	f.check(f.hrm.SaveWorkWeek(pay, f.c, hrm.WorkWeek{EffectiveFrom: "2026-03-16", OffDays: []int{0}}))
	c, err = f.hrm.WorkCalendar(pay, f.c, 2026)
	if err != nil || len(c.Holidays) != 1 || len(c.WorkWeeks) != 1 || !slices.Equal(c.AllowedActions, []string{"manage"}) {
		t.Fatalf("pay sees %+v %v", c, err)
	}

	// The grid of March greys Saturdays only until the 16th, Sundays all month.
	ts := f.timesheet(must(t)(f.hrm.CreateTimesheet(f.hr, hrm.NewTimesheet{OrgUnitID: f.a, Month: "2026-03"})))
	want := []string{"2026-03-01", "2026-03-07", "2026-03-08", "2026-03-14", "2026-03-15", "2026-03-22", "2026-03-29"}
	if !slices.Equal(ts.OffDays, want) {
		t.Fatalf("off days %v", ts.OffDays)
	}

	// A closed April freezes its holidays and the work week in force over it.
	f.closePeriod("2026-04-01", "2026-04-30", ts.ID)
	wantErr(t, f.hrm.DeleteHoliday(pay, f.c, "2026-04-30"), "payroll_period_closed")
	wantErr(t, f.hrm.SaveWorkWeek(pay, f.c, hrm.WorkWeek{EffectiveFrom: "2026-03-16", OffDays: []int{0, 6}}), "payroll_period_closed")
	f.check(f.hrm.SaveWorkWeek(pay, f.c, hrm.WorkWeek{EffectiveFrom: "2026-05-01", OffDays: []int{0, 6}}))
	// A version ends where the next one starts, so one from February stops before April.
	f.check(f.hrm.SaveWorkWeek(pay, f.c, hrm.WorkWeek{EffectiveFrom: "2026-02-01", OffDays: []int{0}}))
}

func TestLegalParams(t *testing.T) {
	f := newLeaveFixture(t)
	pay := f.ctxFor("pay", "payroll@*")
	l, err := f.hrm.LegalParams(f.hr)
	if err != nil || len(l.Items) != 21 || len(l.AllowedActions) != 0 {
		t.Fatalf("defaults: %d items, %v %v", len(l.Items), l.AllowedActions, err)
	}
	wantErr(t, f.hrm.SaveLegalParam(f.hr, hrm.LegalParam{Key: "base_salary", EffectiveFrom: "2027-01-01", Value: "2600000"}), "forbidden")
	wantErr(t, f.hrm.SaveLegalParam(pay, hrm.LegalParam{Key: "base_salary", EffectiveFrom: "2027-01-01", Value: "1e9"}), "invalid_legal_param")
	wantErr(t, f.hrm.SaveLegalParam(pay, hrm.LegalParam{Key: "pit_brackets", EffectiveFrom: "2027-01-01", Value: "[[10,0.05],[5,0.1],[null,0.2]]"}), "invalid_legal_param")
	wantErr(t, f.hrm.SaveLegalParam(pay, hrm.LegalParam{Key: "nope", EffectiveFrom: "2027-01-01", Value: "1"}), "not_found")
	f.check(f.hrm.SaveLegalParam(pay, hrm.LegalParam{Key: "pit_brackets", EffectiveFrom: "2027-01-01", Value: "[[10000000,0.05],[null,0.1]]"}))

	// The July base salary covers July onward; closing August freezes it, not the January one.
	f.closePeriod("2026-08-01", "2026-08-31", must(t)(f.hrm.CreateTimesheet(f.hr, hrm.NewTimesheet{OrgUnitID: f.a, Month: "2026-08"})))
	wantErr(t, f.hrm.SaveLegalParam(pay, hrm.LegalParam{Key: "base_salary", EffectiveFrom: "2026-07-01", Value: "2600000"}), "payroll_period_closed")
	f.check(f.hrm.SaveLegalParam(pay, hrm.LegalParam{Key: "base_salary", EffectiveFrom: "2026-01-01", Value: "2340000"}))
	if l, _ := f.hrm.LegalParams(pay); len(l.Items) != 22 || !slices.Equal(l.AllowedActions, []string{"manage"}) {
		t.Fatalf("after saving: %d items, %v", len(l.Items), l.AllowedActions)
	}
	if n := f.auditCount("hrm.legal_param_changed"); n != 2 {
		t.Fatalf("audited %d", n)
	}
}
