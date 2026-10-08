package platform_test

import (
	"strings"
	"testing"

	"github.com/taoworklabs/mmerp/internal/platform"
)

func TestLoadConfigNamesEveryBadVariable(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("LOG_LEVEL", "loud")
	t.Setenv("ENCRYPTION_KEYS", "1:short")
	_, err := platform.LoadConfig()
	if err == nil {
		t.Fatal("want error")
	}
	for _, name := range []string{"DATABASE_URL", "LOG_LEVEL", "ENCRYPTION_KEYS"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("ENCRYPTION_KEYS", "1:"+key(1))
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("LOG_LEVEL", "")
	cfg, err := platform.LoadConfig()
	if err != nil || cfg.HTTPAddr != ":8080" || cfg.FilesDir != "data/files" || !cfg.RunJobs {
		t.Fatalf("cfg = %+v, err = %v", cfg, err)
	}
}

func TestLoadConfigProducts(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("ENCRYPTION_KEYS", "1:"+key(1))
	t.Setenv("PRODUCTS", " hrm, ,sales")
	cfg, err := platform.LoadConfig()
	if err != nil || strings.Join(cfg.Products, "|") != "hrm|sales" {
		t.Fatalf("Products = %q, err = %v", cfg.Products, err)
	}
}
