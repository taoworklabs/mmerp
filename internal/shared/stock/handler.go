package stock

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type warehousesOutput struct{ Body WarehouseList }

type warehouseInput struct{ Body NewWarehouse }

type updateWarehouseInput struct {
	ID   int64 `path:"id"`
	Body WarehouseFields
}

func registerHandlers(api huma.API, s *Service) {
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
