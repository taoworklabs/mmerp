package stock

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/stock/internal/store"
)

// OnHand lists current balances to anyone who may view stock somewhere.
func (s *Service) OnHand(ctx context.Context, f OnHandFilter) (OnHandList, error) {
	out := OnHandList{Items: []OnHandRow{}}
	if err := s.requireView(ctx); err != nil {
		return out, err
	}
	page, err := onHandList.Run(ctx, onHandQuery(f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, OnHandRow{WarehouseID: r.WarehouseID, WarehouseCode: r.WarehouseCode, ItemID: r.ItemID,
			ItemCode: r.ItemCode, ItemName: r.ItemName, Unit: r.Unit, LotCode: platform.TextPtr(r.LotCode), Expiry: platform.DatePtr(r.LotExpiry),
			SerialCode: platform.TextPtr(r.SerialCode), Quantity: r.Quantity})
	}
	return out, nil
}

func (s *Service) requireView(ctx context.Context) error {
	sc, err := s.d.IAM.Scope(ctx, "stock", PermView)
	if err != nil {
		return err
	}
	if !sc.Any() {
		return platform.ErrForbidden
	}
	return nil
}

// StockCard lists the moves of an item in a warehouse from..to (YYYY-MM-DD) with a running balance.
// It is not paged: the date range, at most maxStockCardDays long, bounds it.
func (s *Service) StockCard(ctx context.Context, warehouse, item int64, from, to string) (StockCard, error) {
	out := StockCard{Rows: []StockCardRow{}}
	if err := s.requireView(ctx); err != nil {
		return out, err
	}
	f, errF := time.Parse(time.DateOnly, from)
	t, errT := time.Parse(time.DateOnly, to)
	switch {
	case errF != nil || errT != nil || f.After(t):
		return out, ErrStockCardRange
	case t.Sub(f) >= maxStockCardDays*24*time.Hour:
		return out, ErrStockCardRangeLen
	}
	open, err := store.StockCardOpening.Run(ctx, store.StockCardOpeningParams{WarehouseID: warehouse, ItemID: item, FromDate: platform.NullDate(&from)})
	if err != nil {
		return out, err
	}
	rows, err := store.StockCardMoves.Run(ctx, store.StockCardMovesParams{WarehouseID: warehouse, ItemID: item,
		FromDate: platform.NullDate(&from), ToDate: platform.NullDate(&to)})
	if err != nil {
		return out, err
	}
	bal, err := decimal.NewFromString(open)
	if err != nil {
		return out, err
	}
	out.Opening = bal.StringFixed(3)
	for _, r := range rows {
		q, err := decimal.NewFromString(r.Quantity)
		if err != nil {
			return out, err
		}
		bal = bal.Add(q)
		out.Rows = append(out.Rows, StockCardRow{Date: *platform.DatePtr(r.Date), DocType: r.DocType, DocID: r.DocID, Number: r.Number,
			LotCode: platform.TextPtr(r.LotCode), SerialCode: platform.TextPtr(r.SerialCode), Quantity: r.Quantity, Balance: bal.StringFixed(3)})
	}
	out.Closing = bal.StringFixed(3)
	return out, nil
}
