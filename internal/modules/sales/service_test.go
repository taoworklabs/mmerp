package sales_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/printing"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/modules/sales"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// fixture: company C with sales teams A and B, a staff user and a manager in A, one
// customer of A and two items.
type fixture struct {
	t        *testing.T
	ctx      context.Context
	admin    context.Context
	staff    context.Context // staff@A
	manager  context.Context // manager@A
	set      *setting.Service
	rec      *record.Service
	appr     *approval.Service
	sales    *sales.Service
	c, a, b  int64
	customer int64
	pen, fix int64 // items: goods at 8 %, a service at 10 %
	ctxFor   func(login string, grants ...string) context.Context
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	pool := pgtest.New(t)
	ctx := platform.WithKeyring(platform.WithProducts(platform.WithDB(t.Context(), pool), []string{"sales"}), pgtest.Keyring())
	set := setting.NewService()
	ids := iam.NewService(iam.Deps{Setting: set, Audit: audit.NewService()})
	set.SetAuthz(ids)
	rec := record.NewService(record.Deps{IAM: ids, Numbering: numbering.NewService(), Audit: audit.NewService()})
	ids.SetTreeHook(rec)
	appr := approval.NewService(approval.Deps{IAM: ids, Record: rec, Audit: audit.NewService(),
		Notification: notification.NewService(notification.Deps{Record: rec, IAM: ids, Audit: audit.NewService(), Setting: set})})
	rec.SetApprovalGate(appr)
	dio := dataio.NewService(dataio.Deps{IAM: ids, Audit: audit.NewService()})
	prn := printing.NewService(printing.Deps{Record: rec, Audit: audit.NewService(), DataIO: dio, IAM: ids})
	s := sales.NewService(sales.Deps{IAM: ids, Record: rec, Audit: audit.NewService(), Setting: set, Printing: prn})
	admin := platform.WithActor(ctx, must(t)(ids.CreateAdmin(ctx, "admin", "Admin", "long enough")))
	f := &fixture{t: t, ctx: ctx, admin: admin, set: set, rec: rec, appr: appr, sales: s}
	f.c = must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	f.a = must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{ParentID: &f.c, Kind: "department", Name: "A"}))
	f.b = must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{ParentID: &f.c, Kind: "department", Name: "B"}))
	f.ctxFor = func(login string, grants ...string) context.Context {
		t.Helper()
		id := must(t)(ids.CreateUser(admin, login, login, "long enough"))
		for _, g := range grants {
			role, unit, _ := strings.Cut(g, "@")
			u := map[string]*int64{"A": &f.a, "B": &f.b, "*": nil}[unit]
			must(t)(ids.GrantRole(admin, id, "sales", role, u))
		}
		return platform.WithActor(ctx, id)
	}
	f.staff, f.manager = f.ctxFor("staff", "staff@A"), f.ctxFor("manager", "manager@A")
	catalog := f.ctxFor("catalog", "catalog_admin@*")
	f.pen = must(t)(s.SaveItem(catalog, 0, sales.ItemFields{Code: "BUT", Name: "Bút bi", Unit: "hộp", Price: 10_005, VatRate: "8", Active: true}))
	f.fix = must(t)(s.SaveItem(catalog, 0, sales.ItemFields{Code: "SUA", Name: "Sửa chữa", Unit: "giờ", Price: 250_000, VatRate: "10", Active: true}))
	f.customer = must(t)(s.CreateCustomer(f.staff, customer("KH1", f.a)))
	return f
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

func customer(code string, unit int64) sales.CustomerFields {
	return sales.CustomerFields{Code: code, Name: "Công ty " + code, TaxCode: str("0101234567"), Address: str("1 Lê Lợi, Huế"),
		PaymentTerms: str("30 ngày"), OrgUnitID: unit, Active: true}
}

var requests atomic.Int64

// fields is a draft of team A for the fixture's customer, dated date: two pens at 8 %
// and an hour of repair at 10 % with a discount.
func (f *fixture) fields(date string) sales.DocFields {
	return sales.DocFields{Date: date, OrgUnitID: f.a, CustomerID: f.customer, ValidUntil: str("2026-12-31"), PaymentTerms: str("30 ngày"),
		Lines: []sales.LineInput{
			{ItemID: f.pen, Description: "Bút bi", Quantity: "1", UnitPrice: 10_005, DiscountPercent: "0", VatRate: "8"},
			{ItemID: f.pen, Description: "Bút bi xanh", Quantity: "1", UnitPrice: 10_005, DiscountPercent: "0", VatRate: "8"},
			{ItemID: f.fix, Description: "Sửa chữa", Quantity: "1.5", UnitPrice: 250_000, DiscountPercent: "10", VatRate: "10"},
		}}
}

func newRequest() string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", requests.Add(1))
}

func (f *fixture) move(ctx context.Context, docType string, id int64, to record.Status) error {
	d, err := f.rec.Get(f.admin, record.Ref{Type: docType, ID: id})
	if err != nil {
		return err
	}
	return f.rec.Transition(ctx, record.Ref{Type: docType, ID: id}, d.Version, to)
}

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	if e, ok := errors.AsType[*platform.Error](err); !ok || e.Code != code {
		t.Fatalf("got %v, want %s", err, code)
	}
}

// managerApproves routes quotations with a discount above 10 % to team managers.
func (f *fixture) managerApproves(docType string) {
	f.check(f.appr.SaveRule(f.admin, docType, approval.RuleInput{
		Steps: []approval.Step{{Condition: &approval.Condition{Field: "max_discount", Op: "gt", Value: "10"},
			Approver: approval.Approver{Kind: "role", Product: "sales", Role: "manager"}}},
		MaxLevels: 3, FallbackProduct: "sales", FallbackRole: "manager",
	}))
}

func (f *fixture) posted(docType string) int {
	f.t.Helper()
	list := f.sales.Quotes
	if docType == "sales.order" {
		list = f.sales.Orders
	}
	l, err := list(f.staff, sales.DocFilter{Status: "posted", Sort: "-date", Page: 1, PageSize: 100})
	f.check(err)
	return int(l.Total)
}

func contract(t *testing.T, docType string) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		f := newFixture(t)
		create, update := f.sales.CreateQuote, f.sales.UpdateQuote
		if docType == "sales.order" {
			create, update = f.sales.CreateOrder, f.sales.UpdateOrder
		}
		fields := func(date string) sales.DocFields {
			fl := f.fields(date)
			fl.Lines[2].DiscountPercent = "15"
			if docType == "sales.order" {
				fl.ValidUntil = nil
			}
			return fl
		}
		return recordtest.Harness{
			Record: f.rec, Actor: f.staff, Admin: f.admin, LegalEntity: f.c,
			Create: func(ctx context.Context, date string) (record.Ref, error) {
				id, err := create(ctx, sales.NewDoc{RequestID: newRequest(), DocFields: fields(date)})
				return record.Ref{Type: docType, ID: id}, err
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return update(ctx, r.ID, sales.DocUpdate{Version: version, DocFields: fields(date)})
			},
			RequireApproval: func(*testing.T) { f.managerApproves(docType) },
			Approve:         func(instance int64) error { return f.appr.Approve(f.manager, instance, 1) },
			// Posting refuses a customer deactivated since the draft was saved.
			BreakPosting: func(*testing.T, record.Ref) {
				c := customer("KH1", f.a)
				c.Active = false
				f.check(f.sales.UpdateCustomer(f.staff, f.customer, c))
			},
			Effects: func(*testing.T, record.Ref) int { return f.posted(docType) },
		}
	})
}

func TestQuoteRecordContract(t *testing.T) { contract(t, "sales.quote") }
func TestOrderRecordContract(t *testing.T) { contract(t, "sales.order") }

// A customer belongs to a team: other teams neither list nor open it, even by id.
func TestCustomerScope(t *testing.T) {
	f := newFixture(t)
	staffB := f.ctxFor("staff_b", "staff@B")
	viewer := f.ctxFor("viewer", "viewer@*")
	nobody := f.ctxFor("nobody")

	if _, err := f.sales.CreateCustomer(f.staff, customer("KHB", f.b)); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("create in B as staff of A: %v", err)
	}
	wantCode(t, must2(f.sales.CreateCustomer(staffB, customer("kh1", f.b))), "customer_code_taken")
	if l, err := f.sales.Customers(staffB, sales.CustomerFilter{Sort: "code", Page: 1, PageSize: 50}); err != nil || l.Total != 0 {
		t.Fatalf("B lists %+v %v", l, err)
	}
	if _, err := f.sales.Customer(staffB, f.customer); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("B opens A's customer: %v", err)
	}
	if err := f.sales.UpdateCustomer(staffB, f.customer, customer("KH1", f.b)); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("B edits A's customer: %v", err)
	}
	if l, _ := f.sales.Customers(nobody, sales.CustomerFilter{Sort: "code", Page: 1, PageSize: 50}); l.Total != 0 {
		t.Fatalf("nobody lists %+v", l)
	}
	c, err := f.sales.Customer(viewer, f.customer)
	if err != nil || len(c.AllowedActions) != 0 || c.Name != "Công ty KH1" {
		t.Fatalf("viewer: %+v %v", c, err)
	}
	if err := f.sales.UpdateCustomer(viewer, f.customer, customer("KH1", f.a)); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("viewer edits: %v", err)
	}
	// Moving a customer needs edit at both ends.
	if err := f.sales.UpdateCustomer(f.staff, f.customer, customer("KH1", f.b)); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("move to B: %v", err)
	}
	if c, _ := f.sales.Customer(f.staff, f.customer); !slices.Equal(c.AllowedActions, []string{"edit", "attach"}) {
		t.Fatalf("staff actions %v", c.AllowedActions)
	}
	// An org unit that does not exist is not found, for a tenant-wide editor too.
	everywhere := f.ctxFor("staff_all", "staff@*")
	if _, err := f.sales.CreateCustomer(everywhere, customer("KH9", 999999)); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("create at a missing unit: %v", err)
	}
	if err := f.sales.UpdateCustomer(everywhere, f.customer, customer("KH1", 999999)); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("move to a missing unit: %v", err)
	}

	// With Sales off the customer reads, and nothing writes.
	off := platform.WithProducts(f.staff, nil)
	if _, err := f.sales.Customer(off, f.customer); err != nil {
		t.Fatal(err)
	}
	wantCode(t, f.sales.UpdateCustomer(off, f.customer, customer("KH1", f.a)), "product_not_enabled")
	wantCode(t, must2(f.sales.CreateCustomer(off, customer("KH2", f.a))), "product_not_enabled")
	if a, _ := f.sales.CustomerActions(off); len(a.AllowedActions) != 0 {
		t.Fatalf("actions with Sales off: %v", a)
	}
}

func must2(_ int64, err error) error { return err }

// The catalogue is tenant-wide: read by any sales user, written only by catalogue admins.
func TestItems(t *testing.T) {
	f := newFixture(t)
	if l, err := f.sales.Items(f.ctxFor("viewer_b", "viewer@B"), sales.ItemFilter{Q: "bút", Page: 1, PageSize: 50}); err != nil || l.Total != 1 {
		t.Fatalf("viewer of B: %+v %v", l, err)
	}
	if _, err := f.sales.Items(f.ctxFor("nobody"), sales.ItemFilter{Page: 1, PageSize: 50}); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("nobody: %v", err)
	}
	if _, err := f.sales.SaveItem(f.manager, 0, sales.ItemFields{Code: "X", Name: "X", Unit: "cái", VatRate: "0", Active: true}); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("manager adds an item: %v", err)
	}
	catalog := f.ctxFor("catalog2", "catalog_admin@*")
	wantCode(t, must2(f.sales.SaveItem(catalog, 0, sales.ItemFields{Code: "but", Name: "Bút", Unit: "cái", VatRate: "8", Active: true})), "item_code_taken")
}

// A quotation computes on the server, keeps what it said once posted, and creating it twice
// with one request id makes one.
func TestQuote(t *testing.T) {
	f := newFixture(t)
	in := sales.NewDoc{RequestID: newRequest(), DocFields: f.fields("2026-03-10")}
	id := must(t)(f.sales.CreateQuote(f.staff, in))
	if again := must(t)(f.sales.CreateQuote(f.staff, in)); again != id {
		t.Fatalf("same request: %d, then %d", id, again)
	}
	wantCode(t, must2(f.sales.CreateOrder(f.staff, in)), "request_id_reused")

	q, err := f.sales.Quote(f.staff, id)
	f.check(err)
	// Pens: 10,005 + 10,005 at 8 % (800 + 800 by line). Repair: 375,000 less 37,500, VAT 33,750.
	want := sales.Totals{Subtotal: 395_010, DiscountTotal: 37_500, VatTotal: 35_350, Total: 392_860}
	if q.Totals != want || q.Number != "BG-2026-00001" || q.Customer.Name != "Công ty KH1" || q.Lines[2].Unit != "giờ" || q.Lines[2].Quantity != "1.5" {
		t.Fatalf("quote %+v", q)
	}
	d, err := f.rec.Get(f.admin, record.Ref{Type: "sales.quote", ID: id})
	f.check(err)
	if *d.Amount != want.Total || d.Fields["max_discount"] != "10" {
		t.Fatalf("header %+v %v", d, d.Fields)
	}

	// total rounding: the 8 % VAT rounds once over both pens, 1,600.8 → 1,601.
	f.check(f.set.SetFor(f.admin, f.c, setting.Rounding, "total"))
	f.check(f.sales.UpdateQuote(f.staff, id, sales.DocUpdate{Version: q.Version, DocFields: f.fields("2026-03-10")}))
	q, _ = f.sales.Quote(f.staff, id)
	if q.Lines[0].Vat != 801 || q.Lines[1].Vat != 800 || q.VatTotal != 35_351 {
		t.Fatalf("total rounding: %+v", q)
	}

	// Validity before the date, an inactive item: refused.
	bad := f.fields("2026-03-10")
	bad.ValidUntil = str("2026-03-09")
	wantCode(t, f.sales.UpdateQuote(f.staff, id, sales.DocUpdate{Version: q.Version, DocFields: bad}), "quote_valid_until_before_date")

	// A 10 % discount needs no approval under "above 10 %": posted at once.
	f.managerApproves("sales.quote")
	f.check(f.move(f.staff, "sales.quote", id, record.Posted))
	// Editing the customer after posting changes nothing on the quotation.
	c := customer("KH1", f.a)
	c.Name = "Công ty đổi tên"
	f.check(f.sales.UpdateCustomer(f.staff, f.customer, c))
	if q, _ := f.sales.Quote(f.staff, id); q.Customer.Name != "Công ty KH1" || q.Status != "posted" || !slices.Contains(q.AllowedActions, "create_order") {
		t.Fatalf("posted quote: %+v", q)
	}

	// 15 % waits for the manager, who sees it in the inbox; the staff cannot approve.
	f2 := f.fields("2026-03-11")
	f2.Lines[2].DiscountPercent = "15"
	id2 := must(t)(f.sales.CreateQuote(f.staff, sales.NewDoc{RequestID: newRequest(), DocFields: f2}))
	f.check(f.move(f.staff, "sales.quote", id2, record.Posted))
	inbox, err := f.appr.Inbox(f.manager)
	f.check(err)
	if len(inbox) != 1 || inbox[0].DocID != id2 {
		t.Fatalf("manager's inbox: %+v", inbox)
	}
	if err := f.appr.Approve(f.staff, inbox[0].InstanceID, 1); err == nil {
		t.Fatal("the submitter approved")
	}
	f.check(f.appr.Approve(f.manager, inbox[0].InstanceID, 1))
	if q, _ := f.sales.Quote(f.staff, id2); q.Status != "posted" || q.Customer.Name != "Công ty đổi tên" {
		t.Fatalf("approved quote: %+v", q)
	}

	// Team B sees none of it.
	staffB := f.ctxFor("staff_b", "staff@B")
	if _, err := f.sales.Quote(staffB, id); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("B opens: %v", err)
	}
	if _, err := f.sales.CreateOrderFromQuote(staffB, id); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("B orders: %v", err)
	}
	if l, _ := f.sales.Quotes(staffB, sales.DocFilter{Sort: "-date", Page: 1, PageSize: 50}); l.Total != 0 {
		t.Fatalf("B lists %+v", l)
	}
	// B cannot quote A's customer, even in B.
	fb := f.fields("2026-03-10")
	fb.OrgUnitID = f.b
	if _, err := f.sales.CreateQuote(staffB, sales.NewDoc{RequestID: newRequest(), DocFields: fb}); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("B quotes A's customer: %v", err)
	}
}

// Concurrent creates with one request id leave one quotation.
func TestQuoteCreateRace(t *testing.T) {
	f := newFixture(t)
	in := sales.NewDoc{RequestID: newRequest(), DocFields: f.fields("2026-03-10")}
	ids := make([]int64, 4)
	var wg sync.WaitGroup
	for i := range ids {
		wg.Go(func() {
			id, err := f.sales.CreateQuote(f.staff, in)
			if err != nil {
				t.Error(err)
			}
			ids[i] = id
		})
	}
	wg.Wait()
	if ids[0] == 0 || slices.ContainsFunc(ids, func(v int64) bool { return v != ids[0] }) {
		t.Fatalf("ids %v", ids)
	}
	if l, _ := f.sales.Quotes(f.staff, sales.DocFilter{Sort: "-date", Page: 1, PageSize: 50}); l.Total != 1 {
		t.Fatalf("%d quotations", l.Total)
	}
}

// A posted quotation becomes one draft order, however many times it is asked; the order
// frees it once cancelled.
func TestOrderFromQuote(t *testing.T) {
	f := newFixture(t)
	today := todayOf(t, f)
	qf := f.fields(today)
	quote := must(t)(f.sales.CreateQuote(f.staff, sales.NewDoc{RequestID: newRequest(), DocFields: qf}))
	wantCode(t, must2(f.sales.CreateOrderFromQuote(f.staff, quote)), "quote_not_posted")
	f.check(f.move(f.staff, "sales.quote", quote, record.Posted))

	// A viewer may read but not order.
	if _, err := f.sales.CreateOrderFromQuote(f.ctxFor("viewer", "viewer@A"), quote); !errors.Is(err, platform.ErrForbidden) {
		t.Fatalf("viewer orders: %v", err)
	}
	order := must(t)(f.sales.CreateOrderFromQuote(f.staff, quote))
	if again := must(t)(f.sales.CreateOrderFromQuote(f.staff, quote)); again != order {
		t.Fatalf("second order %d, first %d", again, order)
	}
	o, err := f.sales.Order(f.staff, order)
	f.check(err)
	q, _ := f.sales.Quote(f.staff, quote)
	if o.Status != "draft" || o.Quote == nil || o.Quote.ID != quote || o.Totals != q.Totals || len(o.Lines) != 3 || o.Lines[2].DiscountPercent != "10" ||
		o.Date != today || q.Order == nil || q.Order.ID != order || slices.Contains(q.AllowedActions, "create_order") {
		t.Fatalf("order %+v from quote %+v", o, q)
	}
	wantCode(t, f.move(f.staff, "sales.quote", quote, record.Cancelled), "quote_has_order")

	// Posted, then cancelled: the quotation may become an order again.
	f.check(f.move(f.staff, "sales.order", order, record.Posted))
	f.check(f.move(f.staff, "sales.order", order, record.Cancelled))
	second := must(t)(f.sales.CreateOrderFromQuote(f.staff, quote))
	if second == order {
		t.Fatal("cancelled order returned")
	}
	// Deleting the draft frees it too; then cancelling the quotation goes through.
	o2, _ := f.sales.Order(f.staff, second)
	f.check(f.sales.DeleteOrder(f.staff, second, o2.Version))
	f.check(f.move(f.staff, "sales.quote", quote, record.Cancelled))

	// An expired quotation stays readable but becomes no order.
	old := f.fields("2026-01-05")
	old.ValidUntil = str("2026-01-31")
	expired := must(t)(f.sales.CreateQuote(f.staff, sales.NewDoc{RequestID: newRequest(), DocFields: old}))
	f.check(f.move(f.staff, "sales.quote", expired, record.Posted))
	if q, _ := f.sales.Quote(f.staff, expired); !q.Expired || slices.Contains(q.AllowedActions, "create_order") {
		t.Fatalf("expired quote %+v", q)
	}
	wantCode(t, must2(f.sales.CreateOrderFromQuote(f.staff, expired)), "quote_expired")

	// An order entered directly has no quotation.
	of := f.fields(today)
	of.ValidUntil = nil
	direct := must(t)(f.sales.CreateOrder(f.staff, sales.NewDoc{RequestID: newRequest(), DocFields: of}))
	if o, _ := f.sales.Order(f.staff, direct); o.Quote != nil || o.Number != "DH-"+today[:4]+"-00003" {
		t.Fatalf("direct order %+v", o)
	}
}

// Concurrent conversions of one quotation make one order.
func TestOrderFromQuoteRace(t *testing.T) {
	f := newFixture(t)
	quote := must(t)(f.sales.CreateQuote(f.staff, sales.NewDoc{RequestID: newRequest(), DocFields: f.fields(todayOf(t, f))}))
	f.check(f.move(f.staff, "sales.quote", quote, record.Posted))
	ids := make([]int64, 4)
	var wg sync.WaitGroup
	for i := range ids {
		wg.Go(func() {
			id, err := f.sales.CreateOrderFromQuote(f.staff, quote)
			if err != nil {
				t.Error(err)
			}
			ids[i] = id
		})
	}
	wg.Wait()
	if ids[0] == 0 || slices.ContainsFunc(ids, func(v int64) bool { return v != ids[0] }) {
		t.Fatalf("ids %v", ids)
	}
}

func todayOf(t *testing.T, f *fixture) string {
	t.Helper()
	var d string
	if err := platform.DBFrom(f.ctx).QueryRow(f.ctx, `SELECT (now() AT TIME ZONE 'Asia/Ho_Chi_Minh')::date::text`).Scan(&d); err != nil {
		t.Fatal(err)
	}
	return d
}

// Sending a draft takes the customer as it is then; edits while it waits for approval, or
// once posted, never reach it. An inactive customer keeps it a draft.
func TestCustomerCopiedAtSending(t *testing.T) {
	f := newFixture(t)
	f.managerApproves("sales.quote")
	rename := func(name string, active bool) {
		c := customer("KH1", f.a)
		c.Name, c.Active = name, active
		f.check(f.sales.UpdateCustomer(f.staff, f.customer, c))
	}
	fl := f.fields("2026-03-10")
	fl.Lines[2].DiscountPercent = "15"
	id := must(t)(f.sales.CreateQuote(f.staff, sales.NewDoc{RequestID: newRequest(), DocFields: fl}))

	rename("Công ty lúc gửi", false)
	wantCode(t, f.move(f.staff, "sales.quote", id, record.Posted), "customer_inactive")
	if q, _ := f.sales.Quote(f.staff, id); q.Status != "draft" || q.Customer.Name != "Công ty KH1" {
		t.Fatalf("refused send: %+v", q)
	}
	rename("Công ty lúc gửi", true)
	f.check(f.move(f.staff, "sales.quote", id, record.Posted))
	rename("Công ty lúc chờ duyệt", true)
	q, _ := f.sales.Quote(f.staff, id)
	if q.Status != "pending_approval" || q.Customer.Name != "Công ty lúc gửi" {
		t.Fatalf("pending: %+v", q)
	}
	inbox, err := f.appr.Inbox(f.manager)
	f.check(err)
	f.check(f.appr.Approve(f.manager, inbox[0].InstanceID, 1))
	if q, _ := f.sales.Quote(f.staff, id); q.Status != "posted" || q.Customer.Name != "Công ty lúc gửi" {
		t.Fatalf("posted: %+v", q)
	}
}
