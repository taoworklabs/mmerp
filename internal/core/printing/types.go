package printing

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// Template is what a module registers to print one of its document types. Its print
// runs as the export printing.<Code>, with params {"id": …, "parts": […]}.
type Template struct {
	Code    string // <module>.<name>, e.g. hrm.contract
	Name    string // i18n key
	DocType string
	// product is the record type's, set by Register.
	product string
	// Blocks are the texts tenant administrators may edit; layouts draw them by key.
	Blocks []Block
	// Data reads the current print data of a record, one part per printable unit, in
	// print order. It is called only once the actor may print the record.
	Data func(ctx context.Context, id int64) ([]Part, error)
	// Layouts draw one part; version n is Layouts[n-1] and the last is current. A released
	// layout never changes: changing what a print shows, labels included, adds a version.
	Layouts []Layout
}

// Part is the data of one printable unit of a record, printed on pages of its own.
type Part struct {
	Key  int64 // 0 for a record printed whole; e.g. the employee of a payslip
	Data json.RawMessage
}

// Layout draws a part's data on p.
type Layout func(p *Page, data json.RawMessage) error

// params are those of a print export; no parts means all of them.
type params struct {
	ID    int64   `json:"id"`
	Parts []int64 `json:"parts"`
}

// Block is a named text of a template: plain text in which only Placeholders, written
// {name}, are filled in.
type Block struct {
	Key          string
	Label        string // i18n key
	Default      string // i18n key of the text until an administrator saves one
	Placeholders []string
}

// maxBlockText bounds the text of a block in one language.
const maxBlockText = 4000

// PrintTemplate is a template as its list shows it; version 0 means the default texts.
type PrintTemplate struct {
	Code        string  `json:"code" doc:"e.g. hrm.contract"`
	Name        string  `json:"name" doc:"In the caller's language"`
	Product     string  `json:"product"`
	Version     int     `json:"version"`
	SavedAt     *string `json:"saved_at" format:"date-time"`
	SavedByName *string `json:"saved_by_name"`
}

// PrintTemplateBlocks is a template with the current text of each block.
type PrintTemplateBlocks struct {
	PrintTemplate
	Blocks []PrintBlock `json:"blocks" nullable:"false"`
}

type PrintBlock struct {
	Key          string   `json:"key"`
	Label        string   `json:"label" doc:"In the caller's language"`
	Placeholders []string `json:"placeholders" nullable:"false"`
	Vi           string   `json:"vi"`
	En           string   `json:"en"`
}

// BlockText is the text of one block in both languages, as saved.
type BlockText struct {
	Key string `json:"key"`
	Vi  string `json:"vi" maxLength:"4000"`
	En  string `json:"en" maxLength:"4000"`
}

// errPlaceholder names a placeholder a block does not offer.
func errPlaceholder(block, name string) error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "print_placeholder_unknown", Params: map[string]any{"block": block, "name": name}}
}

// errBlocks: the blocks saved are not exactly the template's, or a text is too long.
var errBlocks = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_request", Params: map[string]any{"fields": []string{"blocks"}}}
