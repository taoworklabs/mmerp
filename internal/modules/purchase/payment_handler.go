package purchase

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type paymentOutput struct{ Body Payment }

type paymentsOutput struct{ Body PaymentList }

type newPaymentInput struct{ Body NewPayment }

type updatePaymentInput struct {
	ID   int64 `path:"id"`
	Body PaymentUpdate
}

type openInvoicesInput struct {
	ID        int64 `path:"id"`
	OrgUnitID int64 `query:"org_unit_id" required:"true"`
	PaymentID int64 `query:"payment_id" doc:"The payment being edited, whose own allocations are not held back"`
}

type openInvoicesOutput struct {
	Body struct {
		Items []Allocation `json:"items" nullable:"false"`
	}
}

func registerPaymentHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-supplier-open-invoices", Method: http.MethodGet, Path: "/purchase/suppliers/{id}/open-invoices",
		Description: "Posted invoices of the supplier at the org unit with something left to pay, oldest first; amount is the remaining."},
		func(ctx context.Context, in *openInvoicesInput) (*openInvoicesOutput, error) {
			items, err := s.OpenInvoices(ctx, in.ID, in.OrgUnitID, in.PaymentID)
			out := &openInvoicesOutput{}
			out.Body.Items = items
			return out, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-payment-actions", Method: http.MethodGet, Path: "/purchase/payments/actions"},
		func(ctx context.Context, _ *struct{}) (*actionsOutput, error) {
			a, err := s.PaymentActions(ctx)
			return &actionsOutput{Body: a}, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-payments", Method: http.MethodGet, Path: "/purchase/payments"},
		func(ctx context.Context, in *PaymentFilter) (*paymentsOutput, error) {
			l, err := s.Payments(ctx, *in)
			return &paymentsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-payment", Method: http.MethodPost, Path: "/purchase/payments", DefaultStatus: http.StatusCreated,
		Description: "Sending the same request_id again returns the payment created the first time."},
		func(ctx context.Context, in *newPaymentInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreatePayment(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-payment", Method: http.MethodGet, Path: "/purchase/payments/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*paymentOutput, error) {
			r, err := s.Payment(ctx, in.ID)
			return &paymentOutput{Body: r}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-payment", Method: http.MethodPut, Path: "/purchase/payments/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updatePaymentInput) (*struct{}, error) {
			return nil, s.UpdatePayment(ctx, in.ID, in.Body)
		})
}
