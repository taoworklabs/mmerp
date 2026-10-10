package setting_test

import (
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"

	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

func TestGetFallsBackToDefault(t *testing.T) {
	ctx := platform.WithDB(t.Context(), pgtest.New(t))
	s := setting.NewService()
	if v, err := s.Get(ctx, setting.Locale); err != nil || v != "vi" {
		t.Fatalf("default locale = %q, %v", v, err)
	}
	if _, err := platform.DBFrom(ctx).Exec(ctx, `INSERT INTO setting.values VALUES ('setting.locale', 'en')`); err != nil {
		t.Fatal(err)
	}
	if v, err := s.Get(ctx, setting.Locale); err != nil || v != "en" {
		t.Fatalf("stored locale = %q, %v", v, err)
	}
	if _, err := s.Get(ctx, "setting.nope"); err == nil {
		t.Fatal("unknown key: want error")
	}
}

func TestLegalEntitySettings(t *testing.T) {
	ctx := platform.WithDB(t.Context(), pgtest.New(t))
	set := setting.NewService()
	ids := iam.NewService(iam.Deps{Setting: set, Audit: audit.NewService()})
	set.SetAuthz(ids)
	set.RegisterLegalEntityKey("hrm", "hrm.wage_region", "1", []string{"1", "2", "3", "4"})
	admin := platform.WithActor(ctx, must(ids.CreateAdmin(ctx, "admin", "Admin", "long enough")))
	c := must(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	dept := must(ids.CreateOrgUnit(admin, iam.OrgUnitInput{ParentID: &c, Kind: "department", Name: "A"}))

	if v, err := set.GetFor(ctx, c, setting.Rounding); err != nil || v != "line" {
		t.Fatalf("default rounding = %q, %v", v, err)
	}
	if err := set.SetFor(admin, c, setting.Rounding, "half"); errCode(err) != "invalid_setting_value" {
		t.Fatalf("bad value: %v", err)
	}
	if err := set.SetFor(admin, dept, setting.Rounding, "total"); errCode(err) != "not_found" {
		t.Fatalf("not a legal entity: %v", err)
	}
	nobody := platform.WithActor(ctx, must(ids.CreateUser(admin, "nobody", "Nobody", "long enough")))
	if err := set.SetFor(nobody, c, setting.Rounding, "total"); errCode(err) != "forbidden" {
		t.Fatalf("without core.setting.manage: %v", err)
	}
	if err := set.SetFor(admin, c, setting.Rounding, "total"); err != nil {
		t.Fatal(err)
	}
	// A product's setting is written only while the product is enabled.
	if err := set.SetFor(admin, c, "hrm.wage_region", "2"); errCode(err) != "product_not_enabled" {
		t.Fatalf("hrm off: %v", err)
	}
	l, err := set.LegalEntitySettings(admin, c)
	if err != nil || len(l) != 2 || l[0].Key != "hrm.wage_region" || l[0].Value != "1" || l[1].Value != "total" {
		t.Fatalf("settings = %+v, %v", l, err)
	}
	var n int
	if err := platform.DBFrom(ctx).QueryRow(ctx, `SELECT count(*) FROM audit.log WHERE action = 'setting.changed'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("audited %d, %v", n, err)
	}
}

func must(v int64, err error) int64 {
	if err != nil {
		panic(err)
	}
	return v
}

func errCode(err error) string {
	if e, ok := errors.AsType[*platform.Error](err); ok {
		return e.Code
	}
	return ""
}
