package app_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/taoworklabs/mmerp/internal/app"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

type notification struct {
	ID         int64
	Kind       string
	RecordType *string `json:"record_type"`
	RecordID   *int64  `json:"record_id"`
	Label      *string
	ActorName  *string `json:"actor_name"`
	JobID      *int64  `json:"job_id"`
	Read       bool
}

type notificationPage struct {
	Items      []notification
	NextBefore *int64 `json:"next_before"`
}

func notifications(c *client) notificationPage {
	c.t.Helper()
	rec := c.do("GET", "/api/notifications", "")
	var p notificationPage
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &p) != nil {
		c.t.Fatalf("notifications = %d %s", rec.Code, rec.Body)
	}
	return p
}

func unread(c *client) int {
	c.t.Helper()
	rec := c.do("GET", "/api/notifications/unread-count", "")
	var out struct{ Count int }
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &out) != nil {
		c.t.Fatalf("unread = %d %s", rec.Code, rec.Body)
	}
	return out.Count
}

// leaveFixture is a company where staff's leave requests go to their manager, boss, then to HR.
type leaveFixture struct {
	t                              *testing.T
	admin, boss, staff, hr2        *client
	annual, staffEmp, bossEm, dept int64
	ok                             func(c *client, method, path, body string, status int) []byte
	id                             func(c *client, path, body string) int64
	login                          func(name string) *client
	products                       func([]string) *client
}

func newLeaveFixture(t *testing.T, rule string) *leaveFixture {
	pool := pgtest.New(t)
	if err := app.CreateAdmin(t.Context(), pool, "admin", "Admin", "correct horse"); err != nil {
		t.Fatal(err)
	}
	on := env(t, pool, []string{"hrm"})
	h := app.New(on, app.Modules(), nil)
	f := &leaveFixture{t: t}
	f.login = func(name string) *client {
		c := &client{t: t, h: h}
		c.cookie = c.do("POST", "/api/auth/login", fmt.Sprintf(`{"login":%q,"password":"correct horse"}`, name)).Result().Cookies()[0]
		return c
	}
	f.ok = func(c *client, method, path, body string, status int) []byte {
		t.Helper()
		rec := c.do(method, path, body)
		if rec.Code != status {
			t.Fatalf("%s %s = %d %s", method, path, rec.Code, rec.Body)
		}
		return rec.Body.Bytes()
	}
	f.id = func(c *client, path, body string) int64 {
		t.Helper()
		var out struct{ ID int64 }
		_ = json.Unmarshal(f.ok(c, "POST", path, body, 201), &out)
		return out.ID
	}
	f.products = func(products []string) *client {
		e := env(t, pool, products)
		e.Files = on.Files
		return &client{t: t, h: app.New(e, app.Modules(), nil)}
	}
	f.admin = f.login("admin")
	company := f.id(f.admin, "/api/org-units", `{"parent_id":null,"kind":"company","name":"C"}`)
	f.dept = f.id(f.admin, "/api/org-units", fmt.Sprintf(`{"parent_id":%d,"kind":"department","name":"A"}`, company))
	dept := f.dept
	f.id(f.admin, "/api/users/1/roles", `{"product":"hrm","role":"hr","org_unit_id":null}`)
	f.id(f.admin, "/api/users/1/roles", `{"product":"hrm","role":"leave_admin","org_unit_id":null}`)
	for _, u := range []string{"boss", "staff", "hr2"} {
		f.id(f.admin, "/api/users", fmt.Sprintf(`{"login":%q,"name":%q,"password":"correct horse"}`, u, u))
	}
	f.id(f.admin, "/api/users/4/roles", `{"product":"hrm","role":"hr","org_unit_id":null}`)
	employee := func(code, login, manager string) int64 {
		return f.id(f.admin, "/api/hrm/employees", fmt.Sprintf(`{"code":%q,"full_name":%q,"date_of_birth":null,"gender":null,"phone":null,
			"email":null,"address":null,"org_unit_id":%d,"manager_id":%s,"user_login":%q,"hire_date":"2026-01-05","termination_date":null}`, code, login, dept, manager, login))
	}
	f.bossEm = employee("E1", "boss", "null")
	f.staffEmp = employee("E2", "staff", fmt.Sprint(f.bossEm))
	f.annual = f.id(f.admin, "/api/hrm/leave-types", `{"name":"Phép năm","deducts_balance":true,"paid":true,"active":true}`)
	f.ok(f.admin, "POST", fmt.Sprintf("/api/hrm/employees/%d/leave-balances/2026/adjustments", f.staffEmp), `{"delta":"12","reason":"đầu năm"}`, 204)
	f.ok(f.admin, "PUT", "/api/approval-rules/hrm.leave_request", rule, 204)
	f.boss, f.staff, f.hr2 = f.login("boss"), f.login("staff"), f.login("hr2")
	return f
}

// submit sends a new leave request of staff and returns its id.
func (f *leaveFixture) submit(start string) int64 {
	lv := f.id(f.staff, "/api/hrm/leaves", fmt.Sprintf(`{"leave_type_id":%d,"start_date":%q,"end_date":%q,"days":"1","reason":"Việc riêng"}`, f.annual, start, start))
	f.ok(f.staff, "POST", fmt.Sprintf("/api/documents/hrm.leave_request/%d/transitions", lv), `{"to":"posted","version":1}`, 204)
	return lv
}

// Every approver of a step that starts hears of it: the first step on sending, the next
// one once the first is approved, a reassigned step. Nobody hears of their own action.
func TestApprovalRequestedNotifications(t *testing.T) {
	f := newLeaveFixture(t, `{"steps":[{"approver":{"kind":"module"}},{"approver":{"kind":"role","product":"hrm","role":"hr"}}],
		"max_levels":3,"fallback_product":"hrm","fallback_role":"hr"}`)
	lv := f.submit("2026-03-10")

	if n := unread(f.boss); n != 1 {
		t.Fatalf("boss unread = %d", n)
	}
	for _, c := range []*client{f.staff, f.admin, f.hr2} {
		if n := unread(c); n != 0 {
			t.Fatalf("unread = %d before step 2", n)
		}
	}
	p := notifications(f.boss)
	if len(p.Items) != 1 || p.Items[0].Kind != "approval_requested" || *p.Items[0].RecordType != "hrm.leave_request" || *p.Items[0].RecordID != lv ||
		p.Items[0].Label == nil || *p.Items[0].ActorName != "staff" || p.Items[0].Read || p.NextBefore != nil {
		t.Fatalf("boss sees %+v", p)
	}

	f.ok(f.boss, "POST", "/api/approvals/1/approve", `{"step":1}`, 204)
	for _, c := range []*client{f.admin, f.hr2} {
		if n := unread(c); n != 1 {
			t.Fatalf("step 2 unread = %d", n)
		}
	}
	if n := unread(f.staff); n != 0 {
		t.Fatalf("staff hears of a middle step: %d", n)
	}
	// Reassigned by admin, the fallback: hr2 hears again, admin never of their own action.
	f.ok(f.admin, "POST", "/api/approvals/1/reassign", `{"step":2,"login":"hr2"}`, 204)
	if n, m := unread(f.hr2), unread(f.admin); n != 2 || m != 1 {
		t.Fatalf("after reassign hr2 = %d, admin = %d", n, m)
	}

	// Read one, then all; nobody else's can be marked.
	id := notifications(f.boss).Items[0].ID
	wantCode(t, f.staff.do("POST", fmt.Sprintf("/api/notifications/%d/read", id), ""), 404, "not_found")
	f.ok(f.boss, "POST", fmt.Sprintf("/api/notifications/%d/read", id), "", 204)
	if n := unread(f.boss); n != 0 || !notifications(f.boss).Items[0].Read {
		t.Fatalf("after read %d", n)
	}
	f.ok(f.hr2, "POST", "/api/notifications/read-all", "", 204)
	if n := unread(f.hr2); n != 0 {
		t.Fatalf("after read-all %d", n)
	}

	// A disabled product's notifications still read.
	off := f.products(nil)
	off.cookie = f.boss.cookie
	if p := notifications(off); len(p.Items) != 1 || p.Items[0].RecordID == nil {
		t.Fatalf("product off: %+v", p)
	}
	f.ok(off, "POST", "/api/notifications/read-all", "", 204)

	// Boss no longer manages staff, so no longer sees the request: its notification stays,
	// emptied of what names the record.
	f.ok(f.admin, "PUT", fmt.Sprintf("/api/hrm/employees/%d", f.staffEmp), fmt.Sprintf(`{"code":"E2","full_name":"staff","date_of_birth":null,"gender":null,"phone":null,
		"email":null,"address":null,"org_unit_id":%d,"manager_id":null,"user_login":"staff","hire_date":"2026-01-05","termination_date":null}`, f.dept), 204)
	p = notifications(f.boss)
	if len(p.Items) != 1 || p.Items[0].RecordID != nil || p.Items[0].Label != nil || p.Items[0].ActorName != nil || *p.Items[0].RecordType != "hrm.leave_request" {
		t.Fatalf("hidden: %+v", p)
	}
	wantCode(t, f.boss.do("GET", fmt.Sprintf("/api/hrm/leaves/%d", lv), ""), 404, "not_found")
}

// The list pages by id, newest first.
func TestNotificationPages(t *testing.T) {
	f := newLeaveFixture(t, `{"steps":[{"approver":{"kind":"module"}}],"max_levels":3,"fallback_product":"hrm","fallback_role":"hr"}`)
	f.ok(f.admin, "POST", fmt.Sprintf("/api/hrm/employees/%d/leave-balances/2026/adjustments", f.staffEmp), `{"delta":"100","reason":"thử"}`, 204)
	for d := 1; d <= 52; d++ {
		f.submit(fmt.Sprintf("2026-%02d-%02d", 3+d/28, 1+d%28))
	}
	p := notifications(f.boss)
	if len(p.Items) != 50 || p.NextBefore == nil || p.Items[0].ID < p.Items[49].ID {
		t.Fatalf("page 1: %d items, next %v", len(p.Items), p.NextBefore)
	}
	rec := f.boss.do("GET", fmt.Sprintf("/api/notifications?before=%d", *p.NextBefore), "")
	var p2 notificationPage
	_ = json.Unmarshal(rec.Body.Bytes(), &p2)
	if len(p2.Items) != 2 || p2.NextBefore != nil || p2.Items[0].ID >= p.Items[49].ID {
		t.Fatalf("page 2: %+v", p2)
	}
}

// The submitter hears of the outcome only: approved at the last step, or rejected,
// without the reason, which stays on the document.
func TestApprovalOutcomeNotifications(t *testing.T) {
	f := newLeaveFixture(t, `{"steps":[{"approver":{"kind":"module"}},{"approver":{"kind":"role","product":"hrm","role":"hr"}}],
		"max_levels":3,"fallback_product":"hrm","fallback_role":"hr"}`)
	approved := f.submit("2026-03-10")
	f.ok(f.boss, "POST", "/api/approvals/1/approve", `{"step":1}`, 204)
	if n := unread(f.staff); n != 0 {
		t.Fatalf("middle step told staff: %d", n)
	}
	f.ok(f.hr2, "POST", "/api/approvals/1/approve", `{"step":2}`, 204)
	p := notifications(f.staff)
	if len(p.Items) != 1 || p.Items[0].Kind != "approval_approved" || *p.Items[0].RecordID != approved || *p.Items[0].ActorName != "hr2" {
		t.Fatalf("approved: %+v", p)
	}
	if n := unread(f.hr2); n != 1 {
		t.Fatalf("hr2 told of their own approval: %d", n)
	}

	rejected := f.submit("2026-03-12")
	rec := f.boss.do("POST", "/api/approvals/2/reject", `{"step":1,"reason":"Trùng lịch họp quý"}`)
	if rec.Code != 204 {
		t.Fatalf("reject = %d %s", rec.Code, rec.Body)
	}
	raw := f.ok(f.staff, "GET", "/api/notifications", "", 200)
	p = notifications(f.staff)
	if len(p.Items) != 2 || p.Items[0].Kind != "approval_rejected" || *p.Items[0].RecordID != rejected || strings.Contains(string(raw), "Trùng lịch") {
		t.Fatalf("rejected: %s", raw)
	}
}

// A mention tells the mentioned user when they may view the record; anyone else, or a
// login nobody has, is skipped without an error, so the writer learns nothing of who sees it.
func TestMentions(t *testing.T) {
	f := newLeaveFixture(t, `{"steps":[{"approver":{"kind":"module"}}],"max_levels":3,"fallback_product":"hrm","fallback_role":"hr"}`)
	f.id(f.admin, "/api/users", `{"login":"stranger","name":"Người ngoài","password":"correct horse"}`)
	stranger := f.login("stranger")
	lv := f.id(f.staff, "/api/hrm/leaves", fmt.Sprintf(`{"leave_type_id":%d,"start_date":"2026-03-10","end_date":"2026-03-10","days":"1","reason":null}`, f.annual))
	comments := fmt.Sprintf("/api/records/hrm.leave_request/%d/comments", lv)

	var who []struct{ Login, Name string }
	_ = json.Unmarshal(f.ok(f.staff, "GET", fmt.Sprintf("/api/records/hrm.leave_request/%d/mentionable", lv), "", 200), &who)
	logins := []string{}
	for _, u := range who {
		logins = append(logins, u.Login)
	}
	if strings.Join(logins, ",") != "admin,boss,hr2" {
		t.Fatalf("mentionable = %v", logins)
	}
	wantCode(t, stranger.do("GET", fmt.Sprintf("/api/records/hrm.leave_request/%d/mentionable", lv), ""), 404, "not_found")

	f.ok(f.staff, "POST", comments, `{"body":"@BOSS anh xem giúp, @stranger và @nobody nữa. Cảm ơn @staff, @hr2."}`, 204)
	p := notifications(f.boss)
	if len(p.Items) != 1 || p.Items[0].Kind != "mentioned" || *p.Items[0].RecordID != lv || *p.Items[0].ActorName != "staff" {
		t.Fatalf("boss: %+v", p)
	}
	if n := unread(f.hr2); n != 1 {
		t.Fatalf("hr2 = %d", n)
	}
	for _, c := range []*client{stranger, f.staff, f.admin} {
		if n := unread(c); n != 0 {
			t.Fatalf("told %d", n)
		}
	}
	// Mentioned twice in one comment, told once.
	f.ok(f.boss, "POST", comments, `{"body":"@staff ok, @staff."}`, 204)
	if n := unread(f.staff); n != 1 {
		t.Fatalf("staff = %d", n)
	}
}
