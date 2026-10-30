package stock_test

import (
	"context"
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

const testDoc = "stock.testdoc"

type fixture struct {
	core  *coretest.Core
	stock *stock.Service
	unit  int64
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	core := coretest.New(t)
	f := &fixture{core: core}
	var err error
	if f.unit, err = core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "C"}); err != nil {
		t.Fatal(err)
	}
	f.stock = stock.NewService(stock.Deps{IAM: core.IAM, Audit: core.Audit, Numbering: core.Numbering, Record: core.Record, Printing: core.Printing})
	// A bare document type, so ledger tests have documents to move stock for.
	core.Record.Register(record.Type{Code: testDoc, Product: "stock", Shared: true, Kind: record.Document, NumberPrefix: "TT",
		Perms: record.Perms{View: stock.PermView, Edit: stock.PermManage}, OnDelete: func(context.Context, int64) error { return nil }})
	return f
}

// user holds stock roles: "admin" tenant-wide, "viewer" at the company.
func (f *fixture) user(t *testing.T, login string, roles ...string) context.Context {
	t.Helper()
	id, err := f.core.IAM.CreateUser(f.core.Admin, login, login, "long enough")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range roles {
		var unit *int64
		if r == "viewer" {
			unit = &f.unit
		}
		if _, err := f.core.IAM.GrantRole(f.core.Admin, id, "stock", r, unit); err != nil {
			t.Fatal(err)
		}
	}
	return platform.WithActor(f.core.Ctx, id)
}

var page = platform.Paging{Page: 1, PageSize: 50}

func TestDefaultWarehouseBackfilledAndLazy(t *testing.T) {
	f := newFixture(t)
	viewer := f.user(t, "v", "viewer")
	if l, err := f.stock.Warehouses(viewer, stock.WarehouseFilter{Paging: page}); err != nil || l.Total != 0 {
		t.Fatalf("before: %+v %v", l, err)
	}
	var first, second int64
	err := platform.InTx(f.core.Ctx, func(ctx context.Context) (err error) {
		if first, err = f.stock.DefaultWarehouse(ctx, f.unit); err != nil {
			return err
		}
		second, err = f.stock.DefaultWarehouse(ctx, f.unit)
		return err
	})
	if err != nil || first == 0 || first != second {
		t.Fatalf("default %d %d %v", first, second, err)
	}
	l, err := f.stock.Warehouses(viewer, stock.WarehouseFilter{Paging: page})
	if err != nil || l.Total != 1 || !l.Items[0].IsDefault || l.Items[0].Name != "Kho chính" || l.Items[0].Code != "KHO00001" || l.Items[0].LegalEntityID != f.unit {
		t.Fatalf("after: %+v %v", l, err)
	}
	// A later default for the same legal entity cannot exist.
	_, err = f.core.System.Exec(f.core.Ctx, `INSERT INTO stock.warehouses (code, name, legal_entity_id, is_default) VALUES ('X', 'X', $1, true)`, f.unit)
	if err == nil {
		t.Fatal("second default accepted")
	}
}

func TestWarehousePermissions(t *testing.T) {
	f := newFixture(t)
	viewer, admin := f.user(t, "v", "viewer"), f.user(t, "a", "admin")
	in := stock.NewWarehouse{WarehouseFields: stock.WarehouseFields{Name: " Kho A ", LegalEntityID: f.unit, Active: true}}
	if _, err := f.stock.CreateWarehouse(viewer, in); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("viewer creates: %v", err)
	}
	id, err := f.stock.CreateWarehouse(admin, in)
	if err != nil {
		t.Fatal(err)
	}
	l, err := f.stock.Warehouses(viewer, stock.WarehouseFilter{Paging: page})
	if err != nil || l.Total != 1 || l.Items[0].ID != id || l.Items[0].Code != "KHO00001" || l.Items[0].Name != "Kho A" || l.Items[0].IsDefault {
		t.Fatalf("list %+v %v", l, err)
	}
	if err := f.stock.UpdateWarehouse(admin, id, stock.WarehouseFields{Name: "Kho B", LegalEntityID: f.unit}); err != nil {
		t.Fatal(err)
	}
	if err := f.stock.UpdateWarehouse(admin, id+100, stock.WarehouseFields{Name: "X", LegalEntityID: f.unit}); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("update missing: %v", err)
	}
	if _, err := f.stock.Warehouses(f.user(t, "n"), stock.WarehouseFilter{Paging: page}); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("nobody lists: %v", err)
	}
}
