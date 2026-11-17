package sales

import (
	"sync"

	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

// Business line kinds of invoices and receipts; amounts are positive and accounting's rules
// give the side, so a receipt writes the receivable negative.
const (
	KindReceivable  posting.Kind = "sales_receivable"
	KindRevenue     posting.Kind = "sales_revenue"
	KindOutputVAT   posting.Kind = "sales_output_vat"
	KindCashReceipt posting.Kind = "sales_cash_receipt"
	KindBankReceipt posting.Kind = "sales_bank_receipt"
)

// declareKinds runs once per process: tests build many services.
var declareKinds = sync.OnceFunc(func() {
	posting.DeclareKind(KindReceivable, true)
	for _, k := range []posting.Kind{KindRevenue, KindOutputVAT, KindCashReceipt, KindBankReceipt} {
		posting.DeclareKind(k, false)
	}
})
