package purchase_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/modules/purchase"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/partner"
)

func supplier(code string, unit int64) purchase.NewSupplier {
	return purchase.NewSupplier{Code: code, Partner: &partner.PartnerFields{Name: "Công ty " + code, Address: str("2 Hùng Vương, Huế")},
		SupplierFields: purchase.SupplierFields{PaymentTerms: str("15 ngày"), OrgUnitID: unit, Active: true}}
}

func updSupplier(c purchase.NewSupplier) purchase.SupplierUpdate {
	return purchase.SupplierUpdate{SupplierFields: c.SupplierFields, Partner: c.Partner}
}

var page = platform.Paging{Page: 1, PageSize: 50}

func TestSupplierScope(t *testing.T) {
	f := newFixture(t)
	id := must(t)(f.purchase.CreateSupplier(f.staff, supplier("NCC1", f.a)))
	staffB := f.ctxFor("staff_b", "staff@B", "partner.editor@B")
	viewer := f.ctxFor("viewer", "viewer@*", "partner.viewer@*")

	if _, err := f.purchase.CreateSupplier(f.staff, supplier("NCCB", f.b)); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("create in B as staff of A: %v", err)
	}
	wantCode(t, must2(f.purchase.CreateSupplier(staffB, supplier("ncc1", f.b))), "supplier_code_taken")
	if l, err := f.purchase.Suppliers(staffB, purchase.SupplierFilter{Sort: "code", Paging: page}); err != nil || l.Total != 0 {
		t.Fatalf("B lists %+v %v", l, err)
	}
	if _, err := f.purchase.Supplier(staffB, id); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("B opens A's supplier: %v", err)
	}
	if err := f.purchase.UpdateSupplier(staffB, id, updSupplier(supplier("NCC1", f.b))); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("B edits A's supplier: %v", err)
	}
	s, err := f.purchase.Supplier(viewer, id)
	if err != nil || len(s.AllowedActions) != 0 || s.Partner.Name != "Công ty NCC1" || *s.PaymentTerms != "15 ngày" {
		t.Fatalf("viewer: %+v %v", s, err)
	}
	if s, _ := f.purchase.Supplier(f.staff, id); !slices.Equal(s.AllowedActions, []string{"edit"}) {
		t.Fatalf("staff actions %v", s.AllowedActions)
	}
	if l, err := f.purchase.Suppliers(f.staff, purchase.SupplierFilter{Q: "ncc1", Sort: "code", Paging: page}); err != nil || l.Total != 1 || l.Items[0].Name != "Công ty NCC1" {
		t.Fatalf("A lists %+v %v", l, err)
	}
}

// One company may be supplier of several teams, and a customer of Sales too: one partner, many roles.
func TestSupplierOnExistingPartner(t *testing.T) {
	f := newFixture(t)
	p := must(t)(f.partners.Create(f.staff, "", partner.PartnerFields{Name: "Công ty Hai Vai"}))
	in := purchase.NewSupplier{PartnerID: &p, SupplierFields: purchase.SupplierFields{OrgUnitID: f.a, Active: true}}
	id := must(t)(f.purchase.CreateSupplier(f.staff, in))
	s, err := f.purchase.Supplier(f.staff, id)
	if err != nil || s.PartnerID != p || s.Partner.Name != "Công ty Hai Vai" || s.Code != "NC00001" {
		t.Fatalf("supplier %+v %v", s, err)
	}
	wantCode(t, must2(f.purchase.CreateSupplier(f.staff, in)), "supplier_exists_for_partner")
	in.Partner = &partner.PartnerFields{Name: "X"}
	wantCode(t, must2(f.purchase.CreateSupplier(f.staff, in)), "supplier_partner_required")
}

func TestSupplierEditWithoutPartnerEdit(t *testing.T) {
	f := newFixture(t)
	id := must(t)(f.purchase.CreateSupplier(f.staff, supplier("NCC1", f.a)))
	ctx := f.ctxFor("staff_only", "staff@A")
	c := supplier("NCC1", f.a)
	c.PaymentTerms = str("60 ngày")
	c.Partner = nil
	f.check(f.purchase.UpdateSupplier(ctx, id, updSupplier(c)))
	c.Partner = &partner.PartnerFields{Name: "Đổi tên"}
	wantCode(t, f.purchase.UpdateSupplier(ctx, id, updSupplier(c)), "forbidden")
	if got, _ := f.purchase.Supplier(ctx, id); got.Partner.Name != "Công ty NCC1" || *got.PaymentTerms != "60 ngày" {
		t.Fatalf("supplier %+v", got)
	}
}

func TestSupplierWritePurchaseDisabled(t *testing.T) {
	f := newFixture(t)
	off := platform.WithProducts(f.staff, nil)
	wantCode(t, must2(f.purchase.CreateSupplier(off, supplier("NCC2", f.a))), "product_not_enabled")
}
