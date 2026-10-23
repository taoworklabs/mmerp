package sales

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type customersOutput struct{ Body CustomerList }

type customerOutput struct{ Body Customer }

type customerInput struct{ Body NewCustomer }

type updateCustomerInput struct {
	ID   int64 `path:"id"`
	Body CustomerUpdate
}

func registerCustomerHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-customers", Method: http.MethodGet, Path: "/sales/customers"},
		func(ctx context.Context, in *CustomerFilter) (*customersOutput, error) {
			l, err := s.Customers(ctx, *in)
			return &customersOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-customer-actions", Method: http.MethodGet, Path: "/sales/customers/actions"},
		func(ctx context.Context, _ *struct{}) (*actionsOutput, error) {
			a, err := s.CustomerActions(ctx)
			return &actionsOutput{Body: a}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-customer", Method: http.MethodPost, Path: "/sales/customers", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *customerInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateCustomer(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-customer", Method: http.MethodGet, Path: "/sales/customers/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*customerOutput, error) {
			c, err := s.Customer(ctx, in.ID)
			return &customerOutput{Body: c}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-customer", Method: http.MethodPut, Path: "/sales/customers/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateCustomerInput) (*struct{}, error) {
			return nil, s.UpdateCustomer(ctx, in.ID, in.Body)
		})
}
