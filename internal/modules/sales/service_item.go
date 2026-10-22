package sales

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
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
	page, err := itemList.Run(ctx, itemQuery(f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, Item{ID: r.ID, Code: r.Code, ItemFields: ItemFields{Name: r.Name, Unit: r.Unit, Price: r.Price, VatRate: r.VatRate, Active: r.Active}})
	}
	return out, nil
}

// CreateItem adds an item; the catalogue is managed tenant-wide.
func (s *Service) CreateItem(ctx context.Context, in NewItem) (int64, error) {
	var id int64
	err := s.writeItem(ctx, func(ctx context.Context, q *store.Queries) error {
		var err error
		if in.Code, err = s.d.Numbering.Code(ctx, itemCodeSeries, in.Code, itemCodeTaken); err != nil {
			return err
		}
		in.ItemFields = trimItem(in.ItemFields)
		id, err = q.CreateItem(ctx, store.CreateItemParams{Code: in.Code, Name: in.Name, Unit: in.Unit, Price: in.Price, VatRate: in.VatRate, Active: in.Active})
		if platform.Violates(err, "items_code_key") {
			return ErrItemCodeTaken
		}
		if err != nil {
			return err
		}
		return s.auditItem(ctx, id, in.Code, in.ItemFields)
	})
	return id, err
}

// UpdateItem replaces an item's fields; its code never changes.
func (s *Service) UpdateItem(ctx context.Context, id int64, in ItemFields) error {
	return s.writeItem(ctx, func(ctx context.Context, q *store.Queries) error {
		in = trimItem(in)
		code, err := q.UpdateItem(ctx, store.UpdateItemParams{ID: id, Name: in.Name, Unit: in.Unit, Price: in.Price, VatRate: in.VatRate, Active: in.Active})
		if errors.Is(err, pgx.ErrNoRows) {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		return s.auditItem(ctx, id, code, in)
	})
}

func (s *Service) writeItem(ctx context.Context, fn func(context.Context, *store.Queries) error) error {
	if err := platform.ProductGate(ctx, "sales", platform.ClassWrite); err != nil {
		return err
	}
	if err := s.d.IAM.RequireTenantWide(ctx, "sales", PermItemManage); err != nil {
		return err
	}
	return platform.InTx(ctx, func(ctx context.Context) error { return fn(ctx, store.New(platform.DBFrom(ctx))) })
}

func trimItem(in ItemFields) ItemFields {
	in.Name, in.Unit = strings.TrimSpace(in.Name), strings.TrimSpace(in.Unit)
	return in
}

func (s *Service) auditItem(ctx context.Context, id int64, code string, in ItemFields) error {
	return s.d.Audit.Record(ctx, "sales.item_saved", map[string]any{"id": id, "code": code, "name": in.Name, "unit": in.Unit,
		"price": in.Price, "vat_rate": in.VatRate, "active": in.Active})
}

// itemCodeTaken reports whether a code is used, for numbering to skip it.
func itemCodeTaken(ctx context.Context, code string) (bool, error) {
	return store.New(platform.DBFrom(ctx)).ItemCodeTaken(ctx, code)
}

// itemOptions finds items for an approval condition; it includes retired items, since a rule may name an item that is later retired.
func (s *Service) itemOptions(ctx context.Context, q string) ([]record.FieldOption, error) {
	page, err := itemList.Run(ctx, data.Query{Q: q, Paging: platform.Paging{Page: 1, PageSize: 20}})
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
	rows, err := store.New(platform.DBFrom(ctx)).ItemsByID(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]record.FieldOption, len(rows))
	for i, r := range rows {
		out[i] = record.FieldOption{ID: r.ID, Code: r.Code, Name: r.Name}
	}
	return out, nil
}
