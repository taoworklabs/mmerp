package sales

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type invoiceOutput struct{ Body Invoice }

type newInvoiceInput struct{ Body NewInvoice }

type updateInvoiceInput struct {
	ID   int64 `path:"id"`
	Body InvoiceUpdate
}

type invoiceOptionsOutput struct{ Body InvoiceOptions }

func registerInvoiceHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "get-invoice-options", Method: http.MethodGet, Path: "/sales/orders/{id}/invoice-options"},
		func(ctx context.Context, in *platform.IDInput) (*invoiceOptionsOutput, error) {
			o, err := s.InvoiceOptions(ctx, in.ID)
			return &invoiceOptionsOutput{Body: o}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-invoice", Method: http.MethodPost, Path: "/sales/invoices", DefaultStatus: http.StatusCreated,
		Description: "Sending the same request_id again returns the invoice created the first time."},
		func(ctx context.Context, in *newInvoiceInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateInvoice(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-invoice", Method: http.MethodGet, Path: "/sales/invoices/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*invoiceOutput, error) {
			v, err := s.Invoice(ctx, in.ID)
			return &invoiceOutput{Body: v}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-invoice", Method: http.MethodPut, Path: "/sales/invoices/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateInvoiceInput) (*struct{}, error) {
			return nil, s.UpdateInvoice(ctx, in.ID, in.Body)
		})
}
