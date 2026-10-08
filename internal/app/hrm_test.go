package app_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/taoworklabs/mmerp/internal/app"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

func TestEmployeeAPI(t *testing.T) {
	pool := pgtest.New(t)
	if err := app.CreateAdmin(t.Context(), pool, "admin", "Admin", "correct horse"); err != nil {
		t.Fatal(err)
	}
	c := &client{t: t, h: app.New(env(t, pool, []string{"hrm"}), app.Modules(), nil)}
	c.cookie = c.do("POST", "/api/auth/login", `{"login":"admin","password":"correct horse"}`).Result().Cookies()[0]
	id := func(path, body string) int64 {
		t.Helper()
		rec := c.do("POST", path, body)
		var out struct{ ID int64 }
		if rec.Code != 201 || json.Unmarshal(rec.Body.Bytes(), &out) != nil {
			t.Fatalf("POST %s = %d %s", path, rec.Code, rec.Body)
		}
		return out.ID
	}

	company := id("/api/org-units", `{"parent_id":null,"kind":"company","name":"C","tax_code":"0101"}`)
	dept := id("/api/org-units", fmt.Sprintf(`{"parent_id":%d,"kind":"department","name":"A"}`, company))
	id("/api/users/1/roles", `{"product":"hrm","role":"hr","org_unit_id":null}`)
	id("/api/users/1/roles", `{"product":"hrm","role":"sensitive_viewer","org_unit_id":null}`)

	emp := id("/api/hrm/employees", fmt.Sprintf(`{"code":"E1","full_name":"An","date_of_birth":null,"gender":null,"phone":null,
		"email":null,"address":null,"org_unit_id":%d,"manager_id":null,"user_login":null,"hire_date":"2026-01-05",
		"termination_date":null,"sensitive":{"national_id":"079123456789"}}`, dept))

	rec := c.do("GET", "/api/hrm/employees", "")
	if rec.Code != 200 {
		t.Fatalf("list = %d %s", rec.Code, rec.Body)
	}
	rec = c.do("GET", fmt.Sprintf("/api/hrm/employees/%d/sensitive/national_id", emp), "")
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-store" || !json.Valid(rec.Body.Bytes()) {
		t.Fatalf("reveal = %d %v %s", rec.Code, rec.Header(), rec.Body)
	}
	wantCode(t, c.do("GET", fmt.Sprintf("/api/hrm/employees/%d/sensitive/password", emp), ""), 422, "invalid_request")
	wantCode(t, c.do("GET", "/api/hrm/employees/999", ""), 404, "not_found")
	// The static route wins over /hrm/employees/{id}.
	if rec := c.do("GET", "/api/hrm/employees/actions", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"create"`) {
		t.Fatalf("actions = %d %s", rec.Code, rec.Body)
	}

	off := &client{t: t, h: app.New(env(t, pool, nil), app.Modules(), nil), cookie: c.cookie}
	if rec := off.do("GET", "/api/hrm/employees", ""); rec.Code != 200 {
		t.Fatalf("disabled product list = %d", rec.Code)
	}
	wantCode(t, off.do("PUT", fmt.Sprintf("/api/hrm/employees/%d", emp), `{}`), 403, "product_not_enabled")
}
