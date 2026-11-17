package sales

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type receiptOutput struct{ Body Receipt }

type newReceiptInput struct{ Body NewReceipt }

type updateReceiptInput struct {
	ID   int64 `path:"id"`
	Body ReceiptUpdate
}

type openInvoicesInput struct {
	ID        int64 `path:"id"`
	OrgUnitID int64 `query:"org_unit_id" required:"true"`
	ReceiptID int64 `query:"receipt_id" doc:"The receipt being edited, whose own allocations are not held back"`
}

type openInvoicesOutput struct {
	Body struct {
		Items []Allocation `json:"items" nullable:"false"`
	}
}

func registerReceiptHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-open-invoices", Method: http.MethodGet, Path: "/sales/customers/{id}/open-invoices",
		Description: "Posted invoices of the customer at the org unit with something left to receive, oldest first; amount is the remaining."},
		func(ctx context.Context, in *openInvoicesInput) (*openInvoicesOutput, error) {
			items, err := s.OpenInvoices(ctx, in.ID, in.OrgUnitID, in.ReceiptID)
			out := &openInvoicesOutput{}
			out.Body.Items = items
			return out, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-receipt", Method: http.MethodPost, Path: "/sales/receipts", DefaultStatus: http.StatusCreated,
		Description: "Sending the same request_id again returns the receipt created the first time."},
		func(ctx context.Context, in *newReceiptInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateReceipt(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-receipt", Method: http.MethodGet, Path: "/sales/receipts/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*receiptOutput, error) {
			r, err := s.Receipt(ctx, in.ID)
			return &receiptOutput{Body: r}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-receipt", Method: http.MethodPut, Path: "/sales/receipts/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateReceiptInput) (*struct{}, error) {
			return nil, s.UpdateReceipt(ctx, in.ID, in.Body)
		})
}
