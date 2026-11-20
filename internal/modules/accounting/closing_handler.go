package accounting

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type closingsOutput struct{ Body ClosingList }

type closingOutput struct{ Body Closing }

type closingActionsOutput struct{ Body ClosingActions }

type closingAccountsOutput struct{ Body ClosingAccountsOut }

type newClosingInput struct{ Body NewClosing }

type updateClosingInput struct {
	ID   int64 `path:"id"`
	Body ClosingUpdate
}

type closingAccountsInput struct{ Body ClosingAccounts }

func registerClosingHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-closings", Method: http.MethodGet, Path: "/accounting/closings"},
		func(ctx context.Context, in *ClosingFilter) (*closingsOutput, error) {
			l, err := s.Closings(ctx, *in)
			return &closingsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-closing-actions", Method: http.MethodGet, Path: "/accounting/closings/actions"},
		func(ctx context.Context, _ *struct{}) (*closingActionsOutput, error) {
			a, err := s.ClosingActions(ctx)
			return &closingActionsOutput{Body: a}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-closing", Method: http.MethodPost, Path: "/accounting/closings", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *newClosingInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateClosing(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-closing", Method: http.MethodGet, Path: "/accounting/closings/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*closingOutput, error) {
			c, err := s.Closing(ctx, in.ID)
			return &closingOutput{Body: c}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-closing", Method: http.MethodPut, Path: "/accounting/closings/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateClosingInput) (*struct{}, error) {
			return nil, s.UpdateClosing(ctx, in.ID, in.Body)
		})
	huma.Register(api, huma.Operation{OperationID: "get-closing-accounts", Method: http.MethodGet, Path: "/accounting/closing-accounts"},
		func(ctx context.Context, _ *struct{}) (*closingAccountsOutput, error) {
			a, err := s.ClosingAccounts(ctx)
			return &closingAccountsOutput{Body: a}, err
		})
	huma.Register(api, huma.Operation{OperationID: "set-closing-accounts", Method: http.MethodPut, Path: "/accounting/closing-accounts", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *closingAccountsInput) (*struct{}, error) {
			return nil, s.SetClosingAccounts(ctx, in.Body)
		})
}
