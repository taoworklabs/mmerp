package purchase_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/purchase"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/item"
	"github.com/taoworklabs/mmerp/internal/shared/partner"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

// fixture: company C with purchasing teams A and B, a staff user and a manager in A, a
// supplier of A, and three items: lot-tracked goods at 8 %, serial-tracked goods at 10 % and
// a transport service at 10 %.
type fixture struct {
	t                      *testing.T
	core                   *coretest.Core
	ctx                    context.Context
	admin                  context.Context
	staff                  context.Context // staff@A
	manager                context.Context // manager@A
	rec                    *record.Service
	appr                   *approval.Service
	purchase               *purchase.Service
	partners               *partner.Service
	items                  *item.Service
	stock                  *stock.Service
	c, a, b                int64
	supplier               int64
	lot, serial, transport int64
	ctxFor                 func(login string, grants ...string) context.Context
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	core := coretest.New(t, "purchase")
	ctx, admin, ids, rec, num := core.Ctx, core.Admin, core.IAM, core.Record, core.Numbering
	partners := partner.NewService(partner.Deps{IAM: ids, Record: rec, Audit: core.Audit, Numbering: num})
	stk := stock.NewService(stock.Deps{IAM: ids, Audit: core.Audit, Numbering: num, Record: rec, Printing: core.Printing})
	s := purchase.NewService(purchase.Deps{IAM: ids, Record: rec, Audit: core.Audit, Setting: core.Setting, Printing: core.Printing,
		Numbering: num, Partner: partners, Stock: stk, Posting: posting.NewService(posting.Hooks{})})
	f := &fixture{t: t, core: core, ctx: ctx, admin: admin, rec: rec, appr: core.Approval, purchase: s, partners: partners, stock: stk}
	f.c = must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	f.a = must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{ParentID: &f.c, Kind: "department", Name: "A"}))
	f.b = must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{ParentID: &f.c, Kind: "department", Name: "B"}))
	f.ctxFor = func(login string, grants ...string) context.Context {
		t.Helper()
		id := must(t)(ids.CreateUser(admin, login, login, "long enough"))
		for _, g := range grants {
			role, unit, _ := strings.Cut(g, "@")
			u := map[string]*int64{"A": &f.a, "B": &f.b, "*": nil}[unit]
			product := "purchase"
			if p, r, ok := strings.Cut(role, "."); ok {
				product, role = p, r
			}
			must(t)(ids.GrantRole(admin, id, product, role, u))
		}
		return platform.WithActor(ctx, id)
	}
	f.staff = f.ctxFor("staff", "staff@A", "partner.editor@A")
	f.manager = f.ctxFor("manager", "manager@A", "partner.editor@A")
	f.items = item.NewService(item.Deps{IAM: ids, Audit: core.Audit, Numbering: num})
	tr := func(v string) *string { return &v }
	f.lot = must(t)(f.items.CreateItem(admin, item.NewItem{Code: "SUA", ItemFields: item.ItemFields{Name: "Sữa hộp", Unit: "thùng", Price: 50_000, VatRate: "8", Active: true, Tracking: tr("lot")}}))
	f.serial = must(t)(f.items.CreateItem(admin, item.NewItem{Code: "MAY", ItemFields: item.ItemFields{Name: "Máy lọc nước", Unit: "cái", Price: 3_000_000, VatRate: "10", Active: true, Tracking: tr("serial")}}))
	f.transport = must(t)(f.items.CreateItem(admin, item.NewItem{Code: "VC", ItemFields: item.ItemFields{Name: "Vận chuyển", Unit: "chuyến", Price: 500_000, VatRate: "10", Active: true}}))
	f.supplier = must(t)(s.CreateSupplier(f.staff, purchase.NewSupplier{Code: "NCC1",
		Partner:        &partner.PartnerFields{Name: "Công ty NCC1", TaxCode: str("0301234567"), Address: str("2 Hùng Vương, Huế")},
		SupplierFields: purchase.SupplierFields{PaymentTerms: str("15 ngày"), OrgUnitID: f.a, Active: true}}))
	return f
}

var requests atomic.Int64

func newRequest() string {
	return fmt.Sprintf("00000000-0000-4000-9000-%012d", requests.Add(1))
}

// orderFields is a draft of team A for the fixture's supplier: 10 cartons at 50,000 đ and 8 %,
// 2 purifiers at 3,000,000 đ and 10 %, one transport at 500,000 đ and 10 %.
func (f *fixture) orderFields(date string) purchase.OrderFields {
	return purchase.OrderFields{Date: date, OrgUnitID: f.a, SupplierID: f.supplier, PaymentTerms: str("15 ngày"),
		Lines: []purchase.LineInput{
			{ItemID: f.lot, Description: "Sữa hộp", Quantity: "10", UnitPrice: 50_000, DiscountPercent: "0", VatRate: "8"},
			{ItemID: f.serial, Description: "Máy lọc nước", Quantity: "2", UnitPrice: 3_000_000, DiscountPercent: "0", VatRate: "10"},
			{ItemID: f.transport, Description: "Vận chuyển", Quantity: "1", UnitPrice: 500_000, DiscountPercent: "0", VatRate: "10"},
		}}
}

// postedOrder is a confirmed order of the fixture's lines, dated date.
func (f *fixture) postedOrder(date string) int64 {
	f.t.Helper()
	id := must(f.t)(f.purchase.CreateOrder(f.staff, purchase.NewOrder{RequestID: newRequest(), OrderFields: f.orderFields(date)}))
	f.check(f.move(f.staff, "purchase.order", id, record.Posted))
	return id
}

func (f *fixture) move(ctx context.Context, docType string, id int64, to record.Status) error {
	d, err := f.rec.Get(f.admin, record.Ref{Type: docType, ID: id})
	if err != nil {
		return err
	}
	_, err = f.rec.Transition(ctx, record.Ref{Type: docType, ID: id}, d.Version, to)
	return err
}

// managerApproves routes action on docType to team managers.
func (f *fixture) managerApproves(docType string, action record.Action) {
	f.check(f.appr.SaveRule(f.admin, docType, action, approval.RuleInput{
		Flow:      approval.Flow{{Step: &approval.StepDef{Approver: approval.Approver{Kind: "role", Product: "purchase", Role: "manager"}}}},
		MaxLevels: 3, FallbackProduct: "purchase", FallbackRole: "manager",
	}))
}

func must(t *testing.T) func(int64, error) int64 {
	return func(v int64, err error) int64 {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func (f *fixture) check(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func str(s string) *string { return &s }

func TestPurchaseRoles(t *testing.T) {
	f := newFixture(t)
	viewer := f.ctxFor("viewer", "viewer@A")
	for _, c := range []struct {
		ctx  context.Context
		perm string
		want bool
	}{
		{f.staff, purchase.PermOrderEdit, true},
		{f.staff, purchase.PermSupplierEdit, true},
		{viewer, purchase.PermOrderView, true},
		{viewer, purchase.PermOrderEdit, false},
	} {
		if got := f.core.IAM.Require(c.ctx, "purchase", c.perm, f.a) == nil; got != c.want {
			t.Errorf("%s = %v, want %v", c.perm, got, c.want)
		}
	}
}

func must2(_ int64, err error) error { return err }

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	if e, ok := errors.AsType[*platform.Error](err); !ok || e.Code != code {
		t.Fatalf("got %v, want %s", err, code)
	}
}
