package sales

import (
	"sync"

	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

// Business line kinds of invoices, receipts and deliveries; amounts are positive and
// accounting's rules give the side, so a receipt writes the receivable negative.
const (
	KindReceivable  posting.Kind = "sales_receivable"
	KindRevenue     posting.Kind = "sales_revenue"
	KindOutputVAT   posting.Kind = "sales_output_vat"
	KindCashReceipt posting.Kind = "sales_cash_receipt"
	KindBankReceipt posting.Kind = "sales_bank_receipt"
	// KindCostOfGoodsSold: the value a delivery takes out of stock.
	KindCostOfGoodsSold posting.Kind = "sales_cost_of_goods_sold"
)

// declareKinds runs once per process: tests build many services.
var declareKinds = sync.OnceFunc(func() {
	posting.DeclareKind(KindReceivable, true)
	for _, k := range []posting.Kind{KindRevenue, KindOutputVAT, KindCashReceipt, KindBankReceipt, KindCostOfGoodsSold} {
		posting.DeclareKind(k, false)
	}
})
