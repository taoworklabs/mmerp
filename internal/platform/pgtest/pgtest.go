// Package pgtest gives each test its own Postgres database. Never mock the database.
package pgtest

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/migrations"
)

const defaultURL = "postgres://mmerp:mmerp@localhost:5433/mmerp?sslmode=disable"

// Serialises template creation across test binaries running in parallel.
const templateLockKey = 7_346_201_002

// New returns a pool on a fresh database cloned from the migrated template.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	return create(t, template(t))
}

// Empty returns a pool on a fresh database with no migrations applied.
func Empty(t testing.TB) *pgxpool.Pool {
	t.Helper()
	return create(t, "template0")
}

func baseURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return defaultURL
}

func admin(t testing.TB) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(t.Context(), baseURL())
	if err != nil {
		t.Fatalf("pgtest: connect (is Postgres running? docker compose up -d postgres): %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

func create(t testing.TB, tpl string) *pgxpool.Pool {
	t.Helper()
	ctx := t.Context()
	name := "t_" + rand.Text()[:16]
	conn := admin(t)
	if _, err := conn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %q TEMPLATE %q`, name, tpl)); err != nil {
		t.Fatalf("pgtest: create database: %v", err)
	}
	pool, err := pgxpool.New(ctx, urlFor(t, name))
	if err != nil {
		t.Fatalf("pgtest: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		c, err := pgx.Connect(context.Background(), baseURL())
		if err != nil {
			return
		}
		defer func() { _ = c.Close(context.Background()) }()
		_, _ = c.Exec(context.Background(), fmt.Sprintf(`DROP DATABASE %q WITH (FORCE)`, name))
	})
	return pool
}

// template creates the migrated template once per migration set; its name
// carries a hash of the migrations so a change builds a new one.
func template(t testing.TB) string {
	t.Helper()
	ctx := t.Context()
	h := sha256.New()
	err := fs.WalkDir(migrations.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(migrations.FS, p)
		h.Write([]byte(p))
		h.Write(b)
		return err
	})
	if err != nil {
		t.Fatalf("pgtest: %v", err)
	}
	// River's migrations ship with the library, so a River upgrade must build a new template too.
	river, err := platform.RiverMigrator()
	if err != nil {
		t.Fatalf("pgtest: %v", err)
	}
	_, _ = fmt.Fprint(h, len(river.AllVersions()))
	name := "mmerp_tpl_" + hex.EncodeToString(h.Sum(nil))[:12]

	conn := admin(t)
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, templateLockKey); err != nil {
		t.Fatalf("pgtest: %v", err)
	}
	defer func() { _, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, templateLockKey) }()

	var exists bool
	if err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&exists); err != nil {
		t.Fatalf("pgtest: %v", err)
	}
	if exists {
		return name
	}
	if _, err := conn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %q TEMPLATE template0`, name)); err != nil {
		t.Fatalf("pgtest: %v", err)
	}
	pool, err := pgxpool.New(ctx, urlFor(t, name))
	if err != nil {
		t.Fatalf("pgtest: %v", err)
	}
	defer pool.Close()
	if err := platform.Migrate(ctx, pool, migrations.FS); err != nil {
		// Do not leave a half-built template behind.
		pool.Close()
		_, _ = conn.Exec(context.Background(), fmt.Sprintf(`DROP DATABASE %q`, name))
		t.Fatalf("pgtest: migrate template: %v", err)
	}
	return name
}

func urlFor(t testing.TB, db string) string {
	t.Helper()
	u, err := url.Parse(baseURL())
	if err != nil {
		t.Fatalf("pgtest: TEST_DATABASE_URL: %v", err)
	}
	u.Path = "/" + db
	return u.String()
}

// Keyring is a fixed column-encryption keyring for tests.
func Keyring() *platform.Keyring {
	k, err := platform.NewKeyring(map[byte][]byte{1: []byte("pgtest key, 32 bytes, never real")})
	if err != nil {
		panic(err)
	}
	return k
}
