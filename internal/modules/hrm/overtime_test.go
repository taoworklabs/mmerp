package hrm_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// overtime is a request on date with the kind the default calendar gives it: Saturday and
// Sunday off, no holidays.
func overtime(date, day, night string) hrm.OvertimeFields {
	kind := "weekday"
	if d, _ := time.Parse(time.DateOnly, date); d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		kind = "weekly_off"
	}
	return hrm.OvertimeFields{Date: date, DayKind: kind, DayHours: day, NightHours: night}
}

func (f *leaveFixture) postedOvertimes() int {
	f.t.Helper()
	l, err := f.hrm.Overtimes(f.hr, hrm.OvertimeFilter{Status: "posted", Sort: "-date", Paging: platform.Paging{Page: 1, PageSize: 20}})
	f.check(err)
	return int(l.Total)
}

func TestOvertimeRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		f := newLeaveFixture(t)
		ref := func(id int64, err error) (record.Ref, error) {
			return record.Ref{Type: "hrm.overtime_request", ID: id}, err
		}
		return recordtest.Harness{
			Core: f.core, Actor: f.hr, Admin: f.admin, LegalEntity: f.c,
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				return ref(f.hrm.CreateOvertime(ctx, hrm.NewOvertime{RequestID: requestID, EmployeeID: &f.emp, OvertimeFields: overtime(date, "2", "0")}))
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return f.hrm.UpdateOvertime(ctx, r.ID, hrm.OvertimeUpdate{Version: version, OvertimeFields: overtime(date, "2", "0")})
			},
			Get:    func(ctx context.Context, r record.Ref) (any, error) { return f.hrm.Overtime(ctx, r.ID) },
			Reject: func(instance int64) error { return f.appr.Reject(f.m, instance, 1, "không") },
			RequireApproval: func(*testing.T) {
				f.check(f.appr.SaveRule(f.admin, "hrm.overtime_request", approval.RuleInput{
					Steps: []approval.Step{{Approver: approval.Approver{Kind: "module"}}}, MaxLevels: 3, FallbackProduct: "hrm", FallbackRole: "hr",
				}))
			},
			Approve:      func(instance int64) error { return f.appr.Approve(f.m, instance, 1) },
			BreakPosting: func(_ *testing.T, r record.Ref) { f.closePeriod("2026-03-01", "2026-03-31", r.ID) },
			Effects:      func(*testing.T, record.Ref) int { return f.postedOvertimes() },
		}
	})
}

func TestOvertime(t *testing.T) {
	f := newLeaveFixture(t)
	for _, c := range []struct{ day, night string }{{"0", "0"}, {"20", "4.5"}, {"1.2", "0"}} {
		_, err := f.hrm.CreateOvertime(f.e, hrm.NewOvertime{OvertimeFields: overtime("2026-03-31", c.day, c.night)})
		wantErr(t, err, "invalid_overtime_hours")
	}
	// The day is the affected range: 31/03 passes with April closed.
	id := must(t)(f.hrm.CreateOvertime(f.e, hrm.NewOvertime{OvertimeFields: overtime("2026-03-31", "2", "1.5")}))
	d, err := f.rec.Get(f.admin, record.Ref{Type: "hrm.overtime_request", ID: id})
	f.check(err)
	if d.Fields["hours"] != "3.5" || d.Fields["day_kind"] != "weekday" || d.Number != "TC-2026-00001" {
		t.Fatalf("header %+v", d)
	}
	f.closePeriod("2026-04-01", "2026-04-30", id)
	f.check(f.rec.Transition(f.e, record.Ref{Type: "hrm.overtime_request", ID: id}, d.Version, record.Posted))
	april := must(t)(f.hrm.CreateOvertime(f.e, hrm.NewOvertime{OvertimeFields: overtime("2026-04-01", "2", "0")}))
	wantErr(t, f.rec.Transition(f.e, record.Ref{Type: "hrm.overtime_request", ID: april}, 1, record.Posted), "payroll_period_closed")

	// The manager sees E's request read-only; someone else does not see it.
	if o, err := f.hrm.Overtime(f.m, id); err != nil || o.DayHours != "2" || o.NightHours != "1.5" || len(o.AllowedActions) != 0 {
		t.Fatalf("manager: %+v %v", o, err)
	}
	if _, err := f.hrm.Overtime(f.ctxFor("other"), id); errCode(err) != "not_found" {
		t.Fatalf("other: %v", err)
	}
}

// Approval rules show the day kinds in the viewer's language.
func TestDayKindLabels(t *testing.T) {
	f := newLeaveFixture(t)
	for locale, want := range map[string]string{"vi": "Ngày lễ", "en": "Public holiday"} {
		f.check(f.iam.SetLocale(f.admin, locale))
		rules, err := f.appr.Rules(f.admin, "")
		f.check(err)
		var got string
		for _, r := range rules {
			for _, fl := range r.Fields {
				for _, o := range fl.Options {
					if r.DocType == "hrm.overtime_request" && o.Value == "holiday" {
						got = o.Label
					}
				}
			}
		}
		if got != want {
			t.Fatalf("%s: %q, want %q", locale, got, want)
		}
	}
}

// The API enums of day kinds are written by hand; they must list exactly the kinds the service knows.
func TestDayKindEnums(t *testing.T) {
	for _, v := range []any{hrm.OvertimeFields{}, hrm.OvertimeListItem{}} {
		f, _ := reflect.TypeOf(v).FieldByName("DayKind")
		if got := strings.Split(f.Tag.Get("enum"), ","); !reflect.DeepEqual(got, hrm.DayKinds) {
			t.Fatalf("%T enum %v, want %v", v, got, hrm.DayKinds)
		}
	}
}

// Paperwork goes with an overtime request as with a leave request: whoever sees the request sees
// its files, and whoever may edit it adds them.
func TestOvertimeAttachments(t *testing.T) {
	f := newLeaveFixture(t)
	id := must(t)(f.hrm.CreateOvertime(f.e, hrm.NewOvertime{OvertimeFields: overtime("2026-03-11", "2", "0")}))
	for name, c := range map[string]struct {
		ctx    context.Context
		action record.Action
		want   bool
	}{
		"submitter views": {f.e, record.ViewFiles, true},
		"submitter adds":  {f.e, record.Attach, true},
		"manager views":   {f.m, record.ViewFiles, true},
		"manager adds":    {f.m, record.Attach, false},
		"stranger views":  {f.ctxFor("stranger"), record.ViewFiles, false},
	} {
		if ok, err := f.rec.Can(c.ctx, "hrm.overtime_request", id, c.action); err != nil || ok != c.want {
			t.Fatalf("%s: %v %v", name, ok, err)
		}
	}
}

// The day kind must be the one the calendar gives the date, checked on save and again when
// the request goes into force, since the calendar may change in between. A holiday wins over
// a weekly day off.
func TestDayKindFollowsTheCalendar(t *testing.T) {
	f := newLeaveFixture(t)
	pay := f.ctxFor("pay", "payroll@*")
	f.check(f.hrm.SaveHoliday(pay, f.c, hrm.Holiday{Date: "2026-04-30", Name: "Giải phóng miền Nam"}))
	f.check(f.hrm.SaveHoliday(pay, f.c, hrm.Holiday{Date: "2026-05-03", Name: "Ngày lễ rơi vào Chủ nhật"}))
	create := func(date, kind string) (int64, error) {
		return f.hrm.CreateOvertime(f.e, hrm.NewOvertime{OvertimeFields: hrm.OvertimeFields{Date: date, DayKind: kind, DayHours: "2", NightHours: "0"}})
	}
	for _, c := range []struct{ date, kind, want string }{
		{"2026-04-30", "weekday", "holiday"},
		{"2026-04-29", "weekly_off", "weekday"},
		{"2026-04-26", "weekday", "weekly_off"},
		{"2026-05-03", "weekly_off", "holiday"},
	} {
		_, err := create(c.date, c.kind)
		wantErr(t, err, "overtime_day_kind_mismatch")
		if got := errParams(err)["day_kind"]; got != c.want {
			t.Fatalf("%s as %s: names %v, want %s", c.date, c.kind, got, c.want)
		}
	}
	id := must(t)(create("2026-04-30", "holiday"))
	wantErr(t, f.hrm.UpdateOvertime(f.e, id, hrm.OvertimeUpdate{Version: 1, OvertimeFields: overtime("2026-04-30", "2", "0")}), "overtime_day_kind_mismatch")

	// Right when saved, wrong by the time it is sent: 29/04 became a holiday meanwhile.
	late := must(t)(create("2026-04-29", "weekday"))
	f.check(f.hrm.SaveHoliday(pay, f.c, hrm.Holiday{Date: "2026-04-29", Name: "Nghỉ bù"}))
	wantErr(t, f.rec.Transition(f.e, record.Ref{Type: "hrm.overtime_request", ID: late}, 1, record.Posted), "overtime_day_kind_mismatch")
}
