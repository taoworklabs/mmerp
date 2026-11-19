package purchase_test

import (
	"testing"

	"github.com/taoworklabs/mmerp/internal/modules/purchase"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

func TestPurchaseKindsDeclared(t *testing.T) {
	newFixture(t)
	newFixture(t) // a second service declares nothing twice
	want := map[posting.Kind]bool{purchase.KindPayable: true, purchase.KindGoods: false, purchase.KindExpense: false,
		purchase.KindInputVAT: false, purchase.KindCashPayment: false, purchase.KindBankPayment: false}
	for k, byPartner := range want {
		d, ok := posting.Declared(k)
		if !ok || d.ByPartner != byPartner {
			t.Errorf("%s: declared %v, by partner %v; want by partner %v", k, ok, d.ByPartner, byPartner)
		}
	}
}
