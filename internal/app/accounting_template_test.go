package app_test

import (
	"encoding/json"
	"testing"

	"github.com/taoworklabs/mmerp/internal/app"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// With every product wired, each chart template maps every declared business line kind, so
// no product's documents wait under awaiting mapping on a fresh tenant.
func TestChartTemplatesMapEveryKind(t *testing.T) {
	for _, code := range []string{"vn-tt133", "vn-tt99"} {
		t.Run(code, func(t *testing.T) {
			pool := pgtest.New(t)
			if err := app.CreateAdmin(t.Context(), pool, "admin", "Admin", "correct horse"); err != nil {
				t.Fatal(err)
			}
			f := &jobsFixture{t: t, env: env(t, pool, []string{"accounting"})}
			f.system = f.env.SystemPool
			f.admin = f.login("admin")
			f.ok(f.admin, "POST", "/api/accounting/templates/"+code+"/apply", "", 204)
			wantCode(t, f.admin.do("POST", "/api/accounting/templates/"+code+"/apply", ""), 409, "chart_not_empty")
			var rules struct {
				Items []struct {
					Kind      string `json:"kind"`
					AccountID *int64 `json:"account_id"`
				} `json:"items"`
			}
			if err := json.Unmarshal(f.ok(f.admin, "GET", "/api/accounting/rules", "", 200), &rules); err != nil {
				t.Fatal(err)
			}
			if len(rules.Items) < 25 {
				t.Fatalf("%d kinds", len(rules.Items))
			}
			for _, r := range rules.Items {
				if r.AccountID == nil {
					t.Errorf("%s has no rule", r.Kind)
				}
			}
		})
	}
}
