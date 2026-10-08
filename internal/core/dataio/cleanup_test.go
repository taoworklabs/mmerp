package dataio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// Cleanup removes expired files and old files no row names, and keeps the rest,
// including old files of other modules.
func TestCleanup(t *testing.T) {
	pool := pgtest.New(t)
	root := platform.Files{Dir: t.TempDir()}
	files := root.Sub("dataio")
	ctx := platform.WithFiles(platform.WithDB(t.Context(), pool), root)
	var owner int64
	if err := pool.QueryRow(ctx, `INSERT INTO iam.users (login, name, password_hash) VALUES ('u', 'U', '') RETURNING id`).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	save := func() string {
		id, err := files.Save(strings.NewReader("x"))
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	expired, live, orphan, fresh := save(), save(), save(), save()
	var neighbours []string
	for _, other := range []platform.Files{root, root.Sub("attachment")} {
		id, err := other.Save(strings.NewReader("x"))
		if err != nil {
			t.Fatal(err)
		}
		neighbours = append(neighbours, filepath.Join(other.Dir, id))
	}
	for id, expires := range map[string]string{expired: "now() - interval '1 minute'", live: "now() + interval '1 hour'"} {
		if _, err := pool.Exec(ctx, `INSERT INTO dataio.files (id, name, owner_id, expires_at) VALUES ($1, 'f.xlsx', $2, `+expires+`)`, id, owner); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-25 * time.Hour)
	for _, id := range []string{expired, live, orphan} {
		neighbours = append(neighbours, filepath.Join(files.Dir, id))
	}
	for _, path := range neighbours {
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	if err := cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]bool{expired: false, live: true, orphan: false, fresh: true} {
		if _, err := os.Stat(filepath.Join(files.Dir, id)); (err == nil) != want {
			t.Errorf("file kept = %v, want %v", err == nil, want)
		}
	}
	for _, path := range neighbours[:2] {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("another module's file removed: %v", err)
		}
	}
	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM dataio.files`).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("rows %d %v", rows, err)
	}
}
