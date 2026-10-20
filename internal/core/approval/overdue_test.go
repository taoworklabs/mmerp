package approval_test

import (
	"sync"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// due gives a step a due time of hours.
func due(e approval.Element, hours int32) approval.Element {
	e.Step.DueHours = &hours
	return e
}

// lapse moves the due time of every step of d's open submission into the past.
func (f *fixture) lapse(d record.Doc) {
	f.t.Helper()
	_, err := platform.DBFrom(f.admin).Exec(f.admin,
		`UPDATE approval.instance_steps SET due_at = now() - interval '1 minute' WHERE instance_id = $1 AND due_at IS NOT NULL`, *d.ApprovalTicket)
	f.check(err)
}

func (f *fixture) overdue() int {
	f.t.Helper()
	n, err := f.appr.NotifyOverdue(f.admin)
	f.check(err)
	return n
}

// told counts login's overdue notifications about d.
func (f *fixture) told(login string, d record.Doc) int {
	f.t.Helper()
	page, err := f.core.Notification.List(f.as(login), nil)
	f.check(err)
	n := 0
	for _, it := range page.Items {
		if it.Kind == string(notification.ApprovalOverdue) && it.RecordID != nil && *it.RecordID == d.ID {
			n++
		}
	}
	return n
}

func TestOverdueTellsPendingVotersAndSubmitter(t *testing.T) {
	f := newFixture(t)
	f.rule(due(f.by(2, 1, "c1", "c2"), 1))
	d, err := f.send("1", "a")
	f.check(err)
	f.check(f.appr.Approve(f.as("c1"), *d.ApprovalTicket, 1))
	f.lapse(d)
	if n := f.overdue(); n != 1 {
		t.Fatalf("notified %d steps", n)
	}
	if f.told("c2", d) != 1 || f.told("submitter", d) != 1 || f.told("c1", d) != 0 {
		t.Fatalf("c2 %d, submitter %d, c1 %d", f.told("c2", d), f.told("submitter", d), f.told("c1", d))
	}
}

func TestOverdueNotifiesOnce(t *testing.T) {
	f := newFixture(t)
	f.rule(due(f.user("boss"), 2))
	d, err := f.send("1", "a")
	f.check(err)
	f.lapse(d)
	if n := f.overdue(); n != 1 {
		t.Fatalf("first run notified %d", n)
	}
	if n := f.overdue(); n != 0 || f.told("boss", d) != 1 {
		t.Fatalf("second run notified %d; boss told %d times", n, f.told("boss", d))
	}

	// Two runs at once share the overdue steps between them.
	e, err := f.send("1", "a")
	f.check(err)
	f.lapse(e)
	var wg sync.WaitGroup
	counts := make([]int, 2)
	for i := range counts {
		wg.Go(func() {
			n, err := f.appr.NotifyOverdue(f.admin)
			if err != nil {
				t.Error(err)
			}
			counts[i] = n
		})
	}
	wg.Wait()
	if counts[0]+counts[1] != 1 || f.told("boss", e) != 1 {
		t.Fatalf("concurrent runs %v; boss told %d", counts, f.told("boss", e))
	}
}

func TestOverdueIgnoresDecidedSteps(t *testing.T) {
	f := newFixture(t)
	f.rule(due(f.user("boss"), 1))
	approved, err := f.send("1", "a")
	f.check(err)
	f.lapse(approved)
	f.check(f.appr.Approve(f.as("boss"), *approved.ApprovalTicket, 1))

	withdrawn, err := f.send("1", "a")
	f.check(err)
	f.lapse(withdrawn)
	_, err = f.rec.Transition(f.as("submitter"), withdrawn.Ref, withdrawn.Version, record.Draft)
	f.check(err)

	rejected, err := f.send("1", "a")
	f.check(err)
	f.lapse(rejected)
	f.check(f.appr.Reject(f.as("boss"), *rejected.ApprovalTicket, 1, "no"))

	if n := f.overdue(); n != 0 {
		t.Fatalf("notified %d decided steps", n)
	}
}

func TestNoDueTimeNeverOverdue(t *testing.T) {
	f := newFixture(t)
	f.rule(f.user("boss"))
	d, err := f.send("1", "a")
	f.check(err)
	if s := f.panel("boss", d).Steps[0]; s.DueAt != nil || s.Overdue {
		t.Fatalf("step without a due time %+v", s)
	}
	if n := f.overdue(); n != 0 {
		t.Fatalf("notified %d", n)
	}
}

func TestOverdueTellsDelegates(t *testing.T) {
	f := newFixture(t)
	f.rule(due(f.user("boss"), 1))
	f.delegate("boss", "big_boss", 0, 1)
	d, err := f.send("1", "a")
	f.check(err)
	f.lapse(d)
	f.overdue()
	if f.told("big_boss", d) != 1 {
		t.Fatal("the delegate was not told")
	}
}

func TestStepViewShowsOverdue(t *testing.T) {
	f := newFixture(t)
	f.rule(due(f.user("boss"), 24))
	d, err := f.send("1", "a")
	f.check(err)
	if s := f.panel("boss", d).Steps[0]; s.DueAt == nil || s.Overdue {
		t.Fatalf("before the due time %+v", s)
	}
	f.lapse(d)
	if s := f.panel("boss", d).Steps[0]; !s.Overdue {
		t.Fatalf("after the due time %+v", s)
	}
}

func TestDueHoursChecked(t *testing.T) {
	f := newFixture(t)
	for _, h := range []int32{0, 721} {
		err := f.appr.SaveRule(f.admin, "test.doc", record.Post, ruleOf(due(f.user("boss"), h)))
		if code(err) != "invalid_rule" {
			t.Fatalf("due_hours %d: %v", h, err)
		}
	}
}
