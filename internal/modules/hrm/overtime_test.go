package hrm_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
)

func overtime(date, day, night string) hrm.OvertimeFields {
	return hrm.OvertimeFields{Date: date, DayKind: "weekday", DayHours: day, NightHours: night}
}

func (f *leaveFixture) postedOvertimes() int {
	f.t.Helper()
	l, err := f.hrm.Overtimes(f.hr, hrm.OvertimeFilter{Status: "posted", Sort: "-date", Page: 1, PageSize: 20})
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
			Record: f.rec, Actor: f.hr, Admin: f.admin, LegalEntity: f.c,
			Create: func(ctx context.Context, date string) (record.Ref, error) {
				return ref(f.hrm.CreateOvertime(ctx, hrm.NewOvertime{EmployeeID: &f.emp, OvertimeFields: overtime(date, "2", "0")}))
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return f.hrm.UpdateOvertime(ctx, r.ID, hrm.OvertimeUpdate{Version: version, OvertimeFields: overtime(date, "2", "0")})
			},
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
