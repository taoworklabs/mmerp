package hrm_test

import (
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// closePeriod marks a payroll period of company C as posted, as M6 payroll will;
// any document id stands in for the payroll.
func (f *fixture) closePeriod(start, end string, doc int64) {
	f.t.Helper()
	if _, err := f.pool.Exec(f.admin, `INSERT INTO hrm.payroll_periods (legal_entity_id, period_start, period_end, posted_payroll_id)
		VALUES ($1, $2, $3, $4)`, f.c, start, end, doc); err != nil {
		f.t.Fatal(err)
	}
}

// A leave from 30/03 to 02/04 touches April, so a closed April refuses it; it stays pending.
func TestLeaveAcrossTwoMonthsHitsAClosedPeriod(t *testing.T) {
	f := newLeaveFixture(t)
	f.ruleManager()
	f.grant("12")
	id := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-03-30", "2026-04-02", "4")}))
	f.check(f.send(f.e, id))
	f.closePeriod("2026-04-01", "2026-04-30", id)
	err := f.appr.Approve(f.m, f.ticket(id), 1)
	wantErr(t, err, "payroll_period_closed")
	if p := errParams(err)["periods"].([]hrm.Period); len(p) != 1 || p[0] != (hrm.Period{Start: "2026-04-01", End: "2026-04-30"}) {
		t.Fatalf("periods %v", p)
	}
	if l := f.leave(id); l.Status != "pending_approval" || f.balance() != "12" {
		t.Fatalf("%+v, balance %s", l, f.balance())
	}

	// A posted leave in a period closed later cannot be cancelled.
	may := must(t)(f.hrm.CreateLeave(f.e, hrm.NewLeave{LeaveFields: f.fields("2026-05-04", "2026-05-04", "1")}))
	f.check(f.send(f.e, may))
	f.check(f.appr.Approve(f.m, f.ticket(may), 1))
	f.closePeriod("2026-05-01", "2026-05-31", may)
	wantErr(t, f.rec.Transition(f.hr, record.Ref{Type: "hrm.leave_request", ID: may}, f.leave(may).Version, record.Cancelled), "payroll_period_closed")
}

func errParams(err error) map[string]any {
	if e, ok := errors.AsType[*platform.Error](err); ok {
		return e.Params
	}
	return nil
}
