package purchase

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"sync"

	"github.com/taoworklabs/mmerp/internal/core/printing"
	"github.com/taoworklabs/mmerp/internal/modules/purchase/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

//go:embed i18n/*.json
var i18nFiles embed.FS

// loadTranslations adds the print labels and default text blocks to the catalog, once.
var loadTranslations = sync.OnceValue(func() error {
	sub, err := fs.Sub(i18nFiles, "i18n")
	if err != nil {
		return err
	}
	return platform.LoadTranslations(sub)
})

func (s *Service) registerPrints() {
	footer := printing.Block{Key: "footer", Label: "purchase.print.block.footer", Default: "purchase.print.block.footer.default"}
	s.d.Printing.Register(printing.Template{Code: orderType, Name: "purchase.print.order.name", DocType: orderType,
		Blocks: []printing.Block{
			{Key: "terms", Label: "purchase.print.block.terms", Default: "purchase.print.order.block.terms.default",
				Placeholders: []string{"supplier_name", "buyer_name"}},
			footer,
		},
		Data: s.orderPrintData, Layouts: []printing.Layout{orderLayout1}})
	s.d.Printing.Register(printing.Template{Code: goodsReceiptType, Name: "purchase.print.goods_receipt.name", DocType: goodsReceiptType,
		Blocks: []printing.Block{{Key: "footer", Label: "purchase.print.block.footer", Default: "purchase.print.goods_receipt.block.footer.default"}},
		Data:   s.receiptPrintData, Layouts: []printing.Layout{receiptLayout1}})
	s.d.Printing.Register(printing.Template{Code: invoiceType, Name: "purchase.print.invoice.name", DocType: invoiceType,
		Blocks: []printing.Block{footer}, Data: s.invoicePrintData, Layouts: []printing.Layout{invoiceLayout1}})
	s.d.Printing.Register(printing.Template{Code: paymentType, Name: "purchase.print.payment.name", DocType: paymentType,
		Blocks: []printing.Block{{Key: "footer", Label: "purchase.print.block.footer", Default: "purchase.print.payment.block.footer.default"}},
		Data:   s.paymentPrintData, Layouts: []printing.Layout{paymentLayout1}})
}

// printLine is one priced line as printed.
type printLine struct {
	Description     string `json:"description"`
	Unit            string `json:"unit"`
	Quantity        string `json:"quantity"`
	UnitPrice       int64  `json:"unit_price"`
	DiscountPercent string `json:"discount_percent"`
	VatRate         string `json:"vat_rate"`
	Net             int64  `json:"net"`
}

// orderPrint is what an order or an invoice prints; frozen as is once it is posted.
type orderPrint struct {
	Number       string               `json:"number"`
	Date         string               `json:"date"`
	OrderNumber  *string              `json:"order_number"`
	SupplierRef  *string              `json:"supplier_ref"`
	Buyer        printing.LegalEntity `json:"buyer"`
	Supplier     SupplierCopy         `json:"supplier"`
	ExpectedDate *string              `json:"expected_date"`
	PaymentTerms *string              `json:"payment_terms"`
	Note         *string              `json:"note"`
	Lines        []printLine          `json:"lines"`
	Totals       Totals               `json:"totals"`
}

// receiptPrint is what a goods receipt prints.
type receiptPrint struct {
	Number      string               `json:"number"`
	Date        string               `json:"date"`
	OrderNumber string               `json:"order_number"`
	Buyer       printing.LegalEntity `json:"buyer"`
	Supplier    string               `json:"supplier"`
	Warehouse   string               `json:"warehouse"`
	Note        *string              `json:"note"`
	Lines       []ReceiptLine        `json:"lines"`
}

// paymentPrint is what a payment prints.
type paymentPrint struct {
	Number      string               `json:"number"`
	Date        string               `json:"date"`
	Buyer       printing.LegalEntity `json:"buyer"`
	Supplier    string               `json:"supplier"`
	Method      string               `json:"method"`
	Amount      int64                `json:"amount"`
	Note        *string              `json:"note"`
	Allocations []Allocation         `json:"allocations"`
}

func (s *Service) orderPrintData(ctx context.Context, id int64) ([]printing.Part, error) {
	h, err := store.GetOrder.Run(ctx, id)
	if err != nil {
		return nil, err
	}
	lines, err := store.GetOrderLines.Run(ctx, id)
	if err != nil {
		return nil, err
	}
	buyer, err := s.d.Printing.LegalEntity(ctx, h.LegalEntityID)
	if err != nil {
		return nil, err
	}
	p := orderPrint{Number: h.Number, Date: *platform.DatePtr(h.Date), Buyer: buyer, Supplier: supplierCopy(h),
		ExpectedDate: platform.DatePtr(h.ExpectedDate), PaymentTerms: platform.TextPtr(h.PaymentTerms), Note: platform.TextPtr(h.Note),
		Lines:  make([]printLine, len(lines)),
		Totals: Totals{Subtotal: h.Subtotal, DiscountTotal: h.DiscountTotal, VatTotal: h.VatTotal, Total: h.Total}}
	for i, l := range lines {
		p.Lines[i] = printLine{Description: l.Description, Unit: l.Unit, Quantity: platform.PlainDecimal(l.Quantity), UnitPrice: l.UnitPrice,
			DiscountPercent: platform.PlainDecimal(l.DiscountPercent), VatRate: l.VatRate, Net: l.Amount - l.Discount}
	}
	b, err := json.Marshal(p)
	return []printing.Part{{Data: b}}, err
}

func (s *Service) invoicePrintData(ctx context.Context, id int64) ([]printing.Part, error) {
	h, err := store.GetInvoice.Run(ctx, id)
	if err != nil {
		return nil, err
	}
	o, err := store.GetOrder.Run(ctx, h.OrderID)
	if err != nil {
		return nil, err
	}
	lines, err := store.GetInvoiceLines.Run(ctx, id)
	if err != nil {
		return nil, err
	}
	buyer, err := s.d.Printing.LegalEntity(ctx, o.LegalEntityID)
	if err != nil {
		return nil, err
	}
	ref := h.SupplierInvoiceSymbol.String + " " + h.SupplierInvoiceNo.String
	p := orderPrint{Number: h.Number, Date: *platform.DatePtr(h.Date), OrderNumber: &h.OrderNumber, SupplierRef: &ref, Buyer: buyer,
		Supplier: supplierCopy(o), Note: platform.TextPtr(h.Note), Lines: make([]printLine, len(lines)),
		Totals: Totals{Subtotal: h.Subtotal, DiscountTotal: h.DiscountTotal, VatTotal: h.VatTotal, Total: h.Total}}
	for i, l := range lines {
		p.Lines[i] = printLine{Description: l.Description, Unit: l.Unit, Quantity: platform.PlainDecimal(l.Quantity), UnitPrice: l.UnitPrice,
			DiscountPercent: platform.PlainDecimal(l.DiscountPercent), VatRate: l.VatRate, Net: l.Amount - l.Discount}
	}
	b, err := json.Marshal(p)
	return []printing.Part{{Data: b}}, err
}

func (s *Service) receiptPrintData(ctx context.Context, id int64) ([]printing.Part, error) {
	h, err := store.GetGoodsReceipt.Run(ctx, id)
	if err != nil {
		return nil, err
	}
	o, err := store.GetOrder.Run(ctx, h.OrderID)
	if err != nil {
		return nil, err
	}
	rows, err := store.GetGoodsReceiptLines.Run(ctx, id)
	if err != nil {
		return nil, err
	}
	buyer, err := s.d.Printing.LegalEntity(ctx, o.LegalEntityID)
	if err != nil {
		return nil, err
	}
	p := receiptPrint{Number: h.Number, Date: *platform.DatePtr(h.Date), OrderNumber: h.OrderNumber, Buyer: buyer, Supplier: o.SupplierName,
		Warehouse: h.WarehouseCode + " · " + h.WarehouseName, Note: platform.TextPtr(h.Note), Lines: make([]ReceiptLine, len(rows))}
	for i, r := range rows {
		p.Lines[i] = ReceiptLine{ReceiptLineInput: receiptLineInput(r), ItemCode: r.ItemCode, Description: r.Description, Unit: r.Unit}
	}
	b, err := json.Marshal(p)
	return []printing.Part{{Data: b}}, err
}

func (s *Service) paymentPrintData(ctx context.Context, id int64) ([]printing.Part, error) {
	h, err := store.GetPayment.Run(ctx, id)
	if err != nil {
		return nil, err
	}
	rows, err := store.GetAllocations.Run(ctx, id)
	if err != nil {
		return nil, err
	}
	le, err := s.d.IAM.LegalEntityOf(ctx, h.OrgUnitID)
	if err != nil {
		return nil, err
	}
	buyer, err := s.d.Printing.LegalEntity(ctx, le)
	if err != nil {
		return nil, err
	}
	p := paymentPrint{Number: h.Number, Date: *platform.DatePtr(h.Date), Buyer: buyer, Supplier: h.SupplierName, Method: h.Method,
		Amount: h.Amount, Note: platform.TextPtr(h.Note), Allocations: make([]Allocation, len(rows))}
	for i, r := range rows {
		p.Allocations[i] = Allocation{AllocationInput: AllocationInput{InvoiceID: r.InvoiceID, Amount: r.Amount},
			InvoiceNumber: r.InvoiceNumber, InvoiceDate: *platform.DatePtr(r.InvoiceDate), InvoiceTotal: r.InvoiceTotal}
	}
	b, err := json.Marshal(p)
	return []printing.Part{{Data: b}}, err
}

func text(p *printing.Page) (func(string) string, func(*string) string) {
	return func(k string) string { return p.T("purchase.print."+k, nil) }, func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
}

// pricedLayout prints an order or an invoice: buyer, supplier, lines, totals.
func pricedLayout(p *printing.Page, d orderPrint, title string) {
	t, or := text(p)
	p.Title(t(title))
	p.Center(p.T("purchase.print.number", map[string]any{"number": d.Number, "date": p.Date(d.Date)}))
	if d.OrderNumber != nil {
		p.Center(p.T("purchase.print.from_order", map[string]any{"number": *d.OrderNumber}))
	}
	p.LegalEntity(t("buyer"), t("name"), d.Buyer)
	p.Heading(t("supplier"))
	p.Field(t("name"), d.Supplier.Name)
	p.Field(t("tax_code"), or(d.Supplier.TaxCode))
	p.Field(t("address"), or(d.Supplier.Address))
	if d.SupplierRef != nil {
		p.Field(t("supplier_ref"), *d.SupplierRef)
	}
	cols := []printing.Column{{Title: t("description"), Share: 0.34}, {Title: t("unit"), Share: 0.08}, {Title: t("quantity"), Share: 0.1, Right: true},
		{Title: t("unit_price"), Share: 0.14, Right: true}, {Title: t("discount"), Share: 0.08, Right: true}, {Title: t("vat_rate"), Share: 0.08, Right: true},
		{Title: t("net"), Share: 0.18, Right: true}}
	rows := make([][]string, len(d.Lines))
	for i, l := range d.Lines {
		vat := l.VatRate + "%"
		if l.VatRate == "none" {
			vat = t("vat_none")
		}
		rows[i] = []string{l.Description, l.Unit, p.Decimal(l.Quantity), p.Money(l.UnitPrice), p.Decimal(l.DiscountPercent) + "%", vat, p.Money(l.Net)}
	}
	p.Heading(t("lines"))
	p.Table(cols, rows, false)
	p.Table([]printing.Column{{Share: 0.7}, {Share: 0.3, Right: true}}, [][]string{
		{t("subtotal"), p.Money(d.Totals.Subtotal)},
		{t("discount_total"), p.Money(d.Totals.DiscountTotal)},
		{t("vat_total"), p.Money(d.Totals.VatTotal)},
		{t("total"), p.Money(d.Totals.Total)},
	}, true)
	p.Text(t("currency"))
	if d.ExpectedDate != nil {
		p.Field(t("expected_date"), p.Date(*d.ExpectedDate))
	}
	if d.PaymentTerms != nil {
		p.Field(t("payment_terms"), *d.PaymentTerms)
	}
	if d.Note != nil {
		p.Field(t("note"), *d.Note)
	}
	p.Gap()
}

func orderLayout1(p *printing.Page, raw json.RawMessage) error {
	var d orderPrint
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	pricedLayout(p, d, "order.title")
	p.Block("terms", map[string]string{"supplier_name": d.Supplier.Name, "buyer_name": d.Buyer.Name})
	p.Gap()
	p.Block("footer", nil)
	return nil
}

func invoiceLayout1(p *printing.Page, raw json.RawMessage) error {
	var d orderPrint
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	pricedLayout(p, d, "invoice.title")
	p.Block("footer", nil)
	return nil
}

// receiptLayout1 prints a goods receipt for the storekeeper and the deliverer to sign.
func receiptLayout1(p *printing.Page, raw json.RawMessage) error {
	var d receiptPrint
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	t, _ := text(p)
	p.Title(t("goods_receipt.title"))
	p.Center(p.T("purchase.print.number", map[string]any{"number": d.Number, "date": p.Date(d.Date)}))
	p.Center(p.T("purchase.print.from_order", map[string]any{"number": d.OrderNumber}))
	p.LegalEntity(t("buyer"), t("name"), d.Buyer)
	p.Field(t("supplier"), d.Supplier)
	p.Field(t("warehouse"), d.Warehouse)
	if d.Note != nil {
		p.Field(t("note"), *d.Note)
	}
	rows := make([][]string, len(d.Lines))
	for i, l := range d.Lines {
		track := ""
		switch {
		case l.Lot != nil && l.Lot.Expiry != nil:
			track = l.Lot.Code + " · " + p.Date(*l.Lot.Expiry)
		case l.Lot != nil:
			track = l.Lot.Code
		case l.Serial != nil:
			track = *l.Serial
		}
		rows[i] = []string{l.ItemCode, l.Description, l.Unit, track, p.Decimal(l.Quantity)}
	}
	p.Heading(t("lines"))
	p.Table([]printing.Column{{Title: t("item_code"), Share: 0.14}, {Title: t("description"), Share: 0.38}, {Title: t("unit"), Share: 0.1},
		{Title: t("lot_serial"), Share: 0.24}, {Title: t("quantity"), Share: 0.14, Right: true}}, rows, false)
	p.Gap()
	p.Block("footer", nil)
	return nil
}

// paymentLayout1 prints a payment: to whom, how much, how, and which invoices it settles.
func paymentLayout1(p *printing.Page, raw json.RawMessage) error {
	var d paymentPrint
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	t, _ := text(p)
	p.Title(t("payment.title"))
	p.Center(p.T("purchase.print.number", map[string]any{"number": d.Number, "date": p.Date(d.Date)}))
	p.LegalEntity(t("buyer"), t("name"), d.Buyer)
	p.Field(t("payment.payee"), d.Supplier)
	p.Field(t("payment.method"), t("payment.method."+d.Method))
	p.Field(t("payment.amount"), p.Money(d.Amount))
	if d.Note != nil {
		p.Field(t("note"), *d.Note)
	}
	rows := make([][]string, len(d.Allocations))
	for i, a := range d.Allocations {
		rows[i] = []string{a.InvoiceNumber, p.Date(a.InvoiceDate), p.Money(a.InvoiceTotal), p.Money(a.Amount)}
	}
	p.Heading(t("payment.allocations"))
	p.Table([]printing.Column{{Title: t("payment.invoice"), Share: 0.3}, {Title: t("payment.invoice_date"), Share: 0.2},
		{Title: t("payment.invoice_total"), Share: 0.25, Right: true}, {Title: t("payment.settled"), Share: 0.25, Right: true}}, rows, false)
	p.Text(t("currency"))
	p.Gap()
	p.Block("footer", nil)
	return nil
}
