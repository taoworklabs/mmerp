package sales_test

import (
	"testing"

	"github.com/taoworklabs/mmerp/internal/modules/sales"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// A "%" in the search matches a literal percent sign, not everything: the
// engine escapes LIKE metacharacters.
func TestCustomerListSearchLiteralPercent(t *testing.T) {
	f := newFixture(t)
	l, err := f.sales.Customers(f.admin, sales.CustomerFilter{Q: "%", Sort: "code", Paging: platform.Paging{Page: 1, PageSize: 20}})
	if err != nil {
		t.Fatal(err)
	}
	if l.Total != 0 || len(l.Items) != 0 {
		t.Fatalf("q %q matched %d, want 0", "%", l.Total)
	}
}
