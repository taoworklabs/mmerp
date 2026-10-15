package numbering_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

type fixture struct {
	num          *numbering.Service
	admin, other context.Context
}

func setup(t *testing.T) fixture {
	ctx := platform.WithProducts(platform.WithDB(t.Context(), pgtest.New(t)), []string{"sales"})
	ids := iam.NewService(iam.Deps{Setting: setting.NewService(), Audit: audit.NewService()})
	num := numbering.NewService(numbering.Deps{IAM: ids, Audit: audit.NewService()})
	num.RegisterSeries("sales.customer", "sales", "KH")
	num.RegisterSeries("hrm.employee", "hrm", "NV")
	admin := platform.WithActor(ctx, must(ids.CreateAdmin(ctx, "admin", "Admin", "long enough")))
	other := platform.WithActor(ctx, must(ids.CreateUser(admin, "nobody", "Nobody", "long enough")))
	return fixture{num, admin, other}
}

func free(context.Context, string) (bool, error) { return false, nil }

func TestCode(t *testing.T) {
	f := setup(t)
	if c, err := f.num.Code(f.admin, "sales.customer", " own ", free); err != nil || c != "own" {
		t.Fatalf("typed code: %q, %v", c, err)
	}
	for _, want := range []string{"KH00001", "KH00002"} {
		if c, err := f.num.Code(f.admin, "sales.customer", "", free); err != nil || c != want {
			t.Fatalf("code = %q, %v; want %s", c, err, want)
		}
	}
	// Codes typed by hand are skipped.
	typed := func(_ context.Context, c string) (bool, error) { return c == "KH00003" || c == "KH00004", nil }
	if c, err := f.num.Code(f.admin, "sales.customer", "", typed); err != nil || c != "KH00005" {
		t.Fatalf("skipping typed codes: %q, %v", c, err)
	}
	// A rolled-back transaction gives its number back.
	boom := errors.New("boom")
	err := platform.InTx(f.admin, func(ctx context.Context) error {
		if _, err := f.num.Code(ctx, "sales.customer", "", free); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatal(err)
	}
	if c, err := f.num.Code(f.admin, "sales.customer", "", free); err != nil || c != "KH00006" {
		t.Fatalf("after rollback: %q, %v", c, err)
	}
	// Series count on their own.
	if c, err := f.num.Code(f.admin, "hrm.employee", "", free); err != nil || c != "NV00001" {
		t.Fatalf("other series: %q, %v", c, err)
	}
}

func TestSetPrefix(t *testing.T) {
	f := setup(t)
	must2(f.num.Code(f.admin, "sales.customer", "", free))

	for _, bad := range []string{"ABCDEFGHIJK", "K H", "KH\t"} {
		if err := f.num.SetPrefix(f.admin, "sales.customer", bad); errCode(err) != "invalid_code_prefix" {
			t.Fatalf("prefix %q: %v", bad, err)
		}
	}
	if err := f.num.SetPrefix(f.other, "sales.customer", "CUS"); errCode(err) != "forbidden" {
		t.Fatalf("without core.code_series.manage: %v", err)
	}
	if _, err := f.num.Series(f.other); errCode(err) != "forbidden" {
		t.Fatalf("list without permission: %v", err)
	}
	if err := f.num.SetPrefix(f.admin, "hrm.employee", "E"); errCode(err) != "product_not_enabled" {
		t.Fatalf("hrm off: %v", err)
	}
	if err := f.num.SetPrefix(f.admin, "nope", "X"); errCode(err) != "not_found" {
		t.Fatalf("unknown series: %v", err)
	}
	if err := f.num.SetPrefix(f.admin, "sales.customer", "CUS"); err != nil {
		t.Fatal(err)
	}
	// The count goes on under the new prefix.
	if c, err := f.num.Code(f.admin, "sales.customer", "", free); err != nil || c != "CUS00002" {
		t.Fatalf("after prefix change: %q, %v", c, err)
	}
	if err := f.num.SetPrefix(f.admin, "sales.customer", ""); err != nil {
		t.Fatal(err)
	}

	l, err := f.num.Series(f.admin)
	if err != nil {
		t.Fatal(err)
	}
	want := []numbering.CodeSeries{
		{Code: "hrm.employee", Product: "hrm", Prefix: "NV", AllowedActions: []string{}},
		{Code: "sales.customer", Product: "sales", Prefix: "", AllowedActions: []string{"edit"}},
	}
	if !slices.EqualFunc(l, want, func(a, b numbering.CodeSeries) bool {
		return a.Code == b.Code && a.Product == b.Product && a.Prefix == b.Prefix && slices.Equal(a.AllowedActions, b.AllowedActions)
	}) {
		t.Fatalf("series = %+v", l)
	}
	var n int
	if err := platform.DBFrom(f.admin).QueryRow(f.admin, `SELECT count(*) FROM audit.log WHERE action = 'numbering.prefix_changed'
		AND data->>'series' = 'sales.customer' AND (data->>'old', data->>'new') IN (('KH', 'CUS'), ('CUS', ''))`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("audited %d, %v", n, err)
	}
}

func must(v int64, err error) int64 {
	if err != nil {
		panic(err)
	}
	return v
}

func must2(v string, err error) string {
	if err != nil {
		panic(err)
	}
	return v
}

func errCode(err error) string {
	if e, ok := errors.AsType[*platform.Error](err); ok {
		return e.Code
	}
	if err != nil {
		return err.Error()
	}
	return ""
}
