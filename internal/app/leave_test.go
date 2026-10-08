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

func TestLeaveAPI(t *testing.T) {
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
	type leave struct {
		Status         string
		Version        int32
		Balance        *string
		AllowedActions []string `json:"allowed_actions"`
	}
	get := func(c *client, path string) leave {
		t.Helper()
		var l leave
		rec := c.do("GET", path, "")
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &l) != nil {
			t.Fatalf("GET %s = %d %s", path, rec.Code, rec.Body)
		}
		return l
	}

	company := id("/api/org-units", `{"parent_id":null,"kind":"company","name":"C"}`)
	dept := id("/api/org-units", fmt.Sprintf(`{"parent_id":%d,"kind":"department","name":"A"}`, company))
	id("/api/users/1/roles", `{"product":"hrm","role":"hr","org_unit_id":null}`)
	id("/api/users/1/roles", `{"product":"hrm","role":"leave_admin","org_unit_id":null}`)
	emp := id("/api/hrm/employees", fmt.Sprintf(`{"code":"E1","full_name":"An","date_of_birth":null,"gender":null,"phone":null,
		"email":null,"address":null,"org_unit_id":%d,"manager_id":null,"user_login":"admin","hire_date":"2026-01-05","termination_date":null}`, dept))
	annual := id("/api/hrm/leave-types", `{"name":"Phép năm","deducts_balance":true,"paid":true,"active":true}`)
	ok("POST", fmt.Sprintf("/api/hrm/employees/%d/leave-balances/2026/adjustments", emp), `{"delta":"12","reason":"đầu năm"}`, 204)
	wantCode(t, c.do("POST", fmt.Sprintf("/api/hrm/employees/%d/leave-balances/2026/adjustments", emp), `{"delta":"1","reason":""}`), 422, "invalid_request")

	if rec := c.do("GET", "/api/hrm/leaves/actions", ""); !strings.Contains(rec.Body.String(), `"create"`) || !strings.Contains(rec.Body.String(), fmt.Sprintf(`"self_employee_id":%d`, emp)) {
		t.Fatalf("actions = %s", rec.Body)
	}
	lv := id("/api/hrm/leaves", fmt.Sprintf(`{"leave_type_id":%d,"start_date":"2026-03-10","end_date":"2026-03-11","days":"1.5","reason":null}`, annual))
	path := fmt.Sprintf("/api/hrm/leaves/%d", lv)
	docPath := fmt.Sprintf("/api/documents/hrm.leave_request/%d", lv)
	if l := get(c, path); l.Status != "draft" || l.Version != 1 || *l.Balance != "12" || !slices.Equal(l.AllowedActions, []string{"edit", "delete", "submit"}) {
		t.Fatalf("draft = %+v", l)
	}

	// The only possible approver is the submitter: refused, never auto-approved.
	ok("PUT", "/api/approval-rules/hrm.leave_request", `{"steps":[{"approver":{"kind":"module"}}],"max_levels":3,"fallback_product":"hrm","fallback_role":"hr"}`, 204)
	wantCode(t, c.do("POST", docPath+"/transitions", `{"to":"posted","version":1}`), 422, "no_approver")
	ok("DELETE", "/api/approval-rules/hrm.leave_request", "", 204)

	ok("POST", docPath+"/transitions", `{"to":"posted","version":1}`, 204)
	if l := get(c, path); l.Status != "posted" || *l.Balance != "10.5" || !slices.Equal(l.AllowedActions, []string{"cancel"}) {
		t.Fatalf("posted = %+v", l)
	}
	wantCode(t, c.do("POST", docPath+"/transitions", `{"to":"cancelled","version":1}`), 409, "version_conflict")
	if h := string(ok("GET", docPath+"/history", "", 200)); !strings.Contains(h, "record.created") || !strings.Contains(h, "record.transitioned") {
		t.Fatalf("history = %s", h)
	}
	if a := string(ok("GET", docPath+"/approval", "", 200)); strings.Contains(a, `"instance"`) {
		t.Fatalf("approval = %s", a)
	}

	ok("PUT", fmt.Sprintf("/api/period-locks/%d", company), `{"locked_until":"2026-03-31"}`, 204)
	if l := get(c, path); len(l.AllowedActions) != 0 {
		t.Fatalf("locked = %+v", l)
	}
	wantCode(t, c.do("POST", docPath+"/transitions", `{"to":"cancelled","version":2}`), 409, "period_locked")
	wantCode(t, c.do("PUT", fmt.Sprintf("/api/period-locks/%d", dept), `{"locked_until":"2026-03-31"}`), 404, "not_found")

	// Someone who cannot view the request learns nothing about it from a write.
	id("/api/users", `{"login":"stranger","name":"Stranger","password":"correct horse"}`)
	stranger := &client{t: t, h: c.h}
	stranger.cookie = stranger.do("POST", "/api/auth/login", `{"login":"stranger","password":"correct horse"}`).Result().Cookies()[0]
	wantCode(t, stranger.do("POST", docPath+"/transitions", `{"to":"draft","version":2}`), 404, "not_found")

	off := &client{t: t, h: app.New(env(t, pool, nil), app.Modules(), nil), cookie: c.cookie}
	ok("PUT", fmt.Sprintf("/api/period-locks/%d", company), `{"locked_until":null}`, 204)
	if l := get(off, path); len(l.AllowedActions) != 0 {
		t.Fatalf("disabled product = %+v", l)
	}
	wantCode(t, off.do("POST", docPath+"/transitions", `{"to":"cancelled","version":2}`), 403, "product_not_enabled")
}
