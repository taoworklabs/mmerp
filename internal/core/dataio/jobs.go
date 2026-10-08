package dataio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"

	"github.com/taoworklabs/mmerp/internal/core/dataio/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type importArgs struct {
	Target  string          `json:"target"`
	Product string          `json:"product"`
	FileID  string          `json:"file_id"`
	Params  json.RawMessage `json:"params"`
}

func (importArgs) Kind() string { return "dataio.import" }

func (a importArgs) Spec() platform.JobSpec {
	return platform.JobSpec{Product: a.Product, Class: platform.ClassWrite, Notify: true}
}

// A user waits for an import or export: a few quick retries, not River's 25 over weeks.
func (importArgs) InsertOpts() river.InsertOpts { return river.InsertOpts{MaxAttempts: 3} }

type exportArgs struct {
	Target  string          `json:"target"`
	Product string          `json:"product"`
	Params  json.RawMessage `json:"params"`
}

func (exportArgs) Kind() string { return "dataio.export" }

func (a exportArgs) Spec() platform.JobSpec {
	return platform.JobSpec{Product: a.Product, Class: platform.ClassExport, Notify: true}
}

func (exportArgs) InsertOpts() river.InsertOpts { return river.InsertOpts{MaxAttempts: 3} }

type cleanupArgs struct{}

func (cleanupArgs) Kind() string           { return "dataio.cleanup" }
func (cleanupArgs) Spec() platform.JobSpec { return platform.JobSpec{System: true} }

type importWorker struct {
	river.WorkerDefaults[importArgs]
	s *Service
}

func (w *importWorker) Work(ctx context.Context, j *river.Job[importArgs]) error {
	return w.s.runImport(ctx, j)
}

type exportWorker struct {
	river.WorkerDefaults[exportArgs]
	s *Service
}

func (w *exportWorker) Work(ctx context.Context, j *river.Job[exportArgs]) error {
	return w.s.runExport(ctx, j)
}

type cleanupWorker struct {
	river.WorkerDefaults[cleanupArgs]
}

func (w *cleanupWorker) Work(ctx context.Context, _ *river.Job[cleanupArgs]) error {
	return cleanup(ctx)
}

func addWorkers(s *Service) func(*river.Workers) {
	return func(w *river.Workers) {
		river.AddWorker(w, &importWorker{s: s})
		river.AddWorker(w, &exportWorker{s: s})
		river.AddWorker(w, &cleanupWorker{})
	}
}

// periodic runs cleanup every hour and at start, so files left while the app was
// down go at the next start.
var periodic = []*river.PeriodicJob{river.NewPeriodicJob(river.PeriodicInterval(time.Hour),
	func() (river.JobArgs, *river.InsertOpts) { return cleanupArgs{}, nil }, &river.PeriodicJobOpts{RunOnStart: true})}

// errRollback undoes an import with row errors; it never leaves runImport.
var errRollback = errors.New("dataio: rows rejected")

// runImport reads the requester's file and gives its rows to the module, all or nothing.
// The job completes in the import's transaction, so an import never runs twice.
func (s *Service) runImport(ctx context.Context, j *river.Job[importArgs]) error {
	a := j.Args
	imp, ok := s.imports[a.Target]
	if !ok {
		return platform.ErrNotFound
	}
	rows, err := s.readFile(ctx, a.FileID)
	if err != nil {
		return err
	}
	header, err := imp.Header(ctx, a.Params)
	if err != nil {
		return err
	}
	if got := len(rows[0]); got != len(header) {
		return errColumns(len(header), got)
	}
	for i, want := range header {
		if got := strings.TrimSpace(rows[0][i]); !strings.EqualFold(got, want) {
			return errHeader(i+1, want, got)
		}
	}
	data := rows[1:]
	for i, r := range data {
		if blank(r) {
			data[i] = nil
			continue
		}
		data[i] = append(r, make([]string, max(0, len(header)-len(r)))...)[:len(header)]
	}
	var rowErrs []RowError
	err = platform.InTx(ctx, func(ctx context.Context) error {
		if rowErrs, err = imp.Run(ctx, a.Params, data); err != nil {
			return err
		}
		if len(rowErrs) > 0 {
			return errRollback
		}
		n := len(data)
		return platform.CompleteJob(ctx, j, jobOutput{Rows: &n})
	})
	if len(rowErrs) > 0 {
		return s.rowsInvalid(ctx, imp.Product, rowErrs)
	}
	return err
}

// readFile opens the requester's own upload and returns its first sheet, header
// row included, without trailing blank rows and cells.
func (s *Service) readFile(ctx context.Context, id string) ([][]string, error) {
	row, err := store.New(platform.DBFrom(ctx)).GetFile(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFileGone
	}
	if err != nil {
		return nil, err
	}
	if actor, _ := platform.ActorFrom(ctx); row.OwnerID != actor {
		return nil, platform.ErrForbidden
	}
	f, err := files(ctx).Open(id)
	if err != nil {
		return nil, ErrFileGone
	}
	defer func() { _ = f.Close() }()
	// A small zip may unpack into gigabytes; refuse that rather than fill memory.
	x, err := excelize.OpenReader(f, excelize.Options{UnzipSizeLimit: 64 << 20, UnzipXMLSizeLimit: 32 << 20})
	if err != nil {
		return nil, ErrNotExcel
	}
	defer func() { _ = x.Close() }()
	sheets := x.GetSheetList()
	if len(sheets) == 0 {
		return nil, ErrEmptyFile
	}
	// Raw values, so a number reads the same whatever its display format.
	rows, err := x.GetRows(sheets[0], excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, ErrNotExcel
	}
	for len(rows) > 0 && blank(rows[len(rows)-1]) {
		rows = rows[:len(rows)-1]
	}
	switch {
	case len(rows) == 0:
		return nil, ErrEmptyFile
	case len(rows)-1 > maxImportRows:
		return nil, ErrTooManyRows
	}
	for len(rows[0]) > 0 && rows[0][len(rows[0])-1] == "" {
		rows[0] = rows[0][:len(rows[0])-1]
	}
	return rows, nil
}

func blank(r []string) bool {
	return !slices.ContainsFunc(r, func(c string) bool { return strings.TrimSpace(c) != "" })
}

// rowsInvalid translates row errors into the requester's language, by Excel row.
func (s *Service) rowsInvalid(ctx context.Context, product string, rowErrs []RowError) error {
	me, err := s.d.IAM.Me(ctx)
	if err != nil {
		return err
	}
	slices.SortStableFunc(rowErrs, func(a, b RowError) int { return a.Row - b.Row })
	out := make([]RowMessage, 0, min(len(rowErrs), maxRowErrors))
	for _, e := range rowErrs[:min(len(rowErrs), maxRowErrors)] {
		out = append(out, RowMessage{Row: excelRow(e.Row), Message: platform.Translate(me.Locale, product+".import."+e.Code, e.Params)})
	}
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "import_rows_invalid",
		Params: map[string]any{"rows": out, "count": len(rowErrs)}}
}

// runExport writes the module's export to a file only the requester may download, for a day.
func (s *Service) runExport(ctx context.Context, j *river.Job[exportArgs]) error {
	a := j.Args
	exp, ok := s.exports[a.Target]
	if !ok {
		return platform.ErrNotFound
	}
	file, err := export(ctx, exp, a.Params)
	if err != nil {
		return err
	}
	id, err := files(ctx).Save(bytes.NewReader(file.Body))
	if err != nil {
		return err
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		export := store.CreateFileParams{ExportKind: pgtype.Text{String: a.Target, Valid: true}, ExportParams: a.Params}
		if err := s.keep(ctx, id, file.Name, export); err != nil {
			return err
		}
		if err := s.d.Audit.Record(ctx, "dataio.exported", map[string]any{"kind": a.Target, "params": a.Params, "file_id": id}); err != nil {
			return err
		}
		return platform.CompleteJob(ctx, j, jobOutput{FileID: &id})
	})
}

// export runs a module's export into a file.
func export(ctx context.Context, exp Export, params json.RawMessage) (File, error) {
	if exp.Render != nil {
		return exp.Render(ctx, params)
	}
	sheet, err := exp.Run(ctx, params)
	if err != nil {
		return File{}, err
	}
	b, err := writeSheet(sheet)
	if err != nil {
		return File{}, err
	}
	return File{Name: sheet.Name + ".xlsx", Body: b.Bytes()}, nil
}

// writeSheet writes the header in bold, frozen, and the rows below it.
func writeSheet(sh Sheet) (*bytes.Buffer, error) {
	x := excelize.NewFile()
	defer func() { _ = x.Close() }()
	const name = "Sheet1"
	bold, err := x.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	for c, v := range sh.Header {
		cell, _ := excelize.CoordinatesToCellName(c+1, 1)
		if err := x.SetCellStr(name, cell, v); err != nil {
			return nil, err
		}
		if err := x.SetCellStyle(name, cell, cell, bold); err != nil {
			return nil, err
		}
	}
	if err := x.SetPanes(name, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return nil, err
	}
	for r, row := range sh.Rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			switch v := v.(type) {
			case string:
				err = x.SetCellStr(name, cell, v)
			case int64:
				err = x.SetCellInt(name, cell, v)
			case decimal.Decimal:
				// Written untyped, Excel reads it as a number.
				err = x.SetCellDefault(name, cell, v.String())
			case nil:
			default:
				panic("dataio: unsupported cell type")
			}
			if err != nil {
				return nil, err
			}
		}
	}
	return x.WriteToBuffer()
}

// cleanup removes expired files, then files on disk older than a day that no row
// names (left by a failure between writing a file and its row).
func cleanup(ctx context.Context) error {
	q := store.New(platform.DBFrom(ctx))
	disk := files(ctx)
	expired, err := q.DeleteExpiredFiles(ctx)
	if err != nil {
		return err
	}
	for _, id := range expired {
		if err := disk.Remove(id); err != nil {
			return err
		}
	}
	return disk.Sweep(ctx, time.Now().Add(-24*time.Hour), q.KnownFiles)
}
