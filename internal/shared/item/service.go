// Package item owns the item catalogue: goods and services every product sells or buys.
package item

import (
	"cmp"
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/shared/item/internal/store"
)

// The code series of items; an item is not a record type.
const codeSeries = "item.item"

type Service struct{ d Deps }

// NewService registers the item roles and code series; no product gates them.
func NewService(d Deps) *Service {
	data.RegisterConstraints(data.Constraints{"item.items_code_key": ErrCodeTaken, "item.items_negative_quantity": ErrNegativeNotQuantity})
	d.IAM.RegisterRoles("item", map[string][]string{
		"viewer": {PermView},
		"admin":  {PermView, PermManage},
	}, "admin")
	d.Numbering.RegisterSeries(codeSeries, "", "HH")
	return &Service{d: d}
}

// Items lists the catalogue to anyone who may view it at some org unit.
func (s *Service) Items(ctx context.Context, f ItemFilter) (ItemList, error) {
	out := ItemList{Items: []Item{}}
	sc, err := s.d.IAM.Scope(ctx, "item", PermView)
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
		out.Items = append(out.Items, Item{ID: r.ID, Code: r.Code, Name: r.Name, Unit: r.Unit, Price: r.Price, VatRate: r.VatRate, Active: r.Active,
			Tracking: r.Tracking, AllowNegative: r.AllowNegative})
	}
	return out, nil
}

// CreateItem adds an item; the catalogue is managed tenant-wide.
func (s *Service) CreateItem(ctx context.Context, in NewItem) (int64, error) {
	var id int64
	err := s.write(ctx, func(ctx context.Context) error {
		var err error
		if in.Code, err = s.d.Numbering.Code(ctx, codeSeries, in.Code, store.ItemCodeTaken.Run); err != nil {
			return err
		}
		in.ItemFields = trim(in.ItemFields)
		tracking := cmp.Or(deref(in.Tracking), TrackNone)
		id, err = store.CreateItem.Run(ctx, store.CreateItemParams{Code: in.Code, Name: in.Name, Unit: in.Unit, Price: in.Price, VatRate: in.VatRate, Active: in.Active,
			Tracking: tracking, AllowNegative: deref(in.AllowNegative)})
		if err != nil {
			return err
		}
		return s.audit(ctx, id, in.Code, in.ItemFields)
	})
	return id, err
}

// UpdateItem replaces an item's fields; its code never changes.
func (s *Service) UpdateItem(ctx context.Context, id int64, in ItemFields) error {
	return s.write(ctx, func(ctx context.Context) error {
		in = trim(in)
		code, err := store.UpdateItem.Run(ctx, store.UpdateItemParams{ID: id, Name: in.Name, Unit: in.Unit, Price: in.Price, VatRate: in.VatRate, Active: in.Active,
			Tracking:      pgtype.Text{String: deref(in.Tracking), Valid: in.Tracking != nil},
			AllowNegative: pgtype.Bool{Bool: deref(in.AllowNegative), Valid: in.AllowNegative != nil}})
		if errors.Is(err, pgx.ErrNoRows) {
			// The statement refused: either no such item or a tracking change after moves.
			found, err := store.ItemExists.Run(ctx, id)
			if err != nil {
				return err
			}
			if !found {
				return platform.ErrNotFound
			}
			return ErrTrackingLocked
		}
		if err != nil {
			return err
		}
		return s.audit(ctx, id, code, in)
	})
}

func (s *Service) write(ctx context.Context, fn func(context.Context) error) error {
	if err := s.d.IAM.RequireTenantWide(ctx, "item", PermManage); err != nil {
		return err
	}
	return platform.InTx(ctx, fn)
}

func deref[T any](p *T) (v T) {
	if p != nil {
		v = *p
	}
	return v
}

func trim(in ItemFields) ItemFields {
	in.Name, in.Unit = strings.TrimSpace(in.Name), strings.TrimSpace(in.Unit)
	return in
}

func (s *Service) audit(ctx context.Context, id int64, code string, in ItemFields) error {
	return s.d.Audit.Record(ctx, "item.item_saved", map[string]any{"id": id, "code": code, "name": in.Name, "unit": in.Unit,
		"price": in.Price, "vat_rate": in.VatRate, "active": in.Active, "tracking": in.Tracking, "allow_negative": in.AllowNegative})
}
