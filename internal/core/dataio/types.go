package dataio

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// Import is what a module registers to read rows from Excel into its data.
type Import struct {
	Kind    string // <module>.<data>, e.g. hrm.timesheet
	Product string
	// Header returns the column titles in the actor's language; the file's header must
	// match them, so a file made for other data or another period is refused.
	Header func(ctx context.Context, params json.RawMessage) ([]string, error)
	// Run validates every row (header excluded, each padded to the header's width; a blank
	// row is nil, so row numbers still match the file) and,
	// when none is wrong, writes them through the module's services. It runs in one
	// transaction that is rolled back if any row error is returned.
	Run func(ctx context.Context, params json.RawMessage, rows [][]string) ([]RowError, error)
}

// Export is what a module registers to write its data to Excel.
type Export struct {
	Kind    string
	Product string
	Run     func(ctx context.Context, params json.RawMessage) (Sheet, error)
	// Check says whether the actor may read the export now; every download of its file
	// asks again, so a right revoked since the export also closes the file.
	Check func(ctx context.Context, params json.RawMessage) error
}

// Sheet is one worksheet. A cell is a string (text) or an int64 or a
// decimal.Decimal (number).
type Sheet struct {
	Name   string // file name without extension
	Header []string
	Rows   [][]any
}

// RowError says what is wrong with a row: Row indexes the rows given to Run, and Code
// is translated as <product>.import.<code> with Params.
type RowError struct {
	Row    int
	Code   string
	Params map[string]any
}

// RowMessage is a row error as the requester reads it; Row is numbered as Excel shows it.
type RowMessage struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// Job is a background job, as listed in "my background jobs" or, for an administrator, among the system jobs.
type Job struct {
	ID   int64  `json:"id"`
	Kind string `json:"kind" doc:"e.g. dataio.import"`
	// The import or export kind, e.g. hrm.timesheet.
	Target     *string `json:"target"`
	State      string  `json:"state" enum:"queued,running,retrying,completed,failed"`
	Attempts   int     `json:"attempts"`
	CreatedAt  string  `json:"created_at" format:"date-time"`
	FinishedAt *string `json:"finished_at" format:"date-time"`
	// Rows written by a completed import.
	Rows *int `json:"rows"`
	// The rows of a failed import (import_rows_invalid), already translated.
	RowErrors []RowMessage `json:"row_errors" nullable:"false"`
	// The file of a completed export.
	FileID *string `json:"file_id"`
	// Why a failed or retrying job failed: an error code with its parameters, as for any API error.
	ErrorCode   *string        `json:"error_code"`
	ErrorParams map[string]any `json:"error_params"`
}

const (
	maxImportBytes = 5 << 20
	// ponytail: fixed ceilings; raise them when a customer needs bigger files.
	maxImportRows = 10_000
	maxRowErrors  = 100
)

var (
	ErrFileTooLarge  = &platform.Error{Status: http.StatusRequestEntityTooLarge, Code: "file_too_large", Params: map[string]any{"max_mb": maxImportBytes >> 20}}
	ErrInvalidParams = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_request", Params: map[string]any{"fields": []string{"params"}}}
	// The import file is gone (expired or never there) when its job runs.
	ErrFileGone    = &platform.Error{Status: http.StatusGone, Code: "import_file_gone"}
	ErrNotExcel    = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "import_not_excel"}
	ErrEmptyFile   = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "import_empty"}
	ErrTooManyRows = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "import_too_many_rows", Params: map[string]any{"max": maxImportRows}}
)

// errHeader names the first header cell that is not the expected title.
func errHeader(column int, expected, got string) error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "import_header", Params: map[string]any{"column": column, "expected": expected, "got": got}}
}

func errColumns(expected, got int) error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "import_columns", Params: map[string]any{"expected": expected, "got": got}}
}
