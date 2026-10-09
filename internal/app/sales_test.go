package app_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/taoworklabs/mmerp/internal/app"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
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
	f := &salesFixture{jobsFixture: &jobsFixture{t: t, pool: pool, env: env(t, pool, products)}}
	f.admin = f.login("admin")
	f.company = f.id(f.admin, "/api/org-units", `{"parent_id":null,"kind":"company","name":"C","tax_code":"0100000001","legal_name":"Công ty TNHH C","address":"1 Tràng Tiền"}`)
	f.a = f.id(f.admin, "/api/org-units", fmt.Sprintf(`{"parent_id":%d,"kind":"department","name":"Kinh doanh A"}`, f.company))
	b := f.id(f.admin, "/api/org-units", fmt.Sprintf(`{"parent_id":%d,"kind":"department","name":"Kinh doanh B"}`, f.company))
	user := func(login, role string, unit any) *client {
		id := f.id(f.admin, "/api/users", fmt.Sprintf(`{"login":%q,"name":%q,"password":"correct horse"}`, login, login))
		f.id(f.admin, fmt.Sprintf("/api/users/%d/roles", id), fmt.Sprintf(`{"product":"sales","role":%q,"org_unit_id":%v}`, role, unit))
		return f.login(login)
	}
	f.staffA, f.staffB = user("staff_a", "staff", f.a), user("staff_b", "staff", b)
	catalog := user("catalog", "catalog_admin", "null")
	return f.withData(catalog)
}

func (f *salesFixture) withData(catalog *client) *salesFixture {
	item := f.id(catalog, "/api/sales/items", `{"code":"BUT","name":"Bút bi","unit":"hộp","price":10005,"vat_rate":"8","active":true}`)
	f.customer = f.id(f.staffA, "/api/sales/customers", fmt.Sprintf(`{"code":"KH1","name":"Công ty Ánh Dương","tax_code":"0312345678",
		"address":"5 Nguyễn Huệ","phone":null,"email":null,"contact_name":"Chị Hương","payment_terms":"30 ngày","org_unit_id":%d,"active":true}`, f.a))
	f.quote = f.id(f.staffA, "/api/sales/quotes", fmt.Sprintf(`{"request_id":"7d7c1a52-6c8e-4f7a-9d0e-1f2a3b4c5d6e","date":"2026-03-10",
		"org_unit_id":%d,"customer_id":%d,"valid_until":"2026-12-31","delivery_date":null,"payment_terms":"30 ngày","delivery_terms":null,"note":null,
		"lines":[{"item_id":%d,"description":"Bút bi xanh","quantity":"12","unit_price":10005,"discount_percent":"5","vat_rate":"8"}]}`, f.a, f.customer, item))
	return f
}

func (f *salesFixture) moveQuote(to string) {
	f.t.Helper()
	var d struct{ Version int32 }
	_ = json.Unmarshal(f.ok(f.staffA, "GET", fmt.Sprintf("/api/sales/quotes/%d", f.quote), "", 200), &d)
	f.ok(f.staffA, "POST", fmt.Sprintf("/api/documents/sales.quote/%d/transitions", f.quote), fmt.Sprintf(`{"to":%q,"version":%d}`, to, d.Version), 204)
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
	f.ok(f.staffA, "PUT", fmt.Sprintf("/api/sales/customers/%d", f.customer), fmt.Sprintf(`{"code":"KH1","name":"Công ty Mới","tax_code":null,
		"address":null,"phone":null,"email":null,"contact_name":null,"payment_terms":null,"org_unit_id":%d,"active":true}`, f.a), 204)
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
	wantCode(t, off.do("POST", fmt.Sprintf("/api/sales/quotes/%d/order", f.quote), ""), 403, "product_not_enabled")
	wantCode(t, off.do("POST", fmt.Sprintf("/api/documents/sales.quote/%d/transitions", f.quote), `{"to":"posted","version":1}`), 403, "product_not_enabled")
	if pages := f.printed(off, "sales.quote", fmt.Sprintf(`{"id":%d}`, f.quote)); !strings.Contains(pages[0], "Công ty Ánh Dương") {
		t.Fatalf("print with Sales off: %q", pages[0])
	}
}

// newSalesFixtureOn signs staff of A in again on a server running with products.
func newSalesFixtureOn(f *salesFixture, products []string) *client {
	f.env = env(f.t, f.pool, products)
	return f.login("staff_a")
}
