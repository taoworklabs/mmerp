package sales

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

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
		Q: strings.TrimSpace(f.Q), Active: boolFilter(f.Active), Sort: f.Sort, Lim: f.Limit(), Off: f.Offset(),
	})
	for _, r := range rows {
		out.Total = r.Total
		out.Items = append(out.Items, Item{ID: r.ID, Code: r.Code, ItemFields: ItemFields{Name: r.Name, Unit: r.Unit, Price: r.Price, VatRate: r.VatRate, Active: r.Active}})
	}
	return out, err
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
