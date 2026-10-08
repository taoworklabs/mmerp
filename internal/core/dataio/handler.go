package dataio

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"path"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

const xlsx = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

type importForm struct {
	File   huma.FormFile `form:"file" required:"true"`
	Params string        `form:"params" required:"false" doc:"JSON object, e.g. {\"timesheet_id\": 1}"`
}

type importInput struct {
	Kind    string `path:"kind" doc:"e.g. hrm.timesheet"`
	RawBody huma.MultipartFormFiles[importForm]
}

type kindInput struct {
	Kind   string `path:"kind"`
	Params string `query:"params" doc:"JSON object"`
}

type exportInput struct {
	Kind string `path:"kind"`
	Body struct {
		Params map[string]any `json:"params"`
	}
}

type jobIDOutput struct {
	Body struct {
		JobID int64 `json:"job_id"`
	}
}

type fileOutput struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

type jobsInput struct {
	System bool `query:"system" doc:"System jobs instead of the caller's own; needs core.job.monitor"`
	Failed bool `query:"failed" doc:"Only jobs that failed or wait to retry"`
}

type jobsOutput struct {
	Body []Job `nullable:"false"`
}

type jobResponse struct{ Body Job }

type jobIDInput struct {
	ID int64 `path:"id"`
}

type fileIDInput struct {
	ID string `path:"id"`
}

func jobID(id int64, err error) (*jobIDOutput, error) {
	if err != nil {
		return nil, err
	}
	out := &jobIDOutput{}
	out.Body.JobID = id
	return out, nil
}

// contentTypes are the types of files other than Excel, by name extension.
var contentTypes = map[string]string{".pdf": "application/pdf"}

func attachment(name string, b []byte) *fileOutput {
	t, ok := contentTypes[path.Ext(name)]
	if !ok {
		t = xlsx
	}
	return &fileOutput{
		ContentType:        t,
		ContentDisposition: mime.FormatMediaType("attachment", map[string]string{"filename": name}),
		CacheControl:       "no-store",
		Body:               b,
	}
}

func registerHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "start-import", Method: http.MethodPost, Path: "/imports/{kind}",
		Description: "Keeps the file and queues the import; it runs as the caller, who follows it at /jobs/{id}.",
		Middlewares: platform.UploadLimit(maxImportBytes+64<<10, ErrFileTooLarge), DefaultStatus: http.StatusAccepted},
		func(ctx context.Context, in *importInput) (*jobIDOutput, error) {
			f := in.RawBody.Data().File
			if f.Size > maxImportBytes {
				return nil, ErrFileTooLarge
			}
			return jobID(s.StartImport(ctx, in.Kind, f.Filename, io.LimitReader(f, maxImportBytes), json.RawMessage(in.RawBody.Data().Params)))
		})
	huma.Register(api, huma.Operation{OperationID: "get-import-template", Method: http.MethodGet, Path: "/imports/{kind}/template",
		Description: "An empty import file: the header row only."},
		func(ctx context.Context, in *kindInput) (*fileOutput, error) {
			b, err := s.Template(ctx, in.Kind, json.RawMessage(in.Params))
			if err != nil {
				return nil, err
			}
			return attachment(in.Kind+".xlsx", b), nil
		})
	huma.Register(api, huma.Operation{OperationID: "start-export", Method: http.MethodPost, Path: "/exports/{kind}",
		Description: "Queues the export; it runs as the caller, whose job gives the file to download.", DefaultStatus: http.StatusAccepted},
		func(ctx context.Context, in *exportInput) (*jobIDOutput, error) {
			params, err := json.Marshal(in.Body.Params)
			if err != nil {
				return nil, err
			}
			return jobID(s.StartExport(ctx, in.Kind, params))
		})
	huma.Register(api, huma.Operation{OperationID: "list-jobs", Method: http.MethodGet, Path: "/jobs",
		Description: "The caller's own background jobs, or the system jobs with core.job.monitor, newest first; finished jobs are kept for 7 days."},
		func(ctx context.Context, in *jobsInput) (*jobsOutput, error) {
			jobs, err := s.Jobs(ctx, JobFilter{System: in.System, Failed: in.Failed})
			return &jobsOutput{Body: jobs}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-job", Method: http.MethodGet, Path: "/jobs/{id}"},
		func(ctx context.Context, in *jobIDInput) (*jobResponse, error) {
			j, err := s.Job(ctx, in.ID)
			return &jobResponse{Body: j}, err
		})
	huma.Register(api, huma.Operation{OperationID: "download-file", Method: http.MethodGet, Path: "/files/{id}",
		Description: "Only the file's owner, before it expires; an export only while they may still read it."},
		func(ctx context.Context, in *fileIDInput) (*fileOutput, error) {
			name, f, err := s.File(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			defer func() { _ = f.Close() }()
			b, err := io.ReadAll(f)
			if err != nil {
				return nil, err
			}
			return attachment(name, b), nil
		})
}
