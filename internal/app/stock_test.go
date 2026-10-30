package app_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/taoworklabs/mmerp/internal/app"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// An adjustment line of an untracked-lot item is sent without a lot and read back without one.
func TestStockAdjustmentAPI(t *testing.T) {
	pool := pgtest.New(t)
	if err := app.CreateAdmin(t.Context(), pool, "admin", "Admin", "correct horse"); err != nil {
		t.Fatal(err)
	}
	f := &jobsFixture{t: t, env: env(t, pool, nil)}
	f.system = f.env.SystemPool
	f.admin = f.login("admin")
	company := f.id(f.admin, "/api/org-units", `{"parent_id":null,"kind":"company","name":"C"}`)
	keeper := f.id(f.admin, "/api/users", `{"login":"keeper","name":"keeper","password":"correct horse"}`)
	for _, r := range []string{"item", "stock"} {
		f.id(f.admin, fmt.Sprintf("/api/users/%d/roles", keeper), fmt.Sprintf(`{"product":%q,"role":"admin","org_unit_id":null}`, r))
	}
	c := f.login("keeper")
	item := f.id(c, "/api/items", `{"code":"BUT","name":"Bút bi","unit":"hộp","price":1,"vat_rate":"8","active":true,"tracking":"quantity"}`)
	wh := f.id(c, "/api/stock/warehouses", fmt.Sprintf(`{"name":"Kho 1","legal_entity_id":%d,"active":true}`, company))
	n := 0
	body := func(line, reason string) string {
		n++
		return fmt.Sprintf(`{"request_id":"00000000-0000-4000-8000-%012d","date":"2026-11-02","warehouse_id":%d,"reason":%q,"lines":[%s]}`, n, wh, reason, line)
	}
	id := f.id(c, "/api/stock/adjustments", body(fmt.Sprintf(`{"item_id":%d,"quantity":"5"}`, item), "Kiểm kê"))
	f.id(c, "/api/stock/adjustments", body(fmt.Sprintf(`{"item_id":%d,"lot":null,"serial":null,"quantity":"5"}`, item), "Kiểm kê"))
	f.ok(c, "GET", fmt.Sprintf("/api/stock/adjustments/%d", id), "", 200)
	wantCode(t, c.do("POST", "/api/stock/adjustments", body(fmt.Sprintf(`{"item_id":%d,"quantity":"5"}`, item), "   ")), 422, "stock_reason_required")
	wantCode(t, c.do("POST", "/api/stock/adjustments", body(fmt.Sprintf(`{"item_id":%d,"lot":{"code":"  "},"quantity":"5"}`, item), "x")), 422, "stock_code_blank")
	wantCode(t, c.do("POST", "/api/stock/adjustments", body(fmt.Sprintf(`{"item_id":%d,"serial":" ","quantity":"1"}`, item), "x")), 422, "stock_code_blank")
}

// The stock card names a document only to readers who may view it: a stock viewer
// sees the adjustment's number but not the Sales delivery's.
func TestStockCardHidesUnreadableDocuments(t *testing.T) {
	f := newSalesFixture(t, []string{"sales", "stock"})
	// The fixture's item becomes stock-tracked so it can be delivered.
	var items struct{ Items []struct{ ID int64 } }
	if err := json.Unmarshal(f.ok(f.admin, "GET", "/api/items", "", 200), &items); err != nil {
		t.Fatal(err)
	}
	item := items.Items[0].ID
	f.ok(f.admin, "PUT", fmt.Sprintf("/api/items/%d", item), `{"name":"Bút bi","unit":"hộp","price":10005,"vat_rate":"8","active":true,"tracking":"quantity"}`, 204)
	wh := f.id(f.admin, "/api/stock/warehouses", fmt.Sprintf(`{"name":"Kho 1","legal_entity_id":%d,"active":true}`, f.company))
	move := func(c *client, typ string, id int64, to string) {
		var d struct{ Version int32 }
		path := map[string]string{"sales.order": "/api/sales/orders/%d", "sales.delivery": "/api/sales/deliveries/%d", "stock.adjustment": "/api/stock/adjustments/%d"}[typ]
		if err := json.Unmarshal(f.ok(c, "GET", fmt.Sprintf(path, id), "", 200), &d); err != nil {
			t.Fatal(err)
		}
		f.ok(c, "POST", fmt.Sprintf("/api/documents/%s/%d/transitions", typ, id), fmt.Sprintf(`{"to":%q,"version":%d}`, to, d.Version), 200)
	}
	f.moveQuote("posted")
	var o struct {
		ID   int64
		Date string
	}
	if err := json.Unmarshal(f.ok(f.staffA, "POST", fmt.Sprintf("/api/sales/quotes/%d/order", f.quote), "", 200), &o); err != nil {
		t.Fatal(err)
	}
	move(f.staffA, "sales.order", o.ID, "posted")
	// The order is dated the day it was made: the stock and the delivery use that day.
	if err := json.Unmarshal(f.ok(f.staffA, "GET", fmt.Sprintf("/api/sales/orders/%d", o.ID), "", 200), &o); err != nil {
		t.Fatal(err)
	}
	day := o.Date
	adj := f.id(f.admin, "/api/stock/adjustments", fmt.Sprintf(`{"request_id":"00000000-0000-4000-8000-000000000001","date":%q,"warehouse_id":%d,
		"reason":"Kiểm kê","lines":[{"item_id":%d,"quantity":"10"}]}`, day, wh, item))
	move(f.admin, "stock.adjustment", adj, "posted")
	del := f.id(f.staffA, "/api/sales/deliveries", fmt.Sprintf(`{"request_id":"00000000-0000-4000-8000-000000000002","order_id":%d,"date":%q,
		"warehouse_id":%d,"note":null,"lines":[{"order_position":1,"quantity":"4"}]}`, o.ID, day, wh))
	move(f.staffA, "sales.delivery", del, "posted")

	user := func(login string, roles ...string) *client {
		id := f.id(f.admin, "/api/users", fmt.Sprintf(`{"login":%q,"name":%q,"password":"correct horse"}`, login, login))
		for _, r := range roles {
			product, rest, _ := strings.Cut(r, ".")
			role, unit, _ := strings.Cut(rest, "@")
			f.id(f.admin, fmt.Sprintf("/api/users/%d/roles", id), fmt.Sprintf(`{"product":%q,"role":%q,"org_unit_id":%s}`, product, role, unit))
		}
		return f.login(login)
	}
	company := fmt.Sprint(f.company)
	viewer := user("viewer", "stock.viewer@"+company)
	seller := user("seller", "stock.viewer@"+company, "sales.staff@"+fmt.Sprint(f.a))
	card := func(c *client) (rows []struct {
		DocType string `json:"doc_type"`
		DocID   int64  `json:"doc_id"`
		Number  string `json:"number"`
	}) {
		var out struct {
			Rows []struct {
				DocType string `json:"doc_type"`
				DocID   int64  `json:"doc_id"`
				Number  string `json:"number"`
			}
		}
		if err := json.Unmarshal(f.ok(c, "GET", fmt.Sprintf("/api/stock/card?warehouse_id=%d&item_id=%d&from=%s&to=%s", wh, item, day, day), "", 200), &out); err != nil {
			t.Fatal(err)
		}
		return out.Rows
	}
	rows := card(viewer)
	if len(rows) != 2 || rows[0].DocID != adj || rows[0].Number == "" || rows[1].DocType != "sales.delivery" || rows[1].DocID != 0 || rows[1].Number != "" {
		t.Fatalf("viewer card: %+v", rows)
	}
	rows = card(seller)
	if len(rows) != 2 || rows[1].DocID != del || rows[1].Number == "" {
		t.Fatalf("seller card: %+v", rows)
	}
}
