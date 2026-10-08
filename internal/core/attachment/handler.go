package attachment

import (
	"context"
	"io"
	"mime"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type refInput struct {
	Type string `path:"type" doc:"Record type, e.g. hrm.leave_request"`
	ID   int64  `path:"id"`
}

type listOutput struct{ Body AttachmentList }

type addForm struct {
	File huma.FormFile `form:"file" required:"true"`
}

type addInput struct {
	Type    string `path:"type"`
	ID      int64  `path:"id"`
	RawBody huma.MultipartFormFiles[addForm]
}

type idInput struct {
	ID int64 `path:"id"`
}

type fileOutput struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

func registerHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-attachments", Method: http.MethodGet, Path: "/records/{type}/{id}/attachments",
		Description: "Not found unless the caller may view the record; hidden when its files need a right the caller lacks."},
		func(ctx context.Context, in *refInput) (*listOutput, error) {
			l, err := s.List(ctx, record.Ref{Type: in.Type, ID: in.ID})
			return &listOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "add-attachment", Method: http.MethodPost, Path: "/records/{type}/{id}/attachments",
		Description: "One file: PDF, JPEG, PNG, DOCX or XLSX, told by its content.",
		// Room for the multipart framing around the largest file.
		Middlewares: platform.UploadLimit(maxBytes+64<<10, ErrFileTooLarge), DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *addInput) (*struct{}, error) {
			f := in.RawBody.Data().File
			if f.Size > maxBytes {
				return nil, ErrFileTooLarge
			}
			return nil, s.Add(ctx, record.Ref{Type: in.Type, ID: in.ID}, f.Filename, f)
		})
	huma.Register(api, huma.Operation{OperationID: "remove-attachment", Method: http.MethodDelete, Path: "/attachments/{id}",
		Description: "By its uploader, or whoever may attach to the record; the file goes at the next cleanup.", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *idInput) (*struct{}, error) {
			return nil, s.Remove(ctx, in.ID)
		})
	huma.Register(api, huma.Operation{OperationID: "download-attachment", Method: http.MethodGet, Path: "/attachments/{id}",
		Description: "Asks again whether the caller may see the record's files."},
		func(ctx context.Context, in *idInput) (*fileOutput, error) {
			meta, f, err := s.Open(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			defer func() { _ = f.Close() }()
			b, err := io.ReadAll(f)
			if err != nil {
				return nil, err
			}
			return &fileOutput{
				ContentType:        meta.ContentType,
				ContentDisposition: mime.FormatMediaType("attachment", map[string]string{"filename": meta.Name}),
				CacheControl:       "no-store",
				Body:               b,
			}, nil
		})
}
