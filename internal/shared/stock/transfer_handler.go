package stock

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type transfersOutput struct{ Body TransferList }

type transferOutput struct{ Body Transfer }

type newTransferInput struct{ Body NewTransfer }

type updateTransferInput struct {
	ID   int64 `path:"id"`
	Body TransferUpdate
}

func registerTransferHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-stock-transfers", Method: http.MethodGet, Path: "/stock/transfers"},
		func(ctx context.Context, in *TransferFilter) (*transfersOutput, error) {
			l, err := s.Transfers(ctx, *in)
			return &transfersOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-stock-transfer-actions", Method: http.MethodGet, Path: "/stock/transfers/actions"},
		func(ctx context.Context, _ *struct{}) (*actionsOutput, error) {
			a, err := s.TransferActions(ctx)
			return &actionsOutput{Body: a}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-stock-transfer", Method: http.MethodPost, Path: "/stock/transfers", DefaultStatus: http.StatusCreated,
		Description: "Sending the same request_id again returns the document created the first time."},
		func(ctx context.Context, in *newTransferInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateTransfer(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-stock-transfer", Method: http.MethodGet, Path: "/stock/transfers/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*transferOutput, error) {
			t, err := s.Transfer(ctx, in.ID)
			return &transferOutput{Body: t}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-stock-transfer", Method: http.MethodPut, Path: "/stock/transfers/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateTransferInput) (*struct{}, error) {
			return nil, s.UpdateTransfer(ctx, in.ID, in.Body)
		})
}
