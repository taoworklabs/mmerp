package purchase

import (
	"sync"

	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

// Business line kinds of goods receipts, supplier invoices and payments; amounts are positive
// and accounting's rules give the side, so a payment writes the payable negative.
const (
	// KindGoods is no longer written: invoices clear KindReceivedNotInvoiced instead. Kept for
	// invoices posted before.
	KindGoods       posting.Kind = "purchase_goods"
	KindExpense     posting.Kind = "purchase_expense"
	KindInputVAT    posting.Kind = "purchase_input_vat"
	KindPayable     posting.Kind = "purchase_payable"
	KindCashPayment posting.Kind = "purchase_cash_payment"
	KindBankPayment posting.Kind = "purchase_bank_payment"
	// KindReceivedNotInvoiced: goods received at the order price, cleared by the invoice.
	KindReceivedNotInvoiced posting.Kind = "purchase_received_not_invoiced"
	// KindPriceDifference: an invoice's difference from the order price for goods no longer on hand.
	KindPriceDifference posting.Kind = "purchase_price_difference"
)

// declareKinds runs once per process: tests build many services.
var declareKinds = sync.OnceFunc(func() {
	posting.DeclareKind(KindPayable, true)
	for _, k := range []posting.Kind{KindGoods, KindExpense, KindInputVAT, KindCashPayment, KindBankPayment, KindReceivedNotInvoiced, KindPriceDifference} {
		posting.DeclareKind(k, false)
	}
})
