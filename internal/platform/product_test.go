package platform_test

import (
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/platform"
)

func TestProductGate(t *testing.T) {
	ctx := platform.WithProducts(t.Context(), []string{"hrm"})
	for _, c := range []struct {
		product string
		class   platform.Class
		allowed bool
	}{
		{"hrm", platform.ClassWrite, true},
		{"", platform.ClassWrite, true},
		{"sales", platform.ClassRead, true},
		{"sales", platform.ClassExport, true},
		{"sales", platform.ClassWrite, false},
	} {
		err := platform.ProductGate(ctx, c.product, c.class)
		if (err == nil) != c.allowed {
			t.Errorf("ProductGate(%q, %d) = %v", c.product, c.class, err)
		}
		if e, ok := errors.AsType[*platform.Error](err); err != nil && (!ok || e.Code != "product_not_enabled" || e.Status != 403) {
			t.Errorf("ProductGate(%q, %d) error = %#v", c.product, c.class, err)
		}
	}
}
