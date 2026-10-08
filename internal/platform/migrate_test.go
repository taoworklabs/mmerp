package platform_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

func sqlFS(files map[string]string) fstest.MapFS {
	m := fstest.MapFS{}
	for name, sql := range files {
		m[name] = &fstest.MapFile{Data: []byte(sql)}
	}
	return m
}

func versions(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(t.Context(), `SELECT version FROM public.schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		out = append(out, v)
	}
	return out
}

func TestMigrateAppliesInOrderOnce(t *testing.T) {
	pool := pgtest.Empty(t)
	fsys := sqlFS(map[string]string{
		"0002_demo_add_col.sql": `ALTER TABLE demo.items ADD COLUMN name text;`,
		"0001_demo_init.sql":    `CREATE SCHEMA demo; CREATE TABLE demo.items (id int);`,
		"README.md":             `ignored`,
	})
	for range 2 {
		if err := platform.Migrate(t.Context(), pool, fsys); err != nil {
			t.Fatal(err)
		}
	}
	if got := versions(t, pool); strings.Join(got, ",") != "0001_demo_init.sql,0002_demo_add_col.sql" {
		t.Fatalf("versions = %v", got)
	}
	var river bool
	if err := pool.QueryRow(t.Context(), `SELECT to_regclass('public.river_job') IS NOT NULL`).Scan(&river); err != nil || !river {
		t.Fatalf("River tables: %v %v", river, err)
	}
}

func TestMigrateRollsBackEverythingOnFailure(t *testing.T) {
	pool := pgtest.Empty(t)
	err := platform.Migrate(t.Context(), pool, sqlFS(map[string]string{
		"0001_demo_init.sql":   `CREATE SCHEMA demo;`,
		"0002_demo_broken.sql": `CREATE TABLE nowhere.items (id int);`,
	}))
	if err == nil || !strings.Contains(err.Error(), "0002_demo_broken.sql") {
		t.Fatalf("err = %v", err)
	}
	var n int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM pg_namespace WHERE nspname = 'demo'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("schema demo survived a failed migration")
	}
}

func TestMigrateRefusesUnknownAppliedVersion(t *testing.T) {
	pool := pgtest.Empty(t)
	if err := platform.Migrate(t.Context(), pool, sqlFS(map[string]string{
		"0001_demo_init.sql": `SELECT 1;`,
		"0002_demo_next.sql": `SELECT 1;`,
	})); err != nil {
		t.Fatal(err)
	}
	err := platform.Migrate(t.Context(), pool, sqlFS(map[string]string{"0001_demo_init.sql": `SELECT 1;`}))
	if err == nil || !strings.Contains(err.Error(), "schema_newer_than_binary") {
		t.Fatalf("err = %v", err)
	}
}

func TestMigrateRejectsBadFileName(t *testing.T) {
	pool := pgtest.Empty(t)
	err := platform.Migrate(t.Context(), pool, sqlFS(map[string]string{"1_init.sql": `SELECT 1;`}))
	if err == nil || !strings.Contains(err.Error(), "1_init.sql") {
		t.Fatalf("err = %v", err)
	}
}

func TestMigrateConcurrentRunsApplyOnce(t *testing.T) {
	pool := pgtest.Empty(t)
	// A non-idempotent statement: running it twice fails.
	fsys := sqlFS(map[string]string{"0001_demo_init.sql": `CREATE SCHEMA demo;`})
	var wg sync.WaitGroup
	errs := make([]error, 4)
	for i := range errs {
		wg.Go(func() { errs[i] = platform.Migrate(context.Background(), pool, fsys) })
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := versions(t, pool); len(got) != 1 {
		t.Fatalf("versions = %v", got)
	}
}
