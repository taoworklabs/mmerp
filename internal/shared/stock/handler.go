package stock

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type warehousesOutput struct{ Body WarehouseList }

type onHandOutput struct{ Body OnHandList }

type cardOutput struct{ Body StockCard }

type warehouseInput struct{ Body NewWarehouse }

type updateWarehouseInput struct {
	ID   int64 `path:"id"`
	Body WarehouseUpdate
}

func registerHandlers(api huma.API, s *Service) {
	registerAdjustmentHandlers(api, s)
	huma.Register(api, huma.Operation{OperationID: "list-stock-on-hand", Method: http.MethodGet, Path: "/stock/on-hand"},
		func(ctx context.Context, in *OnHandFilter) (*onHandOutput, error) {
			l, err := s.OnHand(ctx, *in)
			return &onHandOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-stock-card", Method: http.MethodGet, Path: "/stock/card"},
		func(ctx context.Context, in *StockCardFilter) (*cardOutput, error) {
			c, err := s.StockCard(ctx, in.WarehouseID, in.ItemID, in.From, in.To)
			return &cardOutput{Body: c}, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-warehouses", Method: http.MethodGet, Path: "/stock/warehouses"},
		func(ctx context.Context, in *WarehouseFilter) (*warehousesOutput, error) {
			l, err := s.Warehouses(ctx, *in)
			return &warehousesOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-warehouse", Method: http.MethodPost, Path: "/stock/warehouses", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *warehouseInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateWarehouse(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "update-warehouse", Method: http.MethodPut, Path: "/stock/warehouses/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateWarehouseInput) (*struct{}, error) {
			return nil, s.UpdateWarehouse(ctx, in.ID, in.Body)
		})
}
