package platform_test

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/taoworklabs/mmerp/internal/platform"
)

func TestFiles(t *testing.T) {
	f := platform.Files{Dir: t.TempDir()}
	id, err := f.Save(strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := f.Open(id)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(r)
	_ = r.Close()
	if string(b) != "hello" {
		t.Fatalf("read %q", b)
	}
	// Only ids Save makes reach the disk.
	if _, err := f.Open("../" + id); !os.IsNotExist(err) {
		t.Fatalf("open outside: %v", err)
	}
	if err := f.Remove("../x"); err == nil {
		t.Fatal("remove outside")
	}

	if old, err := f.Older(time.Now().Add(-time.Hour)); err != nil || len(old) != 0 {
		t.Fatalf("older than an hour: %v %v", old, err)
	}
	old, err := f.Older(time.Now().Add(time.Second))
	if err != nil || len(old) != 1 || old[0] != id {
		t.Fatalf("older: %v %v", old, err)
	}
	for range 2 {
		if err := f.Remove(id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.Open(id); !os.IsNotExist(err) {
		t.Fatalf("open removed: %v", err)
	}
}

// A module's files live in its own subdirectory, out of sight of the parent's cleanup.
func TestFilesSub(t *testing.T) {
	root := platform.Files{Dir: t.TempDir()}
	sub := root.Sub("attachment")
	if old, err := sub.Older(time.Now().Add(time.Second)); err != nil || len(old) != 0 {
		t.Fatalf("older before any save: %v %v", old, err)
	}
	id, err := sub.Save(strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := root.Open(id); !os.IsNotExist(err) {
		t.Fatalf("parent opened a sub file: %v", err)
	}
	if old, err := root.Older(time.Now().Add(time.Second)); err != nil || len(old) != 0 {
		t.Fatalf("parent lists the sub: %v %v", old, err)
	}
	if old, err := sub.Older(time.Now().Add(time.Second)); err != nil || len(old) != 1 || old[0] != id {
		t.Fatalf("sub older: %v %v", old, err)
	}
}
