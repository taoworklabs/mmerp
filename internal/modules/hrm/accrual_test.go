package hrm_test

import (
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/modules/hrm"
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
