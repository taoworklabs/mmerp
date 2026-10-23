package app_test

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/app"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
	"github.com/taoworklabs/mmerp/migrations"
)

// salesFixture: company C with teams A and B; staff of A, staff of B, a catalogue admin;
// a customer of A, an item, and a draft quotation of A.
type salesFixture struct {
	*jobsFixture
	staffA, staffB  *client
	company, a      int64
	customer, quote int64
}

func newSalesFixture(t *testing.T, products []string) *salesFixture {
	pool := pgtest.New(t)
	if err := app.CreateAdmin(t.Context(), pool, "admin", "Admin", "correct horse"); err != nil {
		t.Fatal(err)
	}
	e := env(t, pool, products)
	f := &salesFixture{jobsFixture: &jobsFixture{t: t, system: e.SystemPool, env: e}}
	f.admin = f.login("admin")
	f.company = f.id(f.admin, "/api/org-units", `{"parent_id":null,"kind":"company","name":"C","tax_code":"0100000001","legal_name":"Công ty TNHH C","address":"1 Tràng Tiền"}`)
	f.a = f.id(f.admin, "/api/org-units", fmt.Sprintf(`{"parent_id":%d,"kind":"department","name":"Kinh doanh A"}`, f.company))
	b := f.id(f.admin, "/api/org-units", fmt.Sprintf(`{"parent_id":%d,"kind":"department","name":"Kinh doanh B"}`, f.company))
	user := func(login string, unit any, roles ...string) *client {
		id := f.id(f.admin, "/api/users", fmt.Sprintf(`{"login":%q,"name":%q,"password":"correct horse"}`, login, login))
		for _, r := range roles {
			product, role, _ := strings.Cut(r, ".")
			f.id(f.admin, fmt.Sprintf("/api/users/%d/roles", id), fmt.Sprintf(`{"product":%q,"role":%q,"org_unit_id":%v}`, product, role, unit))
		}
		return f.login(login)
	}
	f.staffA, f.staffB = user("staff_a", f.a, "sales.staff", "partner.editor"), user("staff_b", b, "sales.staff", "partner.editor")
	catalog := user("catalog", "null", "item.admin")
	return f.withData(catalog)
}

func (f *salesFixture) withData(catalog *client) *salesFixture {
	item := f.id(catalog, "/api/items", `{"code":"BUT","name":"Bút bi","unit":"hộp","price":10005,"vat_rate":"8","active":true}`)
	f.customer = f.id(f.staffA, "/api/sales/customers", fmt.Sprintf(`{"code":"KH1","partner":{"name":"Công ty Ánh Dương","tax_code":"0312345678",
		"address":"5 Nguyễn Huệ","phone":null,"email":null,"contact_name":"Chị Hương"},"payment_terms":"30 ngày","org_unit_id":%d,"active":true}`, f.a))
	f.quote = f.id(f.staffA, "/api/sales/quotes", fmt.Sprintf(`{"request_id":"7d7c1a52-6c8e-4f7a-9d0e-1f2a3b4c5d6e","date":"2026-03-10",
		"org_unit_id":%d,"customer_id":%d,"valid_until":"2026-12-31","delivery_date":null,"payment_terms":"30 ngày","delivery_terms":null,"note":null,
		"lines":[{"item_id":%d,"description":"Bút bi xanh","quantity":"12","unit_price":10005,"discount_percent":"5","vat_rate":"8"}]}`, f.a, f.customer, item))
	return f
}

func (f *salesFixture) moveQuote(to string) {
	f.t.Helper()
	var d struct{ Version int32 }
	_ = json.Unmarshal(f.ok(f.staffA, "GET", fmt.Sprintf("/api/sales/quotes/%d", f.quote), "", 200), &d)
	f.ok(f.staffA, "POST", fmt.Sprintf("/api/documents/sales.quote/%d/transitions", f.quote), fmt.Sprintf(`{"to":%q,"version":%d}`, to, d.Version), 200)
}

// A posted quotation prints the same after its customer and the seller are edited; team B
// can neither print nor open it.
func TestPrintQuote(t *testing.T) {
	f := newSalesFixture(t, []string{"sales"})
	f.work([]string{"sales"})
	params := fmt.Sprintf(`{"id":%d}`, f.quote)
	draft := f.printed(f.staffA, "sales.quote", params)
	// 12 × 10,005 = 120,060, less 5 % (6,003) = 114,057, VAT 9,124.56 → 9,125: 123,182.
	for _, want := range []string{"BÁO GIÁ", "BG-2026-00001", "Công ty Ánh Dương", "Bút bi xanh", "114.057", "9.125", "123.182", "Công ty TNHH C", "BẢN NHÁP"} {
		if !strings.Contains(draft[0], want) {
			t.Errorf("no %q in %q", want, draft[0])
		}
	}
	if j := f.print(f.staffB, "sales.quote", params); j.Code == nil || *j.Code != "not_found" {
		t.Fatalf("B prints: %+v", j)
	}
	f.ok(f.staffB, "GET", fmt.Sprintf("/api/sales/quotes/%d", f.quote), "", 404)

	f.moveQuote("posted")
	first := f.printed(f.staffA, "sales.quote", params)
	f.ok(f.staffA, "PUT", fmt.Sprintf("/api/sales/customers/%d", f.customer), fmt.Sprintf(`{"partner":{"name":"Công ty Mới","tax_code":null,
		"address":null,"phone":null,"email":null,"contact_name":null},"payment_terms":null,"org_unit_id":%d,"active":true}`, f.a), 204)
	f.ok(f.admin, "PUT", fmt.Sprintf("/api/org-units/%d", f.company), `{"parent_id":null,"kind":"company","name":"C","legal_name":"Công ty Cổ phần C"}`, 204)
	if again := f.printed(f.staffA, "sales.quote", params); !slices.Equal(again, first) || strings.Contains(first[0], "BẢN NHÁP") {
		t.Fatalf("reprint differs:\n%q\n%q", again, first)
	}
}

// With Sales off, quotations and customers read and print; nothing writes, and no write
// action is offered.
func TestSalesProductOff(t *testing.T) {
	f := newSalesFixture(t, []string{"sales"})
	off := newSalesFixtureOn(f, nil)
	f.work(nil)
	var q struct {
		AllowedActions []string `json:"allowed_actions"`
	}
	_ = json.Unmarshal(f.ok(off, "GET", fmt.Sprintf("/api/sales/quotes/%d", f.quote), "", 200), &q)
	if !slices.Equal(q.AllowedActions, []string{"print"}) {
		t.Fatalf("actions with Sales off: %v", q.AllowedActions)
	}
	f.ok(off, "GET", fmt.Sprintf("/api/sales/customers/%d", f.customer), "", 200)
	customer := fmt.Sprintf(`{"code":"KH9","partner":{"name":"Khách 9","tax_code":null,"address":null,"phone":null,"email":null,
		"contact_name":null},"payment_terms":null,"org_unit_id":%d,"active":true}`, f.a)
	wantCode(t, off.do("POST", "/api/sales/customers", customer), 403, "product_not_enabled")
	wantCode(t, off.do("PUT", fmt.Sprintf("/api/sales/customers/%d", f.customer), strings.Replace(customer, `"code":"KH9",`, "", 1)), 403, "product_not_enabled")
	wantCode(t, off.do("POST", fmt.Sprintf("/api/sales/quotes/%d/order", f.quote), ""), 403, "product_not_enabled")
	wantCode(t, off.do("POST", fmt.Sprintf("/api/documents/sales.quote/%d/transitions", f.quote), `{"to":"posted","version":1}`), 403, "product_not_enabled")
	if pages := f.printed(off, "sales.quote", fmt.Sprintf(`{"id":%d}`, f.quote)); !strings.Contains(pages[0], "Công ty Ánh Dương") {
		t.Fatalf("print with Sales off: %q", pages[0])
	}
}

// Customers alone are Sales data: with Sales off, its area still shows. Items belong
// to no product, so they alone are nobody's data.
func TestSalesCatalogueIsData(t *testing.T) {
	pool := pgtest.New(t)
	if err := app.CreateAdmin(t.Context(), pool, "admin", "Admin", "correct horse"); err != nil {
		t.Fatal(err)
	}
	e := env(t, pool, []string{"sales"})
	f := &jobsFixture{t: t, system: e.SystemPool, env: e}
	admin := f.login("admin")
	// Without a code, the item gets one issued.
	f.id(admin, "/api/items", `{"name":"Bút bi","unit":"hộp","price":10005,"vat_rate":"8","active":true}`)
	unit := f.id(admin, "/api/org-units", `{"parent_id":null,"kind":"company","name":"C"}`)
	withData := func() []string {
		f.env = env(t, pool, nil)
		var me struct {
			ProductsWithData []string `json:"products_with_data"`
		}
		_ = json.Unmarshal(f.ok(f.login("admin"), "GET", "/api/me", "", 200), &me)
		return me.ProductsWithData
	}
	if got := withData(); len(got) != 0 {
		t.Fatalf("products with data, items only: %v", got)
	}
	f.env = env(t, pool, []string{"sales"})
	f.id(f.login("admin"), "/api/sales/customers", fmt.Sprintf(`{"partner":{"name":"Khách","tax_code":null,"address":null,"phone":null,"email":null,
		"contact_name":null},"payment_terms":null,"org_unit_id":%d,"active":true}`, unit))
	if got := withData(); !slices.Equal(got, []string{"sales"}) {
		t.Fatalf("products with data: %v", got)
	}
}

// newSalesFixtureOn signs staff of A in again on a server running with products.
func newSalesFixtureOn(f *salesFixture, products []string) *client {
	f.env = env(f.t, f.env.Pool, products)
	return f.login("staff_a")
}

// The tenant administrator works in every product with no role of its own, but sees no
// salary without one.
func TestAdminHoldsBusinessPermissions(t *testing.T) {
	f := newSalesFixture(t, []string{"hrm", "sales"})
	var me struct{ Permissions map[string][]string }
	_ = json.Unmarshal(f.ok(f.admin, "GET", "/api/me", "", 200), &me)
	if !slices.Contains(me.Permissions["sales"], "sales.quote.edit") || !slices.Contains(me.Permissions["hrm"], "hrm.employee.edit") ||
		slices.Contains(me.Permissions["hrm"], "hrm.salary.view") || slices.Contains(me.Permissions["hrm"], "hrm.employee.sensitive") {
		t.Fatalf("admin permissions: %v", me.Permissions)
	}
	f.ok(f.admin, "GET", fmt.Sprintf("/api/sales/quotes/%d", f.quote), "", 200)
	f.id(f.admin, "/api/sales/customers", fmt.Sprintf(`{"code":"KH2","partner":{"name":"Khách 2","tax_code":null,"address":null,"phone":null,"email":null,
		"contact_name":null},"payment_terms":null,"org_unit_id":%d,"active":true}`, f.a))
}

// The partner migration, run on customers made before it: one tax code (trimmed) is one
// partner, named after its newest customer; a later customer of that tax code in the same
// org unit gets a partner of its own; Sales roles get the matching partner roles.
func TestPartnerMigration(t *testing.T) {
	const partnerInit = "20261023090622_partner_init.sql"
	pool := pgtest.Empty(t)
	ctx := t.Context()
	before := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() >= partnerInit || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		b, err := fs.ReadFile(migrations.FS, e.Name())
		if err != nil {
			t.Fatal(err)
		}
		before[e.Name()] = &fstest.MapFile{Data: b}
	}
	if err := platform.Migrate(ctx, pool, before); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO iam.org_units (id, kind, name) OVERRIDING SYSTEM VALUE VALUES (1, 'company', 'U1'), (2, 'company', 'U2');
		INSERT INTO iam.users (id, login, name, password_hash) OVERRIDING SYSTEM VALUE VALUES (1, 'staff', 'staff', 'x'), (2, 'viewer', 'viewer', 'x');
		INSERT INTO iam.user_roles (user_id, product, role, org_unit_id) VALUES (1, 'sales', 'staff', 1), (2, 'sales', 'viewer', NULL);
		INSERT INTO sales.customers (id, code, name, tax_code, org_unit_id) OVERRIDING SYSTEM VALUE VALUES
			(1, 'KHA', 'A', '0101 ', 1), (2, 'KHB', 'B', '0101', 2), (3, 'KHC', 'C', NULL, 1), (4, 'KHD', 'D', '0101', 1);
		UPDATE sales.customers SET phone = '', address = ' x ' WHERE id = 3`); err != nil {
		t.Fatal(err)
	}
	if err := platform.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatal(err)
	}
	rows, _ := pool.Query(ctx, `SELECT c.code || ':' || p.code || ':' || p.name || ':' || coalesce(p.tax_code, '-')
		FROM sales.customers c JOIN partner.partners p ON p.id = c.partner_id ORDER BY c.id`)
	got, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if want := []string{"KHA:DT00001:B:0101", "KHB:DT00001:B:0101", "KHC:DT00002:C:-", "KHD:DT00003:D:-"}; err != nil || !slices.Equal(got, want) {
		t.Fatalf("customers %v %v, want %v", got, err, want)
	}
	var phone, address *string
	if err := pool.QueryRow(ctx, `SELECT p.phone, p.address FROM partner.partners p JOIN sales.customers c ON c.partner_id = p.id WHERE c.id = 3`).Scan(&phone, &address); err != nil ||
		phone != nil || address == nil || *address != "x" {
		t.Fatalf("C's identity not cleaned: phone %v, address %v, %v", phone, address, err)
	}
	rows, _ = pool.Query(ctx, `SELECT user_id || ':' || role || ':' || coalesce(org_unit_id::text, '*') FROM iam.user_roles WHERE product = 'partner' ORDER BY user_id`)
	roles, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if want := []string{"1:editor:1", "2:viewer:*"}; err != nil || !slices.Equal(roles, want) {
		t.Fatalf("roles %v %v", roles, err)
	}
	var last int64
	if err := pool.QueryRow(ctx, `SELECT last FROM numbering.series WHERE code = 'partner.partner' AND prefix = 'DT'`).Scan(&last); err != nil || last != 3 {
		t.Fatalf("series %d %v", last, err)
	}
}
