package sales

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type docsOutput struct{ Body DocList }

type docOutput struct{ Body Doc }

type newDocInput struct{ Body NewDoc }

// orderFromQuoteOutput: id is null while an approval request holds the quotation.
type orderFromQuoteOutput struct {
	Body struct {
		ID   *int64 `json:"id"`
		Held bool   `json:"held"`
	}
}

type updateDocInput struct {
	ID   int64 `path:"id"`
	Body DocUpdate
}

func registerDocHandlers(api huma.API, s *Service) {
	docs := []struct {
		path, one, many string
		actions         func(context.Context) (Actions, error)
		list            func(context.Context, DocFilter) (DocList, error)
		get             func(context.Context, int64) (Doc, error)
		create          func(context.Context, NewDoc) (int64, error)
		update          func(context.Context, int64, DocUpdate) error
	}{
		{"/sales/quotes", "quote", "quotes", s.QuoteActions, s.Quotes, s.Quote, s.CreateQuote, s.UpdateQuote},
		{"/sales/orders", "order", "orders", s.OrderActions, s.Orders, s.Order, s.CreateOrder, s.UpdateOrder},
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
			func(ctx context.Context, in *newDocInput) (*platform.CreatedOutput, error) {
				return platform.Created(d.create(ctx, in.Body))
			})
		huma.Register(api, huma.Operation{OperationID: "get-" + d.one, Method: http.MethodGet, Path: d.path + "/{id}"},
			func(ctx context.Context, in *platform.IDInput) (*docOutput, error) {
				doc, err := d.get(ctx, in.ID)
				return &docOutput{Body: doc}, err
			})
		huma.Register(api, huma.Operation{OperationID: "update-" + d.one, Method: http.MethodPut, Path: d.path + "/{id}", DefaultStatus: http.StatusNoContent},
			func(ctx context.Context, in *updateDocInput) (*struct{}, error) {
				return nil, d.update(ctx, in.ID, in.Body)
			})
	}
	huma.Register(api, huma.Operation{OperationID: "create-order-from-quote", Method: http.MethodPost, Path: "/sales/quotes/{id}/order",
		Description: "While an order from the quotation is not cancelled, returns that order instead of making another. " +
			"When an approval rule holds the quotation, answers held with no id."},
		func(ctx context.Context, in *platform.IDInput) (*orderFromQuoteOutput, error) {
			id, held, err := s.CreateOrderFromQuote(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out := &orderFromQuoteOutput{}
			if out.Body.Held = held; !held {
				out.Body.ID = &id
			}
			return out, nil
		})
}
