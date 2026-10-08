package printing

import (
	"context"
	"encoding/json"
)

// Template is what a module registers to print one of its document types. Its print
// runs as the export printing.<Code>, with params {"id": …, "parts": […]}.
type Template struct {
	Code    string // <module>.<name>, e.g. hrm.contract
	DocType string
	Product string
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
