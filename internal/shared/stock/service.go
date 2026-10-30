// Package stock owns warehouses and the stock ledger every product moves goods through.
package stock

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/shared/stock/internal/store"
)

//go:embed i18n/*.json
var i18nFiles embed.FS

var loadTranslations = sync.OnceValue(func() error {
	sub, err := fs.Sub(i18nFiles, "i18n")
	if err != nil {
		return err
	}
	return platform.LoadTranslations(sub)
})

const warehouseSeries = "stock.warehouse"

type Service struct{ d Deps }

// NewService registers the stock roles, the warehouse code series and the adjustment
// document type; no product gates them.
func NewService(d Deps) *Service {
	if err := loadTranslations(); err != nil {
		panic(err) // embedded files: broken only by a bad build
	}
	data.RegisterConstraints(data.Constraints{"stock.warehouses_code_key": ErrWarehouseCodeTaken})
	d.IAM.RegisterRoles("stock", map[string][]string{
		"viewer": {PermView},
		"admin":  {PermView, PermManage},
	}, "admin")
	d.Numbering.RegisterSeries(warehouseSeries, "", "KHO")
	s := &Service{d: d}
	s.registerAdjustment()
	return s
}

// Warehouses lists them to anyone who may view stock at some org unit.
func (s *Service) Warehouses(ctx context.Context, f WarehouseFilter) (WarehouseList, error) {
	out := WarehouseList{Items: []Warehouse{}}
	sc, err := s.d.IAM.Scope(ctx, "stock", PermView)
	if err != nil {
		return out, err
	}
	if !sc.Any() {
		return out, platform.ErrForbidden
	}
	page, err := warehouseList.Run(ctx, warehouseQuery(f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, Warehouse{ID: r.ID, Code: r.Code, IsDefault: r.IsDefault,
			WarehouseFields: WarehouseFields{Name: r.Name, LegalEntityID: r.LegalEntityID, Active: r.Active}})
	}
	return out, nil
}

// CreateWarehouse adds a warehouse to a legal entity; warehouses are managed tenant-wide.
func (s *Service) CreateWarehouse(ctx context.Context, in NewWarehouse) (int64, error) {
	var id int64
	err := s.write(ctx, func(ctx context.Context) error {
		var err error
		if in.Code, err = s.d.Numbering.Code(ctx, warehouseSeries, in.Code, store.WarehouseCodeTaken.Run); err != nil {
			return err
		}
		in.Name = strings.TrimSpace(in.Name)
		id, err = store.CreateWarehouse.Run(ctx, store.CreateWarehouseParams{Code: in.Code, Name: in.Name, LegalEntityID: in.LegalEntityID, Active: in.Active})
		if errors.Is(err, pgx.ErrNoRows) {
			return platform.ErrNotFound // not a company
		}
		if err != nil {
			return err
		}
		return s.audit(ctx, id, in.Code, in.WarehouseFields)
	})
	return id, err
}

// UpdateWarehouse renames or retires a warehouse; its code and legal entity never change.
func (s *Service) UpdateWarehouse(ctx context.Context, id int64, in WarehouseFields) error {
	return s.write(ctx, func(ctx context.Context) error {
		in.Name = strings.TrimSpace(in.Name)
		code, err := store.UpdateWarehouse.Run(ctx, store.UpdateWarehouseParams{ID: id, Name: in.Name, Active: in.Active})
		if errors.Is(err, pgx.ErrNoRows) {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		return s.audit(ctx, id, code, in)
	})
}

// DefaultWarehouse returns the legal entity's default warehouse, creating it when missing.
// Documents call it inside their own write transaction, so it checks no permission.
func (s *Service) DefaultWarehouse(ctx context.Context, legalEntity int64) (int64, error) {
	if id, err := store.DefaultWarehouse.Run(ctx, legalEntity); err == nil {
		return id, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	code, err := s.d.Numbering.Code(ctx, warehouseSeries, "", store.WarehouseCodeTaken.Run)
	if err != nil {
		return 0, err
	}
	// A concurrent creator wins on the partial unique index; its row is read below.
	if err := store.InsertDefaultWarehouse.Run(ctx, store.InsertDefaultWarehouseParams{
		Code: code, Name: platform.Translate("vi", "stock.warehouse.default_name", nil), LegalEntityID: legalEntity}); err != nil {
		return 0, err
	}
	return store.DefaultWarehouse.Run(ctx, legalEntity)
}

func (s *Service) write(ctx context.Context, fn func(context.Context) error) error {
	if err := s.d.IAM.RequireTenantWide(ctx, "stock", PermManage); err != nil {
		return err
	}
	return platform.InTx(ctx, fn)
}

func (s *Service) audit(ctx context.Context, id int64, code string, in WarehouseFields) error {
	return s.d.Audit.Record(ctx, "stock.warehouse_saved", map[string]any{"id": id, "code": code, "name": in.Name, "active": in.Active})
}
