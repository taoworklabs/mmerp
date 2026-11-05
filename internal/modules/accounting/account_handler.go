package accounting

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type accountsOutput struct{ Body AccountList }

type accountInput struct{ Body AccountFields }

type updateAccountInput struct {
	ID   int64 `path:"id"`
	Body AccountFields
}

type accountIDInput struct {
	ID int64 `path:"id"`
}

func registerAccountHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-accounts", Method: http.MethodGet, Path: "/accounting/accounts"},
		func(ctx context.Context, _ *struct{}) (*accountsOutput, error) {
			items, err := s.Accounts(ctx)
			return &accountsOutput{Body: AccountList{Items: items}}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-account", Method: http.MethodPost, Path: "/accounting/accounts", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *accountInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateAccount(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "update-account", Method: http.MethodPut, Path: "/accounting/accounts/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateAccountInput) (*struct{}, error) {
			return nil, s.UpdateAccount(ctx, in.ID, in.Body)
		})
	huma.Register(api, huma.Operation{OperationID: "delete-account", Method: http.MethodDelete, Path: "/accounting/accounts/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *accountIDInput) (*struct{}, error) {
			return nil, s.DeleteAccount(ctx, in.ID)
		})
}
