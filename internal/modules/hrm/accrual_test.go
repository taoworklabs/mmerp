package hrm_test

import (
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// accrualFixture: the leave fixture with E hired long ago, so every month counts.
func newAccrualFixture(t *testing.T, hire string, termination *string) *leaveFixture {
	f := newLeaveFixture(t)
	if _, err := f.system.Exec(f.admin, `UPDATE hrm.employees SET hire_date = $2, termination_date = $3 WHERE id = $1`, f.emp, hire, termination); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *leaveFixture) accrue(today string) {
	f.t.Helper()
	f.check(f.hrm.AccrueLeave(f.admin, today))
}

func (f *leaveFixture) balanceIn(year int32) string {
	f.t.Helper()
	bs, err := f.hrm.LeaveBalances(f.hr, f.emp)
	f.check(err)
	for _, b := range bs {
		if b.Year == year {
			return b.Days
		}
	}
	return "0"
}

func (f *leaveFixture) ledger(year int) []int {
	f.t.Helper()
	rows, err := f.system.Query(f.admin, `SELECT month FROM hrm.leave_accruals WHERE employee_id = $1 AND year = $2 ORDER BY month`, f.emp, year)
	f.check(err)
	defer rows.Close()
	var out []int
	for rows.Next() {
		var m int
		f.check(rows.Scan(&m))
		out = append(out, m)
	}
	return out
}

func (f *leaveFixture) setting(key, value string) {
	f.t.Helper()
	f.check(f.core.Setting.SetFor(f.admin, f.c, key, value))
}

// A run credits every month completed since the last one, once.
func TestAccrualCatchesUp(t *testing.T) {
	f := newAccrualFixture(t, "2023-01-01", nil)
	f.accrue("2026-04-01")
	if got := f.balance(); got != "3" {
		t.Fatalf("balance after March: %s", got)
	}
	if got := f.ledger(2026); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("ledger: %v", got)
	}
	f.accrue("2026-04-01")
	if got := f.balance(); got != "3" {
		t.Fatalf("a second run credited again: %s", got)
	}
	f.accrue("2026-07-02")
	if got := f.balance(); got != "6" {
		t.Fatalf("balance after June: %s", got)
	}
}

// A manual legal entity is HR's to grant: the job credits nothing and decides no month.
func TestAccrualManualUntouched(t *testing.T) {
	f := newAccrualFixture(t, "2023-01-01", nil)
	f.setting(hrm.SettingLeaveAccrual, "manual")
	f.accrue("2026-04-01")
	if got := f.balance(); got != "0" {
		t.Fatalf("balance: %s", got)
	}
	if got := f.ledger(2026); len(got) != 0 {
		t.Fatalf("ledger: %v", got)
	}
}

// Every credit lands in the employee's history with the resulting balance.
func TestAccrualAudited(t *testing.T) {
	f := newAccrualFixture(t, "2023-01-01", nil)
	f.accrue("2026-04-01")
	var data string
	if err := f.system.QueryRow(f.admin, `SELECT data::text FROM audit.log WHERE action = 'hrm.leave_accrued'
		AND doc_type = 'hrm.employee' AND doc_id = $1 AND data->>'month' = '3'`, f.emp).Scan(&data); err != nil {
		t.Fatal(err)
	}
	if data != `{"days": "1", "year": 2026, "month": 3, "balance": "3"}` {
		t.Fatalf("audit data: %s", data)
	}
}

// Last year is caught up in January, never back-filled later by turning accrual on.
func TestAccrualNoBackfillAfterJanuary(t *testing.T) {
	f := newAccrualFixture(t, "2023-01-01", nil)
	f.accrue("2026-10-01")
	if got := f.ledger(2025); len(got) != 0 {
		t.Fatalf("2025 back-filled in October: %v", got)
	}
	if got := f.balance(); got != "9" {
		t.Fatalf("2026 balance: %s", got)
	}

	g := newAccrualFixture(t, "2023-01-01", nil)
	g.setting(hrm.SettingCarryOverMonths, "12")
	g.accrue("2026-01-02")
	if got := g.ledger(2025); len(got) != 12 {
		t.Fatalf("2025 in January: %v", got)
	}
}

// A leaver is credited up to the termination month and no further.
func TestAccrualLeaverStops(t *testing.T) {
	f := newAccrualFixture(t, "2023-01-01", new("2026-03-20"))
	f.accrue("2026-06-01")
	if got := f.ledger(2026); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("ledger: %v", got)
	}
	if got := f.balance(); got != "3" {
		t.Fatalf("balance: %s", got)
	}
}

// An employee hired after the months being decided gets no row for them.
func TestAccrualFutureHire(t *testing.T) {
	f := newAccrualFixture(t, "2026-05-04", nil)
	f.accrue("2026-04-01")
	if got := f.ledger(2026); len(got) != 0 {
		t.Fatalf("ledger: %v", got)
	}
}

// carryFixture: C on manual, E with 5 days left of 2025, carried for months of 2026.
func newCarryFixture(t *testing.T, months string, termination *string) *leaveFixture {
	f := newAccrualFixture(t, "2023-01-01", termination)
	f.setting(hrm.SettingLeaveAccrual, "manual")
	f.setting(hrm.SettingCarryOverMonths, months)
	f.check(f.hrm.AdjustLeaveBalance(f.hr, f.emp, 2025, hrm.BalanceAdjustment{Delta: "5", Reason: "còn lại 2025"}))
	return f
}

func (f *leaveFixture) carried() (string, *string) {
	f.t.Helper()
	bs, err := f.hrm.LeaveBalances(f.hr, f.emp)
	f.check(err)
	for _, b := range bs {
		if b.Year == 2026 {
			return b.Carried, b.CarriedUntil
		}
	}
	return "0", nil
}

func (f *leaveFixture) approved(start, end, days string) int64 {
	f.t.Helper()
	id := must(f.t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields(start, end, days)}))
	f.check(f.send(f.e, id))
	return id
}

func (f *leaveFixture) cancel(id int64) {
	f.t.Helper()
	f.check(errOf(f.rec.Transition(f.hr, record.Ref{Type: "hrm.leave_request", ID: id}, f.leave(id).Version, record.Cancelled)))
}

// On the new year last year's days move into this year as carried days, once.
func TestCarryOver(t *testing.T) {
	f := newCarryFixture(t, "3", nil)
	f.accrue("2026-01-02")
	carried, until := f.carried()
	if f.balanceIn(2025) != "0" || f.balance() != "5" || carried != "5" || until == nil || *until != "2026-03-31" {
		t.Fatalf("2025 %s, 2026 %s, carried %s until %v", f.balanceIn(2025), f.balance(), carried, until)
	}
	if f.auditCount("hrm.leave_carried_over") != 1 {
		t.Fatal("carry-over not audited")
	}
	f.accrue("2026-01-03")
	if f.balance() != "5" || f.auditCount("hrm.leave_carried_over") != 1 {
		t.Fatalf("a rerun carried again: %s", f.balance())
	}
}

// Leave dated up to the expiry spends carried days first; what is left lapses after it.
func TestCarryExpires(t *testing.T) {
	f := newCarryFixture(t, "3", nil)
	f.accrue("2026-01-02")
	f.approved("2026-03-02", "2026-03-03", "2")
	if carried, _ := f.carried(); carried != "3" || f.balance() != "3" {
		t.Fatalf("carried %s, balance %s", carried, f.balance())
	}
	f.accrue("2026-04-01")
	if carried, _ := f.carried(); carried != "0" || f.balance() != "0" {
		t.Fatalf("after expiry: carried %s, balance %s", carried, f.balance())
	}
	if f.auditCount("hrm.leave_carry_expired") != 1 {
		t.Fatal("expiry not audited")
	}
}

// With no carry-over, last year's days lapse on the new year rather than sit unreachable.
func TestCarryOverZeroLapses(t *testing.T) {
	f := newCarryFixture(t, "0", nil)
	f.accrue("2026-01-02")
	if f.balanceIn(2025) != "0" || f.balance() != "0" || f.auditCount("hrm.leave_carry_expired") != 1 {
		t.Fatalf("2025 %s, 2026 %s", f.balanceIn(2025), f.balance())
	}
}

// Cancelling gives carried days back while they last, and drops them once expired.
func TestCarriedRefund(t *testing.T) {
	// Until the end of 2026: still carried on any day of this test.
	f := newCarryFixture(t, "12", nil)
	f.accrue("2026-01-02")
	id := f.approved("2026-03-02", "2026-03-03", "2")
	f.cancel(id)
	if carried, _ := f.carried(); carried != "5" || f.balance() != "5" {
		t.Fatalf("refund before expiry: carried %s, balance %s", carried, f.balance())
	}

	// Expired at the end of March, before today: the carried part is dropped, as it would have lapsed.
	g := newCarryFixture(t, "3", nil)
	g.accrue("2026-01-02")
	id = g.approved("2026-03-02", "2026-03-03", "2")
	g.cancel(id)
	if carried, _ := g.carried(); carried != "3" || g.balance() != "3" {
		t.Fatalf("refund after expiry: carried %s, balance %s", carried, g.balance())
	}
}

// A hand adjustment below the carried part takes the carried part down with it.
func TestAdjustBelowCarried(t *testing.T) {
	f := newCarryFixture(t, "3", nil)
	f.accrue("2026-01-02")
	f.check(f.hrm.AdjustLeaveBalance(f.hr, f.emp, 2026, hrm.BalanceAdjustment{Delta: "-3", Reason: "sửa"}))
	if carried, _ := f.carried(); carried != "2" || f.balance() != "2" {
		t.Fatalf("carried %s, balance %s", carried, f.balance())
	}
}

// A leaver's balance is final once paid out: no expiry after they left, no carry into a later year.
func TestLeaverNotCarriedNorExpired(t *testing.T) {
	f := newCarryFixture(t, "3", new("2026-03-15"))
	f.accrue("2026-01-02")
	f.accrue("2026-04-01")
	if carried, _ := f.carried(); carried != "5" || f.balance() != "5" {
		t.Fatalf("April run touched a leaver: carried %s, balance %s", carried, f.balance())
	}
	f.accrue("2027-01-04")
	if f.balance() != "5" || f.balanceIn(2027) != "0" {
		t.Fatalf("carried into 2027: 2026 %s, 2027 %s", f.balance(), f.balanceIn(2027))
	}
}

// Leave dated after the carried days lapse cannot spend them.
func TestCarriedNotForLaterLeave(t *testing.T) {
	f := newCarryFixture(t, "3", nil)
	f.accrue("2026-01-02")
	f.grant("1")
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-06-01", "2026-06-02", "2")}))
	wantErr(t, f.send(f.e, id), "insufficient_leave_balance")
	f.approved("2026-06-01", "2026-06-01", "1")
	if carried, _ := f.carried(); carried != "5" || f.balance() != "5" {
		t.Fatalf("carried %s, balance %s", carried, f.balance())
	}
}

// The termination month is credited only after the employee has left, yet the payroll that
// pays out their untaken leave counts it; crediting it later changes no payroll input.
func TestPayrollPaysTerminationMonthAccrual(t *testing.T) {
	f := newPayrollFixture(t)
	f.setting(hrm.SettingLeaveAccrual, "monthly")
	x := employee("X", f.a)
	x.HireDate, x.TerminationDate = "2023-01-01", new("2026-04-20")
	emp := must(t)(f.hrm.CreateEmployee(f.hr, x))
	// 22,000,000 over March's 22 working days: a day of leave is worth 1,000,000.
	f.contract(emp, sample{salary: 22_000_000}, "2026-01-01", "2026-06-30")
	f.accrue("2026-04-01")
	if got := f.balanceOfEmp(emp); got != "3" {
		t.Fatalf("balance before leaving: %s", got)
	}
	f.timesheet(f.a, "2026-04", map[int64][]string{emp: workdays("2026-04-01", "2026-04-20", "2026-04-27", "2026-04-30")})
	id := f.payroll("2026-04")
	var line hrm.PayrollLine
	for _, l := range f.payrollDoc(id).Lines {
		if l.EmployeeID == emp {
			line = l
		}
	}
	if line.UnusedLeave != 4_000_000 {
		t.Fatalf("unused leave paid %d, want four days", line.UnusedLeave)
	}
	f.accrue("2026-05-01")
	if got := f.balanceOfEmp(emp); got != "4" {
		t.Fatalf("balance after the job: %s", got)
	}
	if f.payrollDoc(id).SourcesChanged {
		t.Fatal("crediting April changed the payroll's inputs")
	}
}

func (f *payrollFixture) balanceOfEmp(emp int64) string {
	f.t.Helper()
	bs, err := f.hrm.LeaveBalances(f.pay, emp)
	f.check(err)
	for _, b := range bs {
		if b.Year == 2026 {
			return b.Days
		}
	}
	return "0"
}

// A termination entered after the job credited later months: the payout counts only what the
// employee earned up to leaving, not the months credited since.
func TestPayrollLateTerminationTakesBackLaterCredits(t *testing.T) {
	f := newPayrollFixture(t)
	f.setting(hrm.SettingLeaveAccrual, "monthly")
	x := employee("X", f.a)
	x.HireDate = "2023-01-01"
	emp := must(t)(f.hrm.CreateEmployee(f.hr, x))
	// 23,000,000 over July's 23 working days: a day of leave is worth 1,000,000.
	f.contract(emp, sample{salary: 23_000_000}, "2026-01-01", "2026-12-31")
	f.accrue("2026-10-01")
	if got := f.balanceOfEmp(emp); got != "9" {
		t.Fatalf("balance through September: %s", got)
	}
	// Left on 15 August, entered only now: August holds 10 of 21 working days, so it does not count.
	if _, err := f.system.Exec(f.admin, `UPDATE hrm.employees SET termination_date = '2026-08-15' WHERE id = $1`, emp); err != nil {
		t.Fatal(err)
	}
	f.timesheet(f.a, "2026-08", map[int64][]string{emp: workdays("2026-08-03", "2026-08-14")})
	var line hrm.PayrollLine
	for _, l := range f.payrollDoc(f.payroll("2026-08")).Lines {
		if l.EmployeeID == emp {
			line = l
		}
	}
	if line.UnusedLeave != 7_000_000 {
		t.Fatalf("unused leave paid %d, want seven days", line.UnusedLeave)
	}
}

// Carried days lapse even when they last all year: next year's carry-over expires them
// first and moves only this year's own days.
func TestCarriedAllYearLapseAtNextCarry(t *testing.T) {
	f := newCarryFixture(t, "12", nil)
	f.accrue("2026-01-02")
	f.grant("3")
	f.accrue("2027-01-04")
	if f.balance() != "0" || f.balanceIn(2027) != "3" || f.auditCount("hrm.leave_carry_expired") != 1 {
		t.Fatalf("2026 %s, 2027 %s, expiries %d", f.balance(), f.balanceIn(2027), f.auditCount("hrm.leave_carry_expired"))
	}
}

// A disabled product stays read-only, the accrual job included.
func TestAccrualSkipsDisabledHRM(t *testing.T) {
	f := newAccrualFixture(t, "2023-01-01", nil)
	f.check(f.hrm.AccrueLeave(platform.WithProducts(f.admin, nil), "2026-04-01"))
	if got := f.ledger(2026); len(got) != 0 || f.auditCount("hrm.leave_accrued") != 0 {
		t.Fatalf("ledger %v", got)
	}
}

// A termination dated last year but entered after the new year's carry-over gets its days
// back in the termination year, so that year's payroll pays them out.
func TestLateTerminationReversesCarry(t *testing.T) {
	f := newPayrollFixture(t)
	x := employee("X", f.a)
	x.HireDate = "2023-01-01"
	emp := must(t)(f.hrm.CreateEmployee(f.hr, x))
	// 21,000,000 over November's 21 working days: a day of leave is worth 1,000,000.
	f.contract(emp, sample{salary: 21_000_000}, "2026-01-01", "2026-12-31")
	f.check(f.hrm.AdjustLeaveBalance(f.pay, emp, 2026, hrm.BalanceAdjustment{Delta: "5", Reason: "còn lại"}))
	f.accrue("2027-01-04")
	if _, err := f.system.Exec(f.admin, `UPDATE hrm.employees SET termination_date = '2026-12-10' WHERE id = $1`, emp); err != nil {
		t.Fatal(err)
	}
	f.accrue("2027-01-15")
	bs, err := f.hrm.LeaveBalances(f.pay, emp)
	f.check(err)
	got := map[int32]string{}
	for _, b := range bs {
		got[b.Year] = b.Days + "/" + b.Carried
	}
	if got[2026] != "5/0" || got[2027] != "0/0" || f.auditCount("hrm.leave_carry_reversed") != 1 {
		t.Fatalf("balances %v", got)
	}
	f.timesheet(f.a, "2026-12", map[int64][]string{emp: workdays("2026-12-01", "2026-12-10")})
	for _, l := range f.payrollDoc(f.payroll("2026-12")).Lines {
		if l.EmployeeID == emp && l.UnusedLeave != 5_000_000 {
			t.Fatalf("unused leave paid %d, want five days", l.UnusedLeave)
		}
	}
}
