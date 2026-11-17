package sales_test

import (
	"testing"

	"github.com/taoworklabs/mmerp/internal/modules/sales"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

func TestSalesKindsDeclared(t *testing.T) {
	newFixture(t)
	newFixture(t) // a second service declares nothing twice
	want := map[posting.Kind]bool{sales.KindReceivable: true, sales.KindRevenue: false, sales.KindOutputVAT: false,
		sales.KindCashReceipt: false, sales.KindBankReceipt: false}
	for k, byPartner := range want {
		d, ok := posting.Declared(k)
		if !ok || d.ByPartner != byPartner {
			t.Errorf("%s: declared %v, by partner %v; want by partner %v", k, ok, d.ByPartner, byPartner)
		}
	}
}
