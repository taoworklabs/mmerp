package app_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taoworklabs/mmerp/internal/app"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// env builds what the app runs with; its job client only enqueues (see work).
func env(t *testing.T, pool *pgxpool.Pool, products []string) app.Env {
	t.Helper()
	e := app.Env{Pool: pool, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Products: products, Keys: pgtest.Keyring(),
		Files: platform.Files{Dir: t.TempDir()}}
	var err error
	if e.Jobs, err = app.NewJobs(e, app.Modules()); err != nil {
		t.Fatal(err)
	}
	return e
}

type client struct {
	t      *testing.T
	h      http.Handler
	cookie *http.Cookie
}

func (c *client) do(method, path, body string) *httptest.ResponseRecorder {
	c.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	return rec
}

func wantCode(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var body struct{ Code string }
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != status || body.Code != code {
		t.Fatalf("got %d %s, want %d %s", rec.Code, rec.Body, status, code)
	}
}

func auditActions(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var s string
	err := pool.QueryRow(t.Context(), `SELECT coalesce(string_agg(action || ':' || coalesce(actor_id::text, '-'), ',' ORDER BY id), '') FROM audit.log`).Scan(&s)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSessionLifecycle(t *testing.T) {
	pool := pgtest.New(t)
	if err := app.CreateAdmin(t.Context(), pool, "Admin", "Quản trị", "correct horse"); err != nil {
		t.Fatal(err)
	}
	c := &client{t: t, h: app.New(env(t, pool, []string{"hrm"}), app.Modules(), nil)}

	wantCode(t, c.do("GET", "/api/me", ""), 401, "unauthenticated")
	wantCode(t, c.do("POST", "/api/auth/login", `{"login":"admin","password":"wrong"}`), 401, "invalid_credentials")
	wantCode(t, c.do("POST", "/api/auth/login", `{"login":"nobody","password":"wrong"}`), 401, "invalid_credentials")

	rec := c.do("POST", "/api/auth/login", `{"login":"admin","password":"correct horse"}`)
	if rec.Code != 204 {
		t.Fatalf("login = %d %s", rec.Code, rec.Body)
	}
	c.cookie = rec.Result().Cookies()[0]
	if !c.cookie.HttpOnly || !c.cookie.Secure || c.cookie.SameSite != http.SameSiteLaxMode || c.cookie.Value == "" {
		t.Fatalf("cookie = %+v", c.cookie)
	}

	rec = c.do("GET", "/api/me", "")
	if rec.Code != 200 || rec.Header().Get("X-Authz-Version") != "1.1" {
		t.Fatalf("me = %d %v %s", rec.Code, rec.Header(), rec.Body)
	}
	var me struct {
		Login, Name, Locale, Timezone string
		AuthzVersion                  string `json:"authz_version"`
		Products                      []string
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &me)
	if me.Login != "Admin" || me.Locale != "vi" || me.Timezone != "Asia/Ho_Chi_Minh" || me.AuthzVersion != "1.1" || strings.Join(me.Products, ",") != "hrm" {
		t.Fatalf("me = %+v", me)
	}

	if rec := c.do("PATCH", "/api/me", `{"locale":"en"}`); rec.Code != 204 {
		t.Fatalf("set locale = %d %s", rec.Code, rec.Body)
	}
	wantCode(t, c.do("PATCH", "/api/me", `{"locale":"fr"}`), 422, "invalid_request")
	_ = json.Unmarshal(c.do("GET", "/api/me", "").Body.Bytes(), &me)
	if me.Locale != "en" {
		t.Fatalf("locale = %q", me.Locale)
	}

	rec = c.do("POST", "/api/auth/logout", "")
	if rec.Code != 204 || rec.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("logout = %d %v", rec.Code, rec.Result().Cookies())
	}
	// The old cookie is revoked server-side.
	wantCode(t, c.do("GET", "/api/me", ""), 401, "unauthenticated")

	if got := auditActions(t, pool); got != "iam.admin_created:-,iam.login_failed:-,iam.login_failed:-,iam.login:1,iam.logout:1" {
		t.Fatalf("audit = %s", got)
	}
}

func TestProductGateOnRoutes(t *testing.T) {
	sales := platform.Module{Name: "sales", Product: "sales", Routes: func(api huma.API) {
		public := map[string]any{platform.MetaPublic: true}
		huma.Register(api, huma.Operation{OperationID: "list", Method: "GET", Path: "/sales", Metadata: public},
			func(context.Context, *struct{}) (*struct{}, error) { return nil, nil })
		huma.Register(api, huma.Operation{OperationID: "create", Method: "POST", Path: "/sales", Metadata: public},
			func(context.Context, *struct{}) (*struct{}, error) { return nil, nil })
		huma.Register(api, huma.Operation{OperationID: "export", Method: "POST", Path: "/sales/export",
			Metadata: map[string]any{platform.MetaPublic: true, platform.MetaClass: platform.ClassExport}},
			func(context.Context, *struct{}) (*struct{}, error) { return nil, nil })
	}}
	c := &client{t: t, h: app.New(env(t, pgtest.New(t), []string{"hrm"}), []platform.Module{sales}, nil)}

	for path, method := range map[string]string{"/api/sales": "GET", "/api/sales/export": "POST"} {
		if rec := c.do(method, path, ""); rec.Code != 204 {
			t.Errorf("%s %s = %d %s", method, path, rec.Code, rec.Body)
		}
	}
	rec := c.do("POST", "/api/sales", "")
	wantCode(t, rec, 403, "product_not_enabled")
	if !strings.Contains(rec.Body.String(), `"product":"sales"`) {
		t.Fatalf("body = %s", rec.Body)
	}
}

func TestCheckProducts(t *testing.T) {
	if err := app.CheckProducts([]string{"hrm"}); err != nil {
		t.Fatal(err)
	}
	if err := app.CheckProducts([]string{"hrm", "nope"}); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("err = %v", err)
	}
}
