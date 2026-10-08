package notification

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type listInput struct {
	Before int64 `query:"before" doc:"Only notifications older than this id"`
}

type listOutput struct{ Body Page }

type unreadOutput struct {
	Body struct {
		Count int64 `json:"count"`
	}
}

type mailServerOutput struct{ Body MailServer }

type mailServerInput struct{ Body MailServerInput }

type idInput struct {
	ID int64 `path:"id"`
}

func registerHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-notifications", Method: http.MethodGet, Path: "/notifications",
		Description: "The caller's notifications, newest first, a page at a time."},
		func(ctx context.Context, in *listInput) (*listOutput, error) {
			var before *int64
			if in.Before > 0 {
				before = &in.Before
			}
			p, err := s.List(ctx, before)
			return &listOutput{Body: p}, err
		})
	huma.Register(api, huma.Operation{OperationID: "count-unread-notifications", Method: http.MethodGet, Path: "/notifications/unread-count"},
		func(ctx context.Context, _ *struct{}) (*unreadOutput, error) {
			out := &unreadOutput{}
			var err error
			out.Body.Count, err = s.Unread(ctx)
			return out, err
		})
	huma.Register(api, huma.Operation{OperationID: "read-notification", Method: http.MethodPost, Path: "/notifications/{id}/read", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *idInput) (*struct{}, error) { return nil, s.MarkRead(ctx, in.ID) })
	huma.Register(api, huma.Operation{OperationID: "read-all-notifications", Method: http.MethodPost, Path: "/notifications/read-all", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, _ *struct{}) (*struct{}, error) { return nil, s.MarkAllRead(ctx) })
	huma.Register(api, huma.Operation{OperationID: "get-mail-server", Method: http.MethodGet, Path: "/mail-server",
		Description: "The tenant's mail server, without its password; not found when there is none."},
		func(ctx context.Context, _ *struct{}) (*mailServerOutput, error) {
			m, err := s.MailServer(ctx)
			return &mailServerOutput{Body: m}, err
		})
	huma.Register(api, huma.Operation{OperationID: "save-mail-server", Method: http.MethodPut, Path: "/mail-server", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *mailServerInput) (*struct{}, error) {
			return nil, s.SaveMailServer(ctx, in.Body)
		})
	huma.Register(api, huma.Operation{OperationID: "delete-mail-server", Method: http.MethodDelete, Path: "/mail-server", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, _ *struct{}) (*struct{}, error) { return nil, s.DeleteMailServer(ctx) })
	huma.Register(api, huma.Operation{OperationID: "send-test-mail", Method: http.MethodPost, Path: "/mail-server/test", DefaultStatus: http.StatusAccepted,
		Description: "Queues a test message to the caller; its outcome shows among the system jobs."},
		func(ctx context.Context, _ *struct{}) (*struct{}, error) { return nil, s.SendTestMail(ctx) })
}
