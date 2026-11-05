package accounting

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type pendingInput struct {
	LegalEntityID int64 `query:"legal_entity_id" required:"true"`
}

type pendingOutput struct{ Body PendingList }

type postPendingInput struct {
	Body struct {
		LegalEntityID int64 `json:"legal_entity_id"`
	}
}

type postPendingOutput struct {
	Body struct {
		Queued int `json:"queued"`
	}
}

func registerPendingHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-accounting-pending", Method: http.MethodGet, Path: "/accounting/pending"},
		func(ctx context.Context, in *pendingInput) (*pendingOutput, error) {
			items, err := s.Pending(ctx, in.LegalEntityID)
			return &pendingOutput{Body: PendingList{Items: items}}, err
		})
	huma.Register(api, huma.Operation{OperationID: "post-accounting-pending", Method: http.MethodPost, Path: "/accounting/pending/post"},
		func(ctx context.Context, in *postPendingInput) (*postPendingOutput, error) {
			n, err := s.PostPending(ctx, in.Body.LegalEntityID)
			out := &postPendingOutput{}
			out.Body.Queued = n
			return out, err
		})
}
