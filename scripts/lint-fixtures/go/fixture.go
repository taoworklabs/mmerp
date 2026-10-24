//go:build lintfixture

// Package fixture plants one violation per data-access rule configured in
// .golangci.yml, so a misconfigured pattern fails loudly instead of silently
// letting business code call these again. The lintfixture tag keeps this file
// and its store package out of every normal build, vet and lint run; only
// scripts/check-go-lint-fixtures.sh builds with that tag.
package fixture

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/scripts/lint-fixtures/go/store"
)

// jobArgs is the minimal river.JobArgs implementation the Insert/InsertTx violations need.
type jobArgs struct{}

func (jobArgs) Kind() string { return "lintfixture" }

// Violations calls exactly the forms each rule bans, one per line.
func Violations(ctx context.Context) {
	var conn platform.Conn
	_ = store.New(conn) // forbidigo: store.New

	_ = platform.DBFrom(ctx) // forbidigo: platform.DBFrom

	pool := &pgxpool.Pool{}
	_, _ = pool.Begin(ctx) // forbidigo: .Begin(

	_ = pgx.BeginFunc(ctx, pool, func(pgx.Tx) error { return nil }) // forbidigo: pgx.BeginFunc

	client := &river.Client[pgx.Tx]{}
	_, _ = client.Insert(ctx, jobArgs{}, nil) // forbidigo: river.Client.Insert

	var tx pgx.Tx
	_, _ = client.InsertTx(ctx, tx, jobArgs{}, nil) // forbidigo: river.Client.InsertTx

	_ = data.Static("select 1")     // forbidigo: data.Static
	_ = data.GrantAll()             // forbidigo: data.GrantAll
	_ = data.GrantUnits([]int64{1}) // forbidigo: data.GrantUnits

	_ = riverpgxv5.New(nil) // depguard: riverpgxv5 client construction outside internal/app
}
