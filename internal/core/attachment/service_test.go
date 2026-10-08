package attachment

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

var (
	thing   = record.Ref{Type: "test.thing", ID: 1}
	errBoom = errors.New("boom")
	pdf     = "%PDF-1.4\n"
)

// setup gives a service over one catalog type anyone may do anything with, and a
// context with an actor, the product enabled and a fresh file root.
func setup(t *testing.T) (*Service, context.Context, platform.Files) {
	t.Helper()
	pool := pgtest.New(t)
	root := platform.Files{Dir: t.TempDir()}
	ctx := platform.WithProducts(platform.WithFiles(platform.WithDB(t.Context(), pool), root), []string{"test"})
	var actor int64
	if err := pool.QueryRow(ctx, `INSERT INTO iam.users (login, name, password_hash) VALUES ('u', 'U', '') RETURNING id`).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	rec := record.NewService(record.Deps{Audit: audit.NewService()})
	rec.Register(record.Type{Code: thing.Type, Product: "test", Kind: record.Catalog,
		Can: func(context.Context, int64, record.Action) (bool, error) { return true, nil }})
	return NewService(Deps{Record: rec, Audit: audit.NewService()}), platform.WithActor(ctx, actor), root
}

// age makes every file of dir look written two days ago.
func age(t *testing.T, dir string) {
	t.Helper()
	old := time.Now().Add(-48 * time.Hour)
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if err := os.Chtimes(filepath.Join(dir, e.Name()), old, old); err != nil {
			t.Fatal(err)
		}
	}
}

func count(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return len(entries)
}

// An upload whose transaction rolls back leaves no row, and cleanup removes its file.
func TestAddRolledBack(t *testing.T) {
	s, ctx, root := setup(t)
	err := platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.Add(ctx, thing, "a.pdf", strings.NewReader(pdf)); err != nil {
			return err
		}
		return errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatal(err)
	}
	if l, err := s.List(ctx, thing); err != nil || len(l.Items) != 0 {
		t.Fatalf("list = %+v %v", l, err)
	}
	dir := root.Sub("attachment").Dir
	if count(t, dir) != 1 {
		t.Fatal("the upload was not kept on disk")
	}
	age(t, dir)
	if err := cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, dir); n != 0 {
		t.Fatalf("%d files left", n)
	}
}

// A removal whose transaction rolls back keeps the row, and its file still downloads.
func TestRemoveRolledBack(t *testing.T) {
	s, ctx, _ := setup(t)
	if err := s.Add(ctx, thing, "a.pdf", strings.NewReader(pdf)); err != nil {
		t.Fatal(err)
	}
	l, _ := s.List(ctx, thing)
	id := l.Items[0].ID
	err := platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.Remove(ctx, id); err != nil {
			return err
		}
		return errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatal(err)
	}
	_, f, err := s.Open(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(f)
	_ = f.Close()
	if string(b) != pdf {
		t.Fatalf("read %q", b)
	}
}

// Cleanup removes old files no row names and keeps the rest, other modules' files included.
func TestCleanup(t *testing.T) {
	s, ctx, root := setup(t)
	if err := s.Add(ctx, thing, "kept.pdf", strings.NewReader(pdf)); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(ctx, thing, "removed.pdf", strings.NewReader(pdf)); err != nil {
		t.Fatal(err)
	}
	l, _ := s.List(ctx, thing)
	if err := s.Remove(ctx, l.Items[1].ID); err != nil {
		t.Fatal(err)
	}
	var others []string
	for _, f := range []platform.Files{root, root.Sub("dataio")} {
		id, err := f.Save(strings.NewReader("x"))
		if err != nil {
			t.Fatal(err)
		}
		age(t, f.Dir)
		others = append(others, filepath.Join(f.Dir, id))
	}
	dir := root.Sub("attachment").Dir
	age(t, dir)
	fresh, err := root.Sub("attachment").Save(strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	if err := cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, dir); n != 2 {
		t.Fatalf("%d files left, want the kept one and the fresh one", n)
	}
	if _, err := os.Stat(filepath.Join(dir, fresh)); err != nil {
		t.Fatal("fresh file removed")
	}
	if _, f, err := s.Open(ctx, l.Items[0].ID); err != nil {
		t.Fatalf("kept file: %v", err)
	} else {
		_ = f.Close()
	}
	for _, p := range others {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("another module's file removed: %v", err)
		}
	}
}

// A draft's deletion that rolls back keeps its attachments; one that commits leaves its
// file to cleanup.
func TestDeletedWithTheDraft(t *testing.T) {
	s, ctx, root := setup(t)
	if err := s.Add(ctx, thing, "a.pdf", strings.NewReader(pdf)); err != nil {
		t.Fatal(err)
	}
	err := platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.Deleted(ctx, thing); err != nil {
			return err
		}
		return errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatal(err)
	}
	if l, err := s.List(ctx, thing); err != nil || len(l.Items) != 1 {
		t.Fatalf("after a rolled-back deletion: %+v %v", l, err)
	}
	if err := s.Deleted(ctx, thing); err != nil {
		t.Fatal(err)
	}
	dir := root.Sub("attachment").Dir
	age(t, dir)
	if err := cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, dir); n != 0 {
		t.Fatalf("%d files left after the draft went", n)
	}
}
