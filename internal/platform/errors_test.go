package platform_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"

	"github.com/taoworklabs/mmerp/internal/platform"
)

func TestErrorsUseCodeAndParams(t *testing.T) {
	_, api := humatest.New(t)
	type in struct {
		Body struct {
			Days int `json:"days" minimum:"1"`
		}
	}
	huma.Post(api, "/leave", func(ctx context.Context, i *in) (*struct{}, error) {
		return nil, &platform.Error{Status: http.StatusConflict, Code: "period_locked", Params: map[string]any{"date": "2026-03-31"}}
	})
	huma.Get(api, "/boom", func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		return nil, context.DeadlineExceeded
	})

	for _, c := range []struct {
		method, path, body string
		status             int
		want               string
	}{
		{"POST", "/leave", `{"days": 2}`, 409, `{"code":"period_locked","params":{"date":"2026-03-31"}}`},
		{"POST", "/leave", `{"days": 0}`, 422, `{"code":"invalid_request","params":{"fields":["body.days"]}}`},
		{"GET", "/boom", ``, 500, `{"code":"internal_error"}`},
	} {
		req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		api.Adapter().ServeHTTP(rec, req)
		if rec.Code != c.status {
			t.Errorf("%s %s: status %d, want %d", c.method, c.path, rec.Code, c.status)
		}
		var got, want any
		_ = json.Unmarshal(rec.Body.Bytes(), &got)
		_ = json.Unmarshal([]byte(c.want), &want)
		if gb, _ := json.Marshal(got); string(gb) != mustJSON(want) {
			t.Errorf("%s %s: body %s, want %s", c.method, c.path, rec.Body, c.want)
		}
	}
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

// The client gets only internal_error; the cause goes to the request's log.
func TestUnexpectedErrorIsLogged(t *testing.T) {
	_, api := humatest.New(t)
	huma.Get(api, "/boom", func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		return nil, errors.New("decrypt: no key 7")
	})
	var log bytes.Buffer
	req := httptest.NewRequest("GET", "/boom", nil)
	req = req.WithContext(platform.WithLogger(req.Context(), slog.New(slog.NewTextHandler(&log, nil))))
	rec := httptest.NewRecorder()
	api.Adapter().ServeHTTP(rec, req)
	if rec.Code != 500 || strings.Contains(rec.Body.String(), "decrypt") || !strings.Contains(log.String(), "decrypt: no key 7") {
		t.Fatalf("status %d, body %s, log %q", rec.Code, rec.Body, log.String())
	}
}

// Codes are API contract: pinned explicitly, never derived from Go's status text.
func TestNewHumaErrorCodes(t *testing.T) {
	for status, want := range map[int]string{
		400: "bad_request",
		401: "unauthenticated",
		403: "forbidden",
		404: "not_found",
		406: "not_acceptable",
		412: "precondition_failed",
		413: "request_too_large",
		415: "unsupported_media_type",
		422: "invalid_request",
		500: "internal_error",
		418: "unexpected_error",
	} {
		err := platform.NewHumaError(status, "ignored")
		if got := err.(*platform.Error).Code; got != want {
			t.Errorf("status %d: code %q, want %q", status, got, want)
		}
		if err.GetStatus() != status {
			t.Errorf("status %d: GetStatus %d", status, err.GetStatus())
		}
	}
}
