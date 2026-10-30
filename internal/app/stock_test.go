package app_test

import (
	"fmt"
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
