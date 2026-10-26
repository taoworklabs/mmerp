// Package datatest measures dynamic lists against the fixed queries they replace.
package datatest

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// Stats is a latency distribution.
type Stats struct{ P50, P95 time.Duration }

// Measure runs fn warmup times unmeasured, then runs times, and reports the
// percentiles of those runs.
func Measure(tb testing.TB, warmup, runs int, fn func()) Stats {
	tb.Helper()
	for range warmup {
		fn()
	}
	d := make([]time.Duration, runs)
	for i := range d {
		start := time.Now()
		fn()
		d[i] = time.Since(start)
	}
	slices.Sort(d)
	return Stats{P50: d[runs*50/100], P95: d[runs*95/100]}
}

// Explain returns EXPLAIN (ANALYZE, BUFFERS) of c under plan_cache_mode mode
// (force_custom_plan or force_generic_plan). The statement is prepared and
// executed with c.Args, as pgx's statement cache does, inside a transaction that
// is rolled back.
func Explain(tb testing.TB, conn platform.Conn, c data.Compiled, mode string) string {
	tb.Helper()
	if mode != "force_custom_plan" && mode != "force_generic_plan" {
		tb.Fatalf("datatest: plan_cache_mode %q", mode)
	}
	b, ok := conn.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		tb.Fatalf("datatest: %T cannot begin a transaction", conn)
	}
	ctx := tb.Context()
	tx, err := b.Begin(ctx)
	if err != nil {
		tb.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// PREPARE outlives the transaction, so it is deallocated explicitly.
	defer func() { _, _ = tx.Exec(ctx, "DEALLOCATE datatest_explain") }()

	params := make([]string, len(c.Args))
	for i := range params {
		params[i] = "$" + strconv.Itoa(i+1)
	}
	exec := "EXECUTE datatest_explain"
	if len(params) > 0 {
		exec += "(" + strings.Join(params, ", ") + ")"
	}
	var out []string
	err = errors.Join(
		execErr(tx.Exec(ctx, "SET LOCAL plan_cache_mode = "+mode)),
		execErr(tx.Exec(ctx, "PREPARE datatest_explain AS "+c.SQL)),
	)
	if err == nil {
		var rows pgx.Rows
		// EXECUTE's arguments cannot be bound parameters, so pgx inlines them as literals.
		args := append([]any{pgx.QueryExecModeSimpleProtocol}, c.Args...)
		rows, err = tx.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS) "+exec, args...)
		if err == nil {
			out, err = pgx.CollectRows(rows, pgx.RowTo[string])
		}
	}
	if err != nil {
		tb.Fatalf("datatest: explain %s: %v", c.Name, err)
	}
	return strings.Join(out, "\n")
}

func execErr(_ any, err error) error { return err }
