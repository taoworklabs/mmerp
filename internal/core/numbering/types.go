package numbering

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

var ErrInvalidPrefix = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_code_prefix"}

// CodeSeries is a code series as tenant administrators see it.
type CodeSeries struct {
	Code    string `json:"code" doc:"Translated as <code>.code_series"`
	Product string `json:"product"`
	Prefix  string `json:"prefix"`
	// The code the next entry left without one would get, typed codes aside.
	Next string `json:"next"`
	// edit.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}
