package purchase

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type suppliersOutput struct{ Body SupplierList }

type supplierOutput struct{ Body Supplier }

type supplierInput struct{ Body NewSupplier }

type updateSupplierInput struct {
	ID   int64 `path:"id"`
	Body SupplierUpdate
}

func registerSupplierHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-suppliers", Method: http.MethodGet, Path: "/purchase/suppliers"},
		func(ctx context.Context, in *SupplierFilter) (*suppliersOutput, error) {
			l, err := s.Suppliers(ctx, *in)
			return &suppliersOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-supplier-actions", Method: http.MethodGet, Path: "/purchase/suppliers/actions"},
		func(ctx context.Context, _ *struct{}) (*actionsOutput, error) {
			a, err := s.SupplierActions(ctx)
			return &actionsOutput{Body: a}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-supplier", Method: http.MethodPost, Path: "/purchase/suppliers", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *supplierInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateSupplier(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-supplier", Method: http.MethodGet, Path: "/purchase/suppliers/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*supplierOutput, error) {
			c, err := s.Supplier(ctx, in.ID)
			return &supplierOutput{Body: c}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-supplier", Method: http.MethodPut, Path: "/purchase/suppliers/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateSupplierInput) (*struct{}, error) {
			return nil, s.UpdateSupplier(ctx, in.ID, in.Body)
		})
}
