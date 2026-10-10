package sales

import (
	"context"
	"strings"

	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// Items lists the catalogue to anyone who may view it at some org unit.
func (s *Service) Items(ctx context.Context, f ItemFilter) (ItemList, error) {
	out := ItemList{Items: []Item{}}
	sc, err := s.d.IAM.Scope(ctx, "sales", PermItemView)
	if err != nil {
		return out, err
	}
	if !sc.Any() {
		return out, platform.ErrForbidden
	}
	rows, err := store.New(platform.DBFrom(ctx)).ListItems(ctx, store.ListItemsParams{
		Q: strings.TrimSpace(f.Q), Active: boolFilter(f.Active), Lim: int32(f.PageSize), Off: int32((f.Page - 1) * f.PageSize),
	})
	for _, r := range rows {
		out.Total = r.Total
		out.Items = append(out.Items, Item{ID: r.ID, ItemFields: ItemFields{Code: r.Code, Name: r.Name, Unit: r.Unit, Price: r.Price, VatRate: r.VatRate, Active: r.Active}})
	}
	return out, err
}

// SaveItem creates an item (id 0) or replaces one; the catalogue is managed tenant-wide.
func (s *Service) SaveItem(ctx context.Context, id int64, in ItemFields) (int64, error) {
	if err := platform.ProductGate(ctx, "sales", platform.ClassWrite); err != nil {
		return 0, err
	}
	if sc, err := s.d.IAM.Scope(ctx, "sales", PermItemManage); err != nil || !sc.All {
		return 0, platform.OrErr(err, platform.ErrForbidden)
	}
	err := platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		in.Code, in.Name, in.Unit = strings.TrimSpace(in.Code), strings.TrimSpace(in.Name), strings.TrimSpace(in.Unit)
		var err error
		if id == 0 {
			id, err = q.CreateItem(ctx, store.CreateItemParams{Code: in.Code, Name: in.Name, Unit: in.Unit, Price: in.Price, VatRate: in.VatRate, Active: in.Active})
		} else {
			var n int64
			n, err = q.UpdateItem(ctx, store.UpdateItemParams{ID: id, Code: in.Code, Name: in.Name, Unit: in.Unit, Price: in.Price, VatRate: in.VatRate, Active: in.Active})
			if err == nil && n == 0 {
				return platform.ErrNotFound
			}
		}
		if platform.Violates(err, "items_code_key") {
			return ErrItemCodeTaken
		}
		if err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "sales.item_saved", map[string]any{"id": id, "code": in.Code, "name": in.Name, "unit": in.Unit,
			"price": in.Price, "vat_rate": in.VatRate, "active": in.Active})
	})
	return id, err
}
