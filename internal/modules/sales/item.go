package sales

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
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
