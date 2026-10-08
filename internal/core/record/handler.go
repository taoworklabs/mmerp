package record

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type refInput struct {
	Type string `path:"type" doc:"Record type, e.g. hrm.leave_request"`
	ID   int64  `path:"id"`
}

type transitionInput struct {
	Type string `path:"type"`
	ID   int64  `path:"id"`
	Body struct {
		To      string `json:"to" enum:"posted,draft,cancelled" doc:"posted sends (may stop at pending_approval), draft withdraws, cancelled cancels"`
		Version int32  `json:"version"`
	}
}

type historyOutput struct{ Body []HistoryEntry }

type periodLocksOutput struct{ Body []PeriodLock }

type setPeriodLockInput struct {
	LegalEntityID int64 `path:"legal_entity"`
	Body          struct {
		LockedUntil *string `json:"locked_until" format:"date"`
	}
}

func registerHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "transition-document", Method: http.MethodPost, Path: "/documents/{type}/{id}/transitions", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *transitionInput) (*struct{}, error) {
			if _, ok := s.types[in.Type]; !ok {
				return nil, platform.ErrNotFound
			}
			return nil, s.Transition(ctx, Ref{in.Type, in.ID}, in.Body.Version, Status(in.Body.To))
		})
	huma.Register(api, huma.Operation{OperationID: "get-document-history", Method: http.MethodGet, Path: "/documents/{type}/{id}/history"},
		func(ctx context.Context, in *refInput) (*historyOutput, error) {
			if _, ok := s.types[in.Type]; !ok {
				return nil, platform.ErrNotFound
			}
			h, err := s.History(ctx, Ref{in.Type, in.ID})
			return &historyOutput{Body: h}, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-period-locks", Method: http.MethodGet, Path: "/period-locks"},
		func(ctx context.Context, _ *struct{}) (*periodLocksOutput, error) {
			l, err := s.PeriodLocks(ctx)
			return &periodLocksOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "set-period-lock", Method: http.MethodPut, Path: "/period-locks/{legal_entity}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *setPeriodLockInput) (*struct{}, error) {
			return nil, s.SetPeriodLock(ctx, in.LegalEntityID, in.Body.LockedUntil)
		})
}
