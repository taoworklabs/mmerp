package platform

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

// Arbitrary constant shared by every binary version.
const migrationLockKey = 7_346_201_001

var migrationName = regexp.MustCompile(`^[0-9]{4}_[a-z]+_[a-z0-9_]+\.sql$`)

// Migrate applies every missing migration in fsys, then River's, in one
// transaction, so a failure leaves the schema at its previous version.
// ponytail: transaction-scoped lock; switch to a session lock when backup must run between locking and migrating.
func Migrate(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) error {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || path.Ext(e.Name()) != ".sql" {
			continue
		}
		if !migrationName.MatchString(e.Name()) {
			return fmt.Errorf("migration %q: name must match NNNN_<module>_<description>.sql", e.Name())
		}
		files = append(files, e.Name())
	}
	slices.Sort(files) // ReadDir already sorts; keep the order explicit.

	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, migrationLockKey); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS public.schema_migrations (
			version text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
			return err
		}
		rows, _ := tx.Query(ctx, `SELECT version FROM public.schema_migrations`)
		applied, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		for _, v := range applied {
			// An older binary must never run on a schema it does not know.
			if !slices.Contains(files, v) {
				return fmt.Errorf("schema_newer_than_binary: database has migration %q unknown to this binary", v)
			}
		}
		for _, f := range files {
			if slices.Contains(applied, f) {
				continue
			}
			sql, err := fs.ReadFile(fsys, f)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, string(sql)); err != nil {
				return fmt.Errorf("migration %s: %w", f, err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO public.schema_migrations (version) VALUES ($1)`, f); err != nil {
				return err
			}
		}
		m, err := RiverMigrator()
		if err != nil {
			return err
		}
		// Deprecated because some River upgrades cannot share a transaction; ours must, so a
		// River upgrade that needs otherwise fails here, in tests, before it ships.
		_, err = m.MigrateTx(ctx, tx, rivermigrate.DirectionUp, nil) //nolint:staticcheck

		return err
	})
}

// RiverMigrator migrates River's own tables; its versions are part of the schema.
func RiverMigrator() (*rivermigrate.Migrator[pgx.Tx], error) {
	return rivermigrate.New(riverpgxv5.New(nil), nil)
}
