package platform

import (
	"context"
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"time"
)

// Files keeps file contents on local disk under Dir; metadata belongs to the module
// that owns the file. Write the file before its metadata, so a row never names a
// missing file. Each module works in its own Sub, so its cleanup never sees the
// files of another.
type Files struct{ Dir string }

// Sub is the store of one module, a subdirectory created on first save.
func (f Files) Sub(module string) Files { return Files{Dir: filepath.Join(f.Dir, module)} }

var fileID = regexp.MustCompile(`^[A-Z2-7]{26}$`)

// Save writes r under a new random id. A crash leaves at most a temporary file.
func (f Files) Save(r io.Reader) (string, error) {
	if err := os.MkdirAll(f.Dir, 0o750); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(f.Dir, ".tmp-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	id := rand.Text()
	return id, os.Rename(tmp.Name(), filepath.Join(f.Dir, id))
}

// Open reads a saved file; an id that Save could not have made does not exist.
func (f Files) Open(id string) (*os.File, error) {
	if !fileID.MatchString(id) {
		return nil, os.ErrNotExist
	}
	return os.Open(filepath.Join(f.Dir, id))
}

// Remove deletes a file named by Save or listed by Older; one already gone is not an error.
func (f Files) Remove(name string) error {
	if name != filepath.Base(name) || name == "." || name == ".." {
		return os.ErrInvalid
	}
	if err := os.Remove(filepath.Join(f.Dir, name)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Older lists the files, temporary ones included, last written before t, for cleanup.
// Subdirectories belong to other modules and are skipped.
func (f Files) Older(t time.Time) ([]string, error) {
	entries, err := os.ReadDir(f.Dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || e.IsDir() || !info.ModTime().Before(t) {
			continue
		}
		out = append(out, e.Name())
	}
	return out, nil
}

// Sweep removes the files last written before t that known does not name: those left
// by a failure between a file and its row, or whose row was removed.
func (f Files) Sweep(ctx context.Context, t time.Time, known func(ctx context.Context, names []string) ([]string, error)) error {
	old, err := f.Older(t)
	if err != nil || len(old) == 0 {
		return err
	}
	keep, err := known(ctx, old)
	if err != nil {
		return err
	}
	for _, name := range old {
		if !slices.Contains(keep, name) {
			if err := f.Remove(name); err != nil {
				return err
			}
		}
	}
	return nil
}

type filesKey struct{}

// WithFiles puts the file store into ctx; only internal/app calls it.
func WithFiles(ctx context.Context, f Files) context.Context {
	return context.WithValue(ctx, filesKey{}, f)
}

// FilesFrom returns the file store.
func FilesFrom(ctx context.Context) Files {
	f, ok := ctx.Value(filesKey{}).(Files)
	if !ok {
		// Wiring bug, not a runtime condition.
		panic("platform: no file store in context")
	}
	return f
}
