package purchase

import (
	"sync"

	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

// Business line kinds of supplier invoices and payments; amounts are positive and accounting's
// rules give the side, so a payment writes the payable negative.
const (
	KindGoods       posting.Kind = "purchase_goods"
	KindExpense     posting.Kind = "purchase_expense"
	KindInputVAT    posting.Kind = "purchase_input_vat"
	KindPayable     posting.Kind = "purchase_payable"
	KindCashPayment posting.Kind = "purchase_cash_payment"
	KindBankPayment posting.Kind = "purchase_bank_payment"
)

// declareKinds runs once per process: tests build many services.
var declareKinds = sync.OnceFunc(func() {
	posting.DeclareKind(KindPayable, true)
	for _, k := range []posting.Kind{KindGoods, KindExpense, KindInputVAT, KindCashPayment, KindBankPayment} {
		posting.DeclareKind(k, false)
	}
})
