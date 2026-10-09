package sales

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type idInput struct {
	ID int64 `path:"id"`
}

type versionInput struct {
	ID      int64 `path:"id"`
	Version int32 `query:"version" required:"true"`
}

type createdOutput struct {
	Body struct {
		ID int64 `json:"id"`
	}
}

func created(id int64, err error) (*createdOutput, error) {
	if err != nil {
		return nil, err
	}
	out := &createdOutput{}
	out.Body.ID = id
	return out, nil
}

type actionsOutput struct{ Body Actions }

type customersOutput struct{ Body CustomerList }

type customerOutput struct{ Body Customer }

type customerInput struct{ Body CustomerFields }

type updateCustomerInput struct {
	ID   int64 `path:"id"`
	Body CustomerFields
}

type itemsOutput struct{ Body ItemList }

type itemInput struct{ Body ItemFields }

type updateItemInput struct {
	ID   int64 `path:"id"`
	Body ItemFields
}

type docsOutput struct{ Body DocList }

type docOutput struct{ Body Doc }

type newDocInput struct{ Body NewDoc }

type updateDocInput struct {
	ID   int64 `path:"id"`
	Body DocUpdate
}

func registerHandlers(api huma.API, s *Service) {
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
		func(ctx context.Context, in *customerInput) (*createdOutput, error) {
			return created(s.CreateCustomer(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-customer", Method: http.MethodGet, Path: "/sales/customers/{id}"},
		func(ctx context.Context, in *idInput) (*customerOutput, error) {
			c, err := s.Customer(ctx, in.ID)
			return &customerOutput{Body: c}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-customer", Method: http.MethodPut, Path: "/sales/customers/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateCustomerInput) (*struct{}, error) {
			return nil, s.UpdateCustomer(ctx, in.ID, in.Body)
		})

	huma.Register(api, huma.Operation{OperationID: "list-items", Method: http.MethodGet, Path: "/sales/items"},
		func(ctx context.Context, in *ItemFilter) (*itemsOutput, error) {
			l, err := s.Items(ctx, *in)
			return &itemsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-item", Method: http.MethodPost, Path: "/sales/items", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *itemInput) (*createdOutput, error) {
			return created(s.SaveItem(ctx, 0, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "update-item", Method: http.MethodPut, Path: "/sales/items/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateItemInput) (*struct{}, error) {
			_, err := s.SaveItem(ctx, in.ID, in.Body)
			return nil, err
		})

	docs := []struct {
		path, one, many string
		actions         func(context.Context) (Actions, error)
		list            func(context.Context, DocFilter) (DocList, error)
		get             func(context.Context, int64) (Doc, error)
		create          func(context.Context, NewDoc) (int64, error)
		update          func(context.Context, int64, DocUpdate) error
		del             func(context.Context, int64, int32) error
	}{
		{"/sales/quotes", "quote", "quotes", s.QuoteActions, s.Quotes, s.Quote, s.CreateQuote, s.UpdateQuote, s.DeleteQuote},
		{"/sales/orders", "order", "orders", s.OrderActions, s.Orders, s.Order, s.CreateOrder, s.UpdateOrder, s.DeleteOrder},
	}
	for _, d := range docs {
		huma.Register(api, huma.Operation{OperationID: "list-" + d.many, Method: http.MethodGet, Path: d.path},
			func(ctx context.Context, in *DocFilter) (*docsOutput, error) {
				l, err := d.list(ctx, *in)
				return &docsOutput{Body: l}, err
			})
		huma.Register(api, huma.Operation{OperationID: "get-" + d.one + "-actions", Method: http.MethodGet, Path: d.path + "/actions"},
			func(ctx context.Context, _ *struct{}) (*actionsOutput, error) {
				a, err := d.actions(ctx)
				return &actionsOutput{Body: a}, err
			})
		huma.Register(api, huma.Operation{OperationID: "create-" + d.one, Method: http.MethodPost, Path: d.path, DefaultStatus: http.StatusCreated,
			Description: "Sending the same request_id again returns the document created the first time."},
			func(ctx context.Context, in *newDocInput) (*createdOutput, error) {
				return created(d.create(ctx, in.Body))
			})
		huma.Register(api, huma.Operation{OperationID: "get-" + d.one, Method: http.MethodGet, Path: d.path + "/{id}"},
			func(ctx context.Context, in *idInput) (*docOutput, error) {
				doc, err := d.get(ctx, in.ID)
				return &docOutput{Body: doc}, err
			})
		huma.Register(api, huma.Operation{OperationID: "update-" + d.one, Method: http.MethodPut, Path: d.path + "/{id}", DefaultStatus: http.StatusNoContent},
			func(ctx context.Context, in *updateDocInput) (*struct{}, error) {
				return nil, d.update(ctx, in.ID, in.Body)
			})
		huma.Register(api, huma.Operation{OperationID: "delete-" + d.one, Method: http.MethodDelete, Path: d.path + "/{id}", DefaultStatus: http.StatusNoContent},
			func(ctx context.Context, in *versionInput) (*struct{}, error) {
				return nil, d.del(ctx, in.ID, in.Version)
			})
	}
	huma.Register(api, huma.Operation{OperationID: "create-order-from-quote", Method: http.MethodPost, Path: "/sales/quotes/{id}/order",
		DefaultStatus: http.StatusCreated, Description: "While an order from the quotation is not cancelled, returns that order instead of making another."},
		func(ctx context.Context, in *idInput) (*createdOutput, error) {
			return created(s.CreateOrderFromQuote(ctx, in.ID))
		})
}
