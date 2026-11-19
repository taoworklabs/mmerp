package purchase

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type goodsReceiptsOutput struct{ Body GoodsReceiptList }

type goodsReceiptOutput struct{ Body GoodsReceipt }

type newGoodsReceiptInput struct{ Body NewGoodsReceipt }

type updateGoodsReceiptInput struct {
	ID   int64 `path:"id"`
	Body GoodsReceiptUpdate
}

type receiptOptionsOutput struct{ Body ReceiptOptions }

func registerReceiptHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "get-purchase-receipt-options", Method: http.MethodGet, Path: "/purchase/orders/{id}/receipt-options"},
		func(ctx context.Context, in *platform.IDInput) (*receiptOptionsOutput, error) {
			o, err := s.ReceiptOptions(ctx, in.ID)
			return &receiptOptionsOutput{Body: o}, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-goods-receipts", Method: http.MethodGet, Path: "/purchase/goods-receipts"},
		func(ctx context.Context, in *GoodsReceiptFilter) (*goodsReceiptsOutput, error) {
			l, err := s.GoodsReceipts(ctx, *in)
			return &goodsReceiptsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-goods-receipt", Method: http.MethodPost, Path: "/purchase/goods-receipts", DefaultStatus: http.StatusCreated,
		Description: "Sending the same request_id again returns the goods receipt created the first time."},
		func(ctx context.Context, in *newGoodsReceiptInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateGoodsReceipt(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-goods-receipt", Method: http.MethodGet, Path: "/purchase/goods-receipts/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*goodsReceiptOutput, error) {
			r, err := s.GoodsReceipt(ctx, in.ID)
			return &goodsReceiptOutput{Body: r}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-goods-receipt", Method: http.MethodPut, Path: "/purchase/goods-receipts/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateGoodsReceiptInput) (*struct{}, error) {
			return nil, s.UpdateGoodsReceipt(ctx, in.ID, in.Body)
		})
}
