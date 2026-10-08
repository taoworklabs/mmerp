package platform

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
)

// Every importer of platform gets coded errors from the framework too, so no
// handler can leak Huma's default body with internal messages.
func init() { huma.NewError = NewHumaError }

// Error is what the API returns: a stable code with parameters, never a sentence.
// The frontend translates the code.
type Error struct {
	Status int            `json:"-"`
	Code   string         `json:"code"`
	Params map[string]any `json:"params,omitempty"`
}

func (e *Error) Error() string  { return e.Code }
func (e *Error) GetStatus() int { return e.Status }

// Is matches by code, so an error carrying params still matches its sentinel.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Code == e.Code
}

// Codes are part of the API contract, so they are listed here rather than
// derived from status text.
var frameworkCodes = map[int]string{
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
}

// NewHumaError builds framework errors (validation, unexpected) in the same shape.
// A returned *Error, or one passed in errs, reaches the client as is.
func NewHumaError(status int, _ string, errs ...error) huma.StatusError {
	for _, err := range errs {
		if e, ok := errors.AsType[*Error](err); ok {
			return e
		}
	}
	code, ok := frameworkCodes[status]
	if !ok {
		// A framework status nobody mapped yet; add it above rather than relying on this.
		code = "unexpected_error"
	}
	e := &Error{Status: status, Code: code}
	var fields []string
	for _, err := range errs {
		if d, ok := errors.AsType[*huma.ErrorDetail](err); ok && d.Location != "" {
			fields = append(fields, d.Location)
		}
	}
	if fields != nil {
		e.Code = "invalid_request"
		e.Params = map[string]any{"fields": fields}
	}
	return e
}

// UploadLimit refuses, without parsing it, an upload whose announced length is over
// max, with the operation's own error (e.g. file_too_large) rather than the framework's.
// Huma does not bound multipart bodies; the app caps every body for those that hide
// their length.
func UploadLimit(max int64, tooLarge *Error) huma.Middlewares {
	return huma.Middlewares{func(ctx huma.Context, next func(huma.Context)) {
		if n, err := strconv.ParseInt(ctx.Header("Content-Length"), 10, 64); err == nil && n > max {
			// Read to the end (the app's cap bounds it): a browser still sending the body
			// may see a reset connection instead of an answer given before it finished.
			_, _ = io.Copy(io.Discard, ctx.BodyReader())
			ctx.SetHeader("Content-Type", "application/json")
			ctx.SetStatus(tooLarge.Status)
			_ = json.NewEncoder(ctx.BodyWriter()).Encode(tooLarge)
			return
		}
		next(ctx)
	}}
}

// WriteError answers a plain HTTP handler (outside Huma) in the API's error shape.
func WriteError(w http.ResponseWriter, e *Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.Status)
	_ = json.NewEncoder(w).Encode(e)
}

// Shared by every module, so the frontend has one translation for each.
var (
	ErrForbidden = &Error{Status: http.StatusForbidden, Code: "forbidden"}
	ErrNotFound  = &Error{Status: http.StatusNotFound, Code: "not_found"}
)
