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
	err := s.writeItem(ctx, func(ctx context.Context) error {
		var err error
		if in.Code, err = s.d.Numbering.Code(ctx, itemCodeSeries, in.Code, itemCodeTaken); err != nil {
			return err
		}
		in.ItemFields = trimItem(in.ItemFields)
		id, err = store.CreateItem.Run(ctx, store.CreateItemParams{Code: in.Code, Name: in.Name, Unit: in.Unit, Price: in.Price, VatRate: in.VatRate, Active: in.Active})
		if err != nil {
			return err
		}
		return s.auditItem(ctx, id, in.Code, in.ItemFields)
	})
	return id, err
}

// UpdateItem replaces an item's fields; its code never changes.
func (s *Service) UpdateItem(ctx context.Context, id int64, in ItemFields) error {
	return s.writeItem(ctx, func(ctx context.Context) error {
		in = trimItem(in)
		code, err := store.UpdateItem.Run(ctx, store.UpdateItemParams{ID: id, Name: in.Name, Unit: in.Unit, Price: in.Price, VatRate: in.VatRate, Active: in.Active})
		if errors.Is(err, pgx.ErrNoRows) {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		return s.auditItem(ctx, id, code, in)
	})
}

func (s *Service) writeItem(ctx context.Context, fn func(context.Context) error) error {
	if err := platform.ProductGate(ctx, "sales", platform.ClassWrite); err != nil {
		return err
	}
	if err := s.d.IAM.RequireTenantWide(ctx, "sales", PermItemManage); err != nil {
		return err
	}
	return platform.InTx(ctx, fn)
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
	return store.ItemCodeTaken.Run(ctx, code)
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

// ItemFields are what an edit may change; the code is set once, at creation.
type ItemFields struct {
	Name    string `json:"name" minLength:"1" maxLength:"300"`
	Unit    string `json:"unit" minLength:"1" maxLength:"30"`
	Price   int64  `json:"price" minimum:"0" doc:"Default unit price in đồng"`
	VatRate string `json:"vat_rate" enum:"none,0,5,8,10"`
	Active  bool   `json:"active"`
}

type NewItem struct {
	Code string `json:"code,omitempty" maxLength:"50" doc:"Empty or absent: issued from the sales.item code series"`
	ItemFields
}

type Item struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	ItemFields
}

type ItemFilter struct {
	Q      string `query:"q" maxLength:"100"`
	Active string `query:"active" enum:"true,false"`
	Sort   string `query:"sort" enum:"code,-code,name,-name" default:"code"`
	platform.Paging
}

type ItemList struct {
	Items []Item `json:"items" nullable:"false"`
	Total int64  `json:"total"`
}
