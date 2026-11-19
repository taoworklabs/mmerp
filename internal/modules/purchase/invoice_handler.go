package purchase

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type invoicesOutput struct{ Body InvoiceList }

type invoiceOutput struct{ Body Invoice }

type newInvoiceInput struct{ Body NewInvoice }

type updateInvoiceInput struct {
	ID   int64 `path:"id"`
	Body InvoiceUpdate
}

type invoiceOptionsOutput struct{ Body InvoiceOptions }

func registerInvoiceHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "get-purchase-invoice-options", Method: http.MethodGet, Path: "/purchase/orders/{id}/invoice-options"},
		func(ctx context.Context, in *platform.IDInput) (*invoiceOptionsOutput, error) {
			o, err := s.InvoiceOptions(ctx, in.ID)
			return &invoiceOptionsOutput{Body: o}, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-purchase-invoices", Method: http.MethodGet, Path: "/purchase/invoices"},
		func(ctx context.Context, in *InvoiceFilter) (*invoicesOutput, error) {
			l, err := s.Invoices(ctx, *in)
			return &invoicesOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-purchase-invoice", Method: http.MethodPost, Path: "/purchase/invoices", DefaultStatus: http.StatusCreated,
		Description: "Sending the same request_id again returns the invoice created the first time."},
		func(ctx context.Context, in *newInvoiceInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateInvoice(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-purchase-invoice", Method: http.MethodGet, Path: "/purchase/invoices/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*invoiceOutput, error) {
			v, err := s.Invoice(ctx, in.ID)
			return &invoiceOutput{Body: v}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-purchase-invoice", Method: http.MethodPut, Path: "/purchase/invoices/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateInvoiceInput) (*struct{}, error) {
			return nil, s.UpdateInvoice(ctx, in.ID, in.Body)
		})
}
