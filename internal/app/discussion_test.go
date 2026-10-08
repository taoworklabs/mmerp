package app_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/taoworklabs/mmerp/internal/app"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

type discussion struct {
	AllowedActions []string `json:"allowed_actions"`
	Items          []struct {
		AuthorName string `json:"author_name"`
		Body       string
	}
}

// The submitter of a leave request and their manager, who approves it, talk on the
// request; nobody else reads it, and nothing is added once cancelled or with the product off.
func TestLeaveDiscussion(t *testing.T) {
	pool := pgtest.New(t)
	if err := app.CreateAdmin(t.Context(), pool, "admin", "Admin", "correct horse"); err != nil {
		t.Fatal(err)
	}
	on := env(t, pool, []string{"hrm"})
	h := app.New(on, app.Modules(), nil)
	login := func(name string) *client {
		c := &client{t: t, h: h}
		c.cookie = c.do("POST", "/api/auth/login", fmt.Sprintf(`{"login":%q,"password":"correct horse"}`, name)).Result().Cookies()[0]
		return c
	}
	admin := login("admin")
	ok := func(c *client, method, path, body string, status int) []byte {
		t.Helper()
		rec := c.do(method, path, body)
		if rec.Code != status {
			t.Fatalf("%s %s = %d %s", method, path, rec.Code, rec.Body)
		}
		return rec.Body.Bytes()
	}
	id := func(c *client, path, body string) int64 {
		t.Helper()
		var out struct{ ID int64 }
		_ = json.Unmarshal(ok(c, "POST", path, body, 201), &out)
		return out.ID
	}
	read := func(c *client, path string) discussion {
		t.Helper()
		var d discussion
		_ = json.Unmarshal(ok(c, "GET", path, "", 200), &d)
		return d
	}

	company := id(admin, "/api/org-units", `{"parent_id":null,"kind":"company","name":"C"}`)
	dept := id(admin, "/api/org-units", fmt.Sprintf(`{"parent_id":%d,"kind":"department","name":"A"}`, company))
	id(admin, "/api/users/1/roles", `{"product":"hrm","role":"hr","org_unit_id":null}`)
	id(admin, "/api/users/1/roles", `{"product":"hrm","role":"leave_admin","org_unit_id":null}`)
	for _, u := range []string{"boss", "staff", "stranger"} {
		id(admin, "/api/users", fmt.Sprintf(`{"login":%q,"name":%q,"password":"correct horse"}`, u, u))
	}
	employee := func(code, login, manager string) int64 {
		return id(admin, "/api/hrm/employees", fmt.Sprintf(`{"code":%q,"full_name":%q,"date_of_birth":null,"gender":null,"phone":null,
			"email":null,"address":null,"org_unit_id":%d,"manager_id":%s,"user_login":%q,"hire_date":"2026-01-05","termination_date":null}`, code, login, dept, manager, login))
	}
	staff := employee("E2", "staff", fmt.Sprint(employee("E1", "boss", "null")))
	annual := id(admin, "/api/hrm/leave-types", `{"name":"Phép năm","deducts_balance":true,"paid":true,"active":true}`)
	ok(admin, "POST", fmt.Sprintf("/api/hrm/employees/%d/leave-balances/2026/adjustments", staff), `{"delta":"12","reason":"đầu năm"}`, 204)
	ok(admin, "PUT", "/api/approval-rules/hrm.leave_request", `{"steps":[{"approver":{"kind":"module"}}],"max_levels":3,"fallback_product":"hrm","fallback_role":"hr"}`, 204)

	staffC, bossC, stranger := login("staff"), login("boss"), login("stranger")
	lv := id(staffC, "/api/hrm/leaves", fmt.Sprintf(`{"leave_type_id":%d,"start_date":"2026-03-10","end_date":"2026-03-11","days":"1.5","reason":null}`, annual))
	docPath := fmt.Sprintf("/api/documents/hrm.leave_request/%d", lv)
	comments := fmt.Sprintf("/api/records/hrm.leave_request/%d/comments", lv)
	ok(staffC, "POST", docPath+"/transitions", `{"to":"posted","version":1}`, 204)

	ok(bossC, "POST", comments, `{"body":"Ai làm thay việc của em?"}`, 204)
	ok(staffC, "POST", comments, `{"body":"Chị Lan nhận.\nEm đã bàn giao."}`, 204)
	for _, c := range []*client{staffC, bossC} {
		d := read(c, comments)
		if len(d.Items) != 2 || d.Items[0].AuthorName != "boss" || d.Items[1].Body != "Chị Lan nhận.\nEm đã bàn giao." || !slices.Equal(d.AllowedActions, []string{"comment"}) {
			t.Fatalf("discussion = %+v", d)
		}
	}
	if a := auditActions(t, pool); strings.Count(a, "discussion.commented") != 2 {
		t.Fatalf("audit = %s", a)
	}

	wantCode(t, stranger.do("GET", comments, ""), 404, "not_found")
	wantCode(t, stranger.do("POST", comments, `{"body":"?"}`), 404, "not_found")
	wantCode(t, staffC.do("POST", comments, fmt.Sprintf(`{"body":%q}`, strings.Repeat("ạ", 4001))), 422, "comment_too_long")
	wantCode(t, staffC.do("POST", comments, `{"body":" \n "}`), 422, "invalid_request")
	ok(staffC, "POST", comments, fmt.Sprintf(`{"body":%q}`, strings.Repeat("ạ", 4000)), 204)

	offEnv := env(t, pool, nil)
	offEnv.Files = on.Files
	off := &client{t: t, h: app.New(offEnv, app.Modules(), nil), cookie: staffC.cookie}
	if d := read(off, comments); len(d.Items) != 3 || len(d.AllowedActions) != 0 {
		t.Fatalf("disabled product = %+v", d)
	}
	wantCode(t, off.do("POST", comments, `{"body":"?"}`), 403, "product_not_enabled")

	// Withdrawn, then cancelled after posting by HR: still read, closed to comments.
	ok(staffC, "POST", docPath+"/transitions", `{"to":"draft","version":1}`, 204)
	ok(admin, "DELETE", "/api/approval-rules/hrm.leave_request", "", 204)
	ok(staffC, "POST", docPath+"/transitions", `{"to":"posted","version":2}`, 204)
	ok(admin, "POST", docPath+"/transitions", `{"to":"cancelled","version":3}`, 204)
	if d := read(bossC, comments); len(d.Items) != 3 || len(d.AllowedActions) != 0 {
		t.Fatalf("cancelled = %+v", d)
	}
	wantCode(t, bossC.do("POST", comments, `{"body":"?"}`), 409, "document_not_editable")
}

// Deleting a draft takes its attachments and comments with it.
func TestDeletedDraftTakesAttachmentsAndComments(t *testing.T) {
	pool := pgtest.New(t)
	if err := app.CreateAdmin(t.Context(), pool, "admin", "Admin", "correct horse"); err != nil {
		t.Fatal(err)
	}
	c := &client{t: t, h: app.New(env(t, pool, []string{"hrm"}), app.Modules(), nil)}
	c.cookie = c.do("POST", "/api/auth/login", `{"login":"admin","password":"correct horse"}`).Result().Cookies()[0]
	ok := func(method, path, body string, status int) []byte {
		t.Helper()
		rec := c.do(method, path, body)
		if rec.Code != status {
			t.Fatalf("%s %s = %d %s", method, path, rec.Code, rec.Body)
		}
		return rec.Body.Bytes()
	}
	id := func(path, body string) int64 {
		t.Helper()
		var out struct{ ID int64 }
		_ = json.Unmarshal(ok("POST", path, body, 201), &out)
		return out.ID
	}
	company := id("/api/org-units", `{"parent_id":null,"kind":"company","name":"C"}`)
	dept := id("/api/org-units", fmt.Sprintf(`{"parent_id":%d,"kind":"department","name":"A"}`, company))
	id("/api/users/1/roles", `{"product":"hrm","role":"hr","org_unit_id":null}`)
	id("/api/users/1/roles", `{"product":"hrm","role":"leave_admin","org_unit_id":null}`)
	id("/api/hrm/employees", fmt.Sprintf(`{"code":"E1","full_name":"An","date_of_birth":null,"gender":null,"phone":null,
		"email":null,"address":null,"org_unit_id":%d,"manager_id":null,"user_login":"admin","hire_date":"2026-01-05","termination_date":null}`, dept))
	unpaid := id("/api/hrm/leave-types", `{"name":"Không lương","deducts_balance":false,"paid":false,"active":true}`)
	lv := id("/api/hrm/leaves", fmt.Sprintf(`{"leave_type_id":%d,"start_date":"2026-03-10","end_date":"2026-03-10","days":"1","reason":null}`, unpaid))
	files := fmt.Sprintf("/api/records/hrm.leave_request/%d/attachments", lv)
	comments := fmt.Sprintf("/api/records/hrm.leave_request/%d/comments", lv)
	if rec := upload(c, files, "a.pdf", pdf); rec.Code != 204 {
		t.Fatalf("upload = %d %s", rec.Code, rec.Body)
	}
	download := fmt.Sprintf("/api/attachments/%d", listAttachments(c, files).Items[0].ID)
	ok("POST", comments, `{"body":"nháp"}`, 204)

	ok("DELETE", fmt.Sprintf("/api/hrm/leaves/%d?version=1", lv), "", 204)
	wantCode(t, c.do("GET", files, ""), 404, "not_found")
	wantCode(t, c.do("GET", download, ""), 404, "not_found")
	wantCode(t, c.do("GET", comments, ""), 404, "not_found")
	var left int
	if err := pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM attachment.files) + (SELECT count(*) FROM discussion.comments)`).Scan(&left); err != nil || left != 0 {
		t.Fatalf("%d rows left %v", left, err)
	}
}

// An employee's papers (ID card, diplomas) go on their profile and show what the
// sensitive fields hold, so they need hrm.employee.sensitive on top of view or edit.
// Anyone who sees the profile talks on it.
func TestEmployeeAttachmentsAndDiscussion(t *testing.T) {
	f := newJobsFixture(t)
	grant := func(login string, roles ...string) *client {
		u := f.id(f.admin, "/api/users", fmt.Sprintf(`{"login":%q,"name":%q,"password":"correct horse"}`, login, login))
		for _, r := range roles {
			f.id(f.admin, fmt.Sprintf("/api/users/%d/roles", u), fmt.Sprintf(`{"product":"hrm","role":%q,"org_unit_id":%d}`, r, f.p))
		}
		return f.login(login)
	}
	clerk := grant("clerk", "hr", "sensitive_viewer")
	reader := grant("reader", "viewer", "sensitive_viewer")
	view := grant("view", "viewer")
	stranger := grant("stranger")
	files := fmt.Sprintf("/api/records/hrm.employee/%d/attachments", f.e1)
	comments := fmt.Sprintf("/api/records/hrm.employee/%d/comments", f.e1)
	png := []byte("\x89PNG\r\n\x1a\n0000")

	if rec := upload(clerk, files, "cccd.png", png); rec.Code != 204 {
		t.Fatalf("clerk upload = %d %s", rec.Code, rec.Body)
	}
	if l := listAttachments(clerk, files); len(l.Items) != 1 || !slices.Equal(l.AllowedActions, []string{"attach"}) || !slices.Equal(l.Items[0].AllowedActions, []string{"delete"}) {
		t.Fatalf("clerk sees %+v", l)
	}
	l := listAttachments(reader, files)
	if len(l.Items) != 1 || len(l.AllowedActions) != 0 || len(l.Items[0].AllowedActions) != 0 {
		t.Fatalf("reader sees %+v", l)
	}
	download := fmt.Sprintf("/api/attachments/%d", l.Items[0].ID)
	if rec := reader.do("GET", download, ""); rec.Code != 200 {
		t.Fatalf("reader download = %d", rec.Code)
	}
	wantCode(t, upload(reader, files, "x.png", png), 403, "forbidden")
	wantCode(t, reader.do("DELETE", download, ""), 403, "forbidden")

	// HR without the sensitive role, and a plain viewer, see the profile but not its papers.
	for _, c := range []*client{f.hr, view} {
		if l := listAttachments(c, files); !l.Hidden || len(l.Items) != 0 {
			t.Fatalf("sees %+v", l)
		}
		wantCode(t, c.do("GET", download, ""), 404, "not_found")
	}
	wantCode(t, upload(f.hr, files, "x.png", png), 404, "not_found")
	wantCode(t, stranger.do("GET", files, ""), 404, "not_found")
	wantCode(t, stranger.do("GET", comments, ""), 404, "not_found")

	f.ok(view, "POST", comments, `{"body":"Bản CCCD đã hết hạn."}`, 204)
	if rec := f.hr.do("GET", comments, ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), "Bản CCCD đã hết hạn.") {
		t.Fatalf("hr reads = %d %s", rec.Code, rec.Body)
	}
}
