// Package dataio reads and writes Excel for the modules that register imports and
// exports, runs them as jobs on behalf of the requester, and keeps their files.
package dataio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/taoworklabs/mmerp/internal/core/dataio/internal/store"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// fileTTL is how long uploaded and exported files are kept.
const fileTTL = 24 * time.Hour

type Service struct {
	d       Deps
	imports map[string]Import
	exports map[string]Export
}

func NewService(d Deps) *Service {
	return &Service{d: d, imports: map[string]Import{}, exports: map[string]Export{}}
}

// RegisterImport adds an import while wiring modules, before serving.
func (s *Service) RegisterImport(i Import) {
	if i.Product == "" {
		panic("dataio: import " + i.Kind + " has no product to gate it")
	}
	if _, dup := s.imports[i.Kind]; dup {
		panic("dataio: import " + i.Kind + " registered twice")
	}
	s.imports[i.Kind] = i
}

// RegisterExport adds an export while wiring modules, before serving.
func (s *Service) RegisterExport(e Export) {
	if e.Product == "" {
		panic("dataio: export " + e.Kind + " has no product to gate it")
	}
	if _, dup := s.exports[e.Kind]; dup {
		panic("dataio: export " + e.Kind + " registered twice")
	}
	if e.Check == nil {
		panic("dataio: export " + e.Kind + " has no Check")
	}
	if (e.Run == nil) == (e.Render == nil) {
		panic("dataio: export " + e.Kind + " needs exactly one of Run and Render")
	}
	s.exports[e.Kind] = e
}

// StartImport keeps the uploaded file and queues its import. Rights are checked
// when the import runs, as the actor.
func (s *Service) StartImport(ctx context.Context, kind, name string, r io.Reader, params json.RawMessage) (int64, error) {
	imp, ok := s.imports[kind]
	if !ok {
		return 0, platform.ErrNotFound
	}
	if params, ok = jsonObject(params); !ok {
		return 0, ErrInvalidParams
	}
	// Fail before keeping a file that could never be imported.
	if err := platform.ProductGate(ctx, imp.Product, platform.ClassWrite); err != nil {
		return 0, err
	}
	// The file first: a row must never name a missing file.
	id, err := files(ctx).Save(r)
	if err != nil {
		return 0, err
	}
	var job int64
	err = platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.keep(ctx, id, name, store.CreateFileParams{}); err != nil {
			return err
		}
		job, err = platform.Enqueue(ctx, importArgs{Target: kind, Product: imp.Product, FileID: id, Params: params})
		return err
	})
	return job, err
}

// StartExport queues an export; rights are checked when it runs, as the actor.
func (s *Service) StartExport(ctx context.Context, kind string, params json.RawMessage) (int64, error) {
	exp, ok := s.exports[kind]
	if !ok {
		return 0, platform.ErrNotFound
	}
	if params, ok = jsonObject(params); !ok {
		return 0, ErrInvalidParams
	}
	return platform.Enqueue(ctx, exportArgs{Target: kind, Product: exp.Product, Params: params})
}

// Template returns an empty import file: the header row only.
func (s *Service) Template(ctx context.Context, kind string, params json.RawMessage) ([]byte, error) {
	imp, ok := s.imports[kind]
	if !ok {
		return nil, platform.ErrNotFound
	}
	if params, ok = jsonObject(params); !ok {
		return nil, ErrInvalidParams
	}
	header, err := imp.Header(ctx, params)
	if err != nil {
		return nil, err
	}
	b, err := writeSheet(Sheet{Header: header})
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// File opens a file the actor owns, before it expires, and for an export while its
// module still lets the actor read it; any other does not exist.
func (s *Service) File(ctx context.Context, id string) (name string, f *os.File, err error) {
	row, err := store.New(platform.DBFrom(ctx)).GetFile(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, platform.ErrNotFound
	}
	if err != nil {
		return "", nil, err
	}
	if actor, _ := platform.ActorFrom(ctx); row.OwnerID != actor || !row.ExpiresAt.Time.After(time.Now()) {
		return "", nil, platform.ErrNotFound
	}
	if row.ExportKind.Valid {
		exp, ok := s.exports[row.ExportKind.String]
		if !ok {
			return "", nil, platform.ErrNotFound
		}
		if err := exp.Check(ctx, row.ExportParams); err != nil {
			var e *platform.Error
			if errors.As(err, &e) {
				return "", nil, platform.ErrNotFound
			}
			return "", nil, err
		}
	}
	f, err = files(ctx).Open(id)
	if os.IsNotExist(err) {
		return "", nil, platform.ErrNotFound
	}
	return row.Name, f, err
}

// JobFilter narrows a job list: System lists system jobs instead of the actor's own
// (core.job.monitor only); Failed keeps the jobs that failed or wait to retry.
type JobFilter struct{ System, Failed bool }

// Jobs lists the actor's own background jobs, or the system jobs, newest first.
func (s *Service) Jobs(ctx context.Context, f JobFilter) ([]Job, error) {
	p := river.NewJobListParams().OrderBy(river.JobListOrderByID, river.SortOrderDesc).First(100)
	if f.System {
		if err := s.d.IAM.RequireCore(ctx, iam.PermMonitorJobs); err != nil {
			return nil, err
		}
		p = p.Where("NOT (metadata ? 'requested_by')")
	} else {
		actor, _ := platform.ActorFrom(ctx)
		meta, err := json.Marshal(map[string]int64{"requested_by": actor})
		if err != nil {
			return nil, err
		}
		p = p.Metadata(string(meta))
	}
	if f.Failed {
		p = p.States(rivertype.JobStateRetryable, rivertype.JobStateCancelled, rivertype.JobStateDiscarded)
	} else {
		p = p.States(rivertype.JobStateAvailable, rivertype.JobStateScheduled, rivertype.JobStatePending, rivertype.JobStateRetryable,
			rivertype.JobStateRunning, rivertype.JobStateCompleted, rivertype.JobStateCancelled, rivertype.JobStateDiscarded)
	}
	res, err := platform.JobsFrom(ctx).JobList(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]Job, len(res.Jobs))
	for i, j := range res.Jobs {
		out[i] = job(j)
	}
	return out, nil
}

// Job returns one of the actor's own jobs; anyone else's does not exist.
func (s *Service) Job(ctx context.Context, id int64) (Job, error) {
	j, err := platform.JobsFrom(ctx).JobGet(ctx, id)
	if errors.Is(err, river.ErrNotFound) {
		return Job{}, platform.ErrNotFound
	}
	if err != nil {
		return Job{}, err
	}
	var meta platform.JobMeta
	if actor, _ := platform.ActorFrom(ctx); json.Unmarshal(j.Metadata, &meta) != nil || meta.RequestedBy != actor {
		return Job{}, platform.ErrNotFound
	}
	return job(j), nil
}

// jobOutput is the output of every dataio job; other jobs share its error field.
type jobOutput struct {
	platform.JobOutput
	Rows   *int    `json:"rows,omitempty"`
	FileID *string `json:"file_id,omitempty"`
}

func job(j *rivertype.JobRow) Job {
	out := Job{ID: j.ID, Kind: j.Kind, Attempts: j.Attempt, CreatedAt: j.CreatedAt.Format(time.RFC3339), RowErrors: []RowMessage{}}
	var args struct {
		Target string `json:"target"`
	}
	if json.Unmarshal(j.EncodedArgs, &args) == nil && args.Target != "" {
		out.Target = &args.Target
	}
	if j.FinalizedAt != nil {
		at := j.FinalizedAt.Format(time.RFC3339)
		out.FinishedAt = &at
	}
	var o jobOutput
	_ = json.Unmarshal(j.Output(), &o)
	switch j.State {
	case rivertype.JobStateRunning:
		out.State = "running"
	case rivertype.JobStateCompleted:
		out.State, out.Rows, out.FileID = "completed", o.Rows, o.FileID
	case rivertype.JobStateRetryable:
		code := lastError(j)
		out.State, out.ErrorCode = "retrying", &code
	case rivertype.JobStateCancelled, rivertype.JobStateDiscarded:
		code := lastError(j)
		if o.Error != nil {
			code, out.ErrorParams = o.Error.Code, o.Error.Params
		}
		out.State, out.ErrorCode = "failed", &code
		if rows, ok := out.ErrorParams["rows"]; ok {
			b, _ := json.Marshal(rows)
			_ = json.Unmarshal(b, &out.RowErrors)
			delete(out.ErrorParams, "rows")
		}
	default:
		out.State = "queued"
	}
	return out
}

// errorCode is what a job's error message is when it is a stable error code.
var errorCode = regexp.MustCompile(`^[a-z][a-z_]*$`)

// lastError is the code of a job's last error; any other message (a path, a driver's
// text) may name what the reader must not see, so it shows as internal_error.
func lastError(j *rivertype.JobRow) string {
	if n := len(j.Errors); n > 0 && errorCode.MatchString(j.Errors[n-1].Error) {
		return j.Errors[n-1].Error
	}
	return "internal_error"
}

// keep records a saved file as the actor's, until it expires; p names the export, if any.
func (s *Service) keep(ctx context.Context, id, name string, p store.CreateFileParams) error {
	actor, _ := platform.ActorFrom(ctx)
	p.ID, p.Name, p.OwnerID = id, name, actor
	p.ExpiresAt = pgtype.Timestamptz{Time: time.Now().Add(fileTTL), Valid: true}
	return store.New(platform.DBFrom(ctx)).CreateFile(ctx, p)
}

// jsonObject accepts params as a JSON object; empty means {}.
func jsonObject(params json.RawMessage) (json.RawMessage, bool) {
	if len(bytes.TrimSpace(params)) == 0 {
		return json.RawMessage(`{}`), true
	}
	var m map[string]any
	return params, json.Unmarshal(params, &m) == nil && m != nil
}

// excelRow numbers a data row the way Excel shows it, below the header.
func excelRow(i int) int { return i + 2 }

// files is the store of dataio, apart from the files of other modules.
func files(ctx context.Context) platform.Files { return platform.FilesFrom(ctx).Sub("dataio") }
