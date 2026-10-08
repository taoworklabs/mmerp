package app_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/taoworklabs/mmerp/internal/app"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

func TestHealthz(t *testing.T) {
	pool := pgtest.New(t)
	h := app.New(env(t, pool, nil), app.Modules(), nil)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz = %d", rec.Code)
	}

	pool.Close()
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "database_unreachable") {
		t.Fatalf("healthz with closed pool = %d %q", rec.Code, rec.Body)
	}
}

func TestServesFrontend(t *testing.T) {
	web := fstest.MapFS{
		"index.html":    {Data: []byte("index")},
		"assets/app.js": {Data: []byte("js")},
	}
	h := app.New(env(t, pgtest.New(t), nil), app.Modules(), web)
	for _, c := range []struct {
		path   string
		status int
		body   string
	}{
		{"/", 200, "index"},
		{"/hrm/leaves/42", 200, "index"},
		{"/assets/app.js", 200, "js"},
		{"/api/nope", 404, ""},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", c.path, nil))
		if rec.Code != c.status || (c.body != "" && rec.Body.String() != c.body) {
			t.Errorf("GET %s = %d %q, want %d %q", c.path, rec.Code, rec.Body, c.status, c.body)
		}
	}
}
