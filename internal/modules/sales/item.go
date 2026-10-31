package sales

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

// itemOptionList feeds the approval item picker over the shared catalogue.
var itemOptionList = data.Register(data.List[store.ItemOptionsBaseRow]{
	Name:     "sales.ItemOptions",
	Base:     store.ItemOptionsBase,
	Scope:    data.Unscoped("tenant-wide catalogue"),
	Search:   []string{"code", "name"},
	Default:  []data.Order{data.Asc("code").Lower()},
	Tiebreak: data.Asc("id"),
})

// itemOptions finds items for an approval condition; it includes retired items, since a rule may name an item that is later retired.
func (s *Service) itemOptions(ctx context.Context, q string) ([]record.FieldOption, error) {
	page, err := itemOptionList.Run(ctx, data.Query{Q: q, Paging: platform.Paging{Page: 1, PageSize: 20}})
	if err != nil {
		return nil, err
	}
	out := make([]record.FieldOption, len(page.Items))
	for i, r := range page.Items {
		out[i] = record.FieldOption{ID: r.ID, Code: r.Code, Name: r.Name}
	}
	return out, nil
}

// itemLabels is the "items" approval field's Labels: items by id, for showing a saved
// condition before any search.
func (s *Service) itemLabels(ctx context.Context, ids []int64) ([]record.FieldOption, error) {
	rows, err := store.ItemsByID.Run(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]record.FieldOption, len(rows))
	for i, r := range rows {
		out[i] = record.FieldOption{ID: r.ID, Code: r.Code, Name: r.Name}
	}
	return out, nil
}

// lineItemList feeds the item cell of a quotation or order line.
var lineItemList = data.Register(data.List[store.ItemItem]{
	Name:     "sales.LineItems",
	Base:     store.LineItemsBase,
	Scope:    data.Unscoped("tenant-wide catalogue"),
	Search:   []string{"code", "name"},
	Default:  []data.Order{data.Asc("code").Lower()},
	Tiebreak: data.Asc("id"),
})

// LineItems finds active items for a document line. Picking an item is part of writing a
// quotation, order or stock adjustment, so writing any of them is enough; the catalogue
// screen still needs item view.
func (s *Service) LineItems(ctx context.Context, q string) (LineItemList, error) {
	out := LineItemList{Items: []LineItem{}}
	for _, p := range [][2]string{{"sales", PermQuoteEdit}, {"sales", PermOrderEdit}, {"stock", stock.PermManage}} {
		sc, err := s.d.IAM.Scope(ctx, p[0], p[1])
		if err != nil {
			return out, err
		}
		if sc.Any() {
			page, err := lineItemList.Run(ctx, data.Query{Q: q, Paging: platform.Paging{Page: 1, PageSize: 20}, Where: data.Eq("active", true)})
			if err != nil {
				return out, err
			}
			for _, r := range page.Items {
				out.Items = append(out.Items, LineItem{ID: r.ID, Code: r.Code, Name: r.Name, Unit: r.Unit, Price: r.Price, VatRate: r.VatRate, Tracking: r.Tracking})
			}
			return out, nil
		}
	}
	return out, platform.ErrForbidden
}

// LineItem is what a line takes from the catalogue: the name for its description, the
// default price and VAT rate; stock lines take the tracking.
type LineItem struct {
	ID       int64  `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Unit     string `json:"unit"`
	Price    int64  `json:"price"`
	VatRate  string `json:"vat_rate" enum:"none,0,5,8,10"`
	Tracking string `json:"tracking" enum:"none,quantity,lot,serial"`
}

type LineItemList struct {
	Items []LineItem `json:"items" nullable:"false"`
}
