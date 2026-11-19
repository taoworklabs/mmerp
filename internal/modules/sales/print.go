package sales

import (
	"context"
	"encoding/json"

	"github.com/taoworklabs/mmerp/internal/core/printing"
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func (s *Service) registerPrints() {
	for _, k := range []kind{quoteKind, orderKind} {
		s.d.Printing.Register(printing.Template{Code: k.docType, Name: "sales.print." + k.name + ".name", DocType: k.docType,
			Blocks: []printing.Block{
				{Key: "terms", Label: "sales.print.block.terms", Default: "sales.print." + k.name + ".block.terms.default",
					Placeholders: []string{"customer_name", "seller_name"}},
				{Key: "footer", Label: "sales.print.block.footer", Default: "sales.print.block.footer.default"},
			},
			Data: s.printData, Layouts: []printing.Layout{docLayout1}})
	}
}

// docPrint is what a quotation or order prints; frozen as is once it is posted.
type docPrint struct {
	Kind          string               `json:"kind"`
	Number        string               `json:"number"`
	Date          string               `json:"date"`
	QuoteNumber   *string              `json:"quote_number"`
	Seller        printing.LegalEntity `json:"seller"`
	Customer      CustomerCopy         `json:"customer"`
	ValidUntil    *string              `json:"valid_until"`
	DeliveryDate  *string              `json:"delivery_date"`
	PaymentTerms  *string              `json:"payment_terms"`
	DeliveryTerms *string              `json:"delivery_terms"`
	Note          *string              `json:"note"`
	Lines         []Line               `json:"lines"`
	Totals        Totals               `json:"totals"`
}

// printData reads a quotation or order as printed; printing checked the actor may print it.
func (s *Service) printData(ctx context.Context, id int64) ([]printing.Part, error) {
	h, err := store.GetHeader.Run(ctx, id)
	if err != nil {
		return nil, err
	}
	lines, err := store.GetLines.Run(ctx, id)
	if err != nil {
		return nil, err
	}
	seller, err := s.d.Printing.LegalEntity(ctx, h.LegalEntityID)
	if err != nil {
		return nil, err
	}
	p := docPrint{Kind: h.Kind, Number: h.Number, Date: *platform.DatePtr(h.Date), QuoteNumber: platform.TextPtr(h.QuoteNumber),
		Seller:   seller,
		Customer: customerCopy(h), ValidUntil: platform.DatePtr(h.ValidUntil), DeliveryDate: platform.DatePtr(h.DeliveryDate),
		PaymentTerms: platform.TextPtr(h.PaymentTerms), DeliveryTerms: platform.TextPtr(h.DeliveryTerms), Note: platform.TextPtr(h.Note),
		Lines:  make([]Line, len(lines)),
		Totals: Totals{Subtotal: h.Subtotal, DiscountTotal: h.DiscountTotal, VatTotal: h.VatTotal, Total: h.Total}}
	for i, l := range lines {
		p.Lines[i] = Line{LineInput: LineInput{ItemID: l.ItemID, Description: l.Description, Quantity: platform.PlainDecimal(l.Quantity), UnitPrice: l.UnitPrice,
			DiscountPercent: platform.PlainDecimal(l.DiscountPercent), VatRate: l.VatRate}, ItemCode: l.ItemCode, Unit: l.Unit, Amount: l.Amount, Discount: l.Discount, Vat: l.Vat}
	}
	b, err := json.Marshal(p)
	return []printing.Part{{Data: b}}, err
}

// docLayout1 prints a quotation or order: seller, customer, lines, totals, terms.
func docLayout1(p *printing.Page, raw json.RawMessage) error {
	var d docPrint
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	t := func(k string) string { return p.T("sales.print."+k, nil) }
	or := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	p.Title(t(d.Kind + ".title"))
	p.Center(p.T("sales.print.number", map[string]any{"number": d.Number, "date": p.Date(d.Date)}))
	if d.QuoteNumber != nil {
		p.Center(p.T("sales.print.order.from_quote", map[string]any{"number": *d.QuoteNumber}))
	}
	p.LegalEntity(t("seller"), t("name"), d.Seller)
	p.Heading(t("customer"))
	p.Field(t("name"), d.Customer.Name)
	p.Field(t("tax_code"), or(d.Customer.TaxCode))
	p.Field(t("address"), or(d.Customer.Address))
	p.Field(t("contact"), or(d.Customer.ContactName))
	p.Field(t("phone"), or(d.Customer.Phone))
	cols := []printing.Column{{Title: t("description"), Share: 0.34}, {Title: t("unit"), Share: 0.08}, {Title: t("quantity"), Share: 0.1, Right: true},
		{Title: t("unit_price"), Share: 0.14, Right: true}, {Title: t("discount"), Share: 0.08, Right: true}, {Title: t("vat_rate"), Share: 0.08, Right: true},
		{Title: t("net"), Share: 0.18, Right: true}}
	rows := make([][]string, len(d.Lines))
	for i, l := range d.Lines {
		rows[i] = []string{l.Description, l.Unit, p.Decimal(l.Quantity), p.Money(l.UnitPrice), p.Decimal(l.DiscountPercent) + "%",
			vatLabel(p, l.VatRate), p.Money(l.Amount - l.Discount)}
	}
	p.Heading(t("lines"))
	p.Table(cols, rows, false)
	total := []printing.Column{{Share: 0.7}, {Share: 0.3, Right: true}}
	p.Table(total, [][]string{
		{t("subtotal"), p.Money(d.Totals.Subtotal)},
		{t("discount_total"), p.Money(d.Totals.DiscountTotal)},
		{t("vat_total"), p.Money(d.Totals.VatTotal)},
		{t("total"), p.Money(d.Totals.Total)},
	}, true)
	p.Text(t("currency"))
	p.Heading(t("terms"))
	if d.ValidUntil != nil {
		p.Field(t("valid_until"), p.Date(*d.ValidUntil))
	}
	if d.DeliveryDate != nil {
		p.Field(t("delivery_date"), p.Date(*d.DeliveryDate))
	}
	p.Field(t("payment_terms"), or(d.PaymentTerms))
	p.Field(t("delivery_terms"), or(d.DeliveryTerms))
	if d.Note != nil {
		p.Field(t("note"), *d.Note)
	}
	p.Gap()
	p.Block("terms", map[string]string{"customer_name": d.Customer.Name, "seller_name": d.Seller.Name})
	p.Gap()
	p.Block("footer", nil)
	return nil
}

func vatLabel(p *printing.Page, rate string) string {
	if rate == "none" {
		return p.T("sales.print.vat_none", nil)
	}
	return rate + "%"
}

// invoicePrint is what an invoice prints; frozen as is once it is posted.
type invoicePrint struct {
	Number      string               `json:"number"`
	Date        string               `json:"date"`
	OrderNumber string               `json:"order_number"`
	Seller      printing.LegalEntity `json:"seller"`
	Customer    CustomerCopy         `json:"customer"`
	Note        *string              `json:"note"`
	Lines       []InvoiceLine        `json:"lines"`
	Totals      Totals               `json:"totals"`
}

// receiptPrint is what a receipt prints; frozen as is once it is posted.
type receiptPrint struct {
	Number       string               `json:"number"`
	Date         string               `json:"date"`
	Seller       printing.LegalEntity `json:"seller"`
	CustomerName string               `json:"customer_name"`
	Method       string               `json:"method"`
	Amount       int64                `json:"amount"`
	Note         *string              `json:"note"`
	Allocations  []Allocation         `json:"allocations"`
}

func (s *Service) registerPaymentPrints() {
	s.d.Printing.Register(printing.Template{Code: invoiceType, Name: "sales.print.invoice.name", DocType: invoiceType,
		Blocks: []printing.Block{
			{Key: "terms", Label: "sales.print.block.terms", Default: "sales.print.invoice.block.terms.default",
				Placeholders: []string{"customer_name", "seller_name"}},
			{Key: "footer", Label: "sales.print.block.footer", Default: "sales.print.block.footer.default"},
		},
		Data: s.invoicePrintData, Layouts: []printing.Layout{invoiceLayout1}})
	s.d.Printing.Register(printing.Template{Code: receiptType, Name: "sales.print.receipt.name", DocType: receiptType,
		Blocks: []printing.Block{{Key: "footer", Label: "sales.print.block.footer", Default: "sales.print.receipt.block.footer.default"}},
		Data:   s.receiptPrintData, Layouts: []printing.Layout{receiptLayout1}})
}

// invoicePrintData reads an invoice as printed; printing checked the actor may print it.
func (s *Service) invoicePrintData(ctx context.Context, id int64) ([]printing.Part, error) {
	h, err := store.GetInvoice.Run(ctx, id)
	if err != nil {
		return nil, err
	}
	o, err := store.GetHeader.Run(ctx, h.OrderID)
	if err != nil {
		return nil, err
	}
	lines, err := store.GetInvoiceLines.Run(ctx, id)
	if err != nil {
		return nil, err
	}
	seller, err := s.d.Printing.LegalEntity(ctx, o.LegalEntityID)
	if err != nil {
		return nil, err
	}
	p := invoicePrint{Number: h.Number, Date: *platform.DatePtr(h.Date), OrderNumber: h.OrderNumber, Seller: seller,
		Customer: customerCopy(o), Note: platform.TextPtr(h.Note), Lines: make([]InvoiceLine, len(lines)),
		Totals: Totals{Subtotal: h.Subtotal, DiscountTotal: h.DiscountTotal, VatTotal: h.VatTotal, Total: h.Total}}
	for i, l := range lines {
		p.Lines[i] = InvoiceLine{InvoiceLineInput: InvoiceLineInput{Quantity: platform.PlainDecimal(l.Quantity)}, Description: l.Description,
			Unit: l.Unit, UnitPrice: l.UnitPrice, DiscountPercent: platform.PlainDecimal(l.DiscountPercent), VatRate: l.VatRate,
			Amount: l.Amount, Discount: l.Discount, Vat: l.Vat}
	}
	b, err := json.Marshal(p)
	return []printing.Part{{Data: b}}, err
}

// receiptPrintData reads a receipt as printed; printing checked the actor may print it.
func (s *Service) receiptPrintData(ctx context.Context, id int64) ([]printing.Part, error) {
	h, err := store.GetReceipt.Run(ctx, id)
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
	seller, err := s.d.Printing.LegalEntity(ctx, le)
	if err != nil {
		return nil, err
	}
	p := receiptPrint{Number: h.Number, Date: *platform.DatePtr(h.Date), Seller: seller, CustomerName: h.CustomerName, Method: h.Method,
		Amount: h.Amount, Note: platform.TextPtr(h.Note), Allocations: make([]Allocation, len(rows))}
	for i, r := range rows {
		p.Allocations[i] = Allocation{AllocationInput: AllocationInput{InvoiceID: r.InvoiceID, Amount: r.Amount},
			InvoiceNumber: r.InvoiceNumber, InvoiceDate: *platform.DatePtr(r.InvoiceDate), InvoiceTotal: r.InvoiceTotal}
	}
	b, err := json.Marshal(p)
	return []printing.Part{{Data: b}}, err
}

// invoiceLayout1 prints an invoice: seller, customer, lines, totals; an internal print, not an e-invoice.
func invoiceLayout1(p *printing.Page, raw json.RawMessage) error {
	var d invoicePrint
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	t := func(k string) string { return p.T("sales.print."+k, nil) }
	or := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	p.Title(t("invoice.title"))
	p.Center(p.T("sales.print.number", map[string]any{"number": d.Number, "date": p.Date(d.Date)}))
	p.Center(p.T("sales.print.invoice.from_order", map[string]any{"number": d.OrderNumber}))
	p.Center(t("invoice.internal"))
	p.LegalEntity(t("seller"), t("name"), d.Seller)
	p.Heading(t("customer"))
	p.Field(t("name"), d.Customer.Name)
	p.Field(t("tax_code"), or(d.Customer.TaxCode))
	p.Field(t("address"), or(d.Customer.Address))
	cols := []printing.Column{{Title: t("description"), Share: 0.34}, {Title: t("unit"), Share: 0.08}, {Title: t("quantity"), Share: 0.1, Right: true},
		{Title: t("unit_price"), Share: 0.14, Right: true}, {Title: t("discount"), Share: 0.08, Right: true}, {Title: t("vat_rate"), Share: 0.08, Right: true},
		{Title: t("net"), Share: 0.18, Right: true}}
	rows := make([][]string, len(d.Lines))
	for i, l := range d.Lines {
		rows[i] = []string{l.Description, l.Unit, p.Decimal(l.Quantity), p.Money(l.UnitPrice), p.Decimal(l.DiscountPercent) + "%",
			vatLabel(p, l.VatRate), p.Money(l.Amount - l.Discount)}
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
	if d.Note != nil {
		p.Field(t("note"), *d.Note)
	}
	p.Gap()
	p.Block("terms", map[string]string{"customer_name": d.Customer.Name, "seller_name": d.Seller.Name})
	p.Gap()
	p.Block("footer", nil)
	return nil
}

// receiptLayout1 prints a receipt: who paid, how much, how, and which invoices it settles.
func receiptLayout1(p *printing.Page, raw json.RawMessage) error {
	var d receiptPrint
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	t := func(k string) string { return p.T("sales.print."+k, nil) }
	p.Title(t("receipt.title"))
	p.Center(p.T("sales.print.number", map[string]any{"number": d.Number, "date": p.Date(d.Date)}))
	p.LegalEntity(t("seller"), t("name"), d.Seller)
	p.Field(t("receipt.payer"), d.CustomerName)
	p.Field(t("receipt.method"), t("receipt.method."+d.Method))
	p.Field(t("receipt.amount"), p.Money(d.Amount))
	if d.Note != nil {
		p.Field(t("note"), *d.Note)
	}
	rows := make([][]string, len(d.Allocations))
	for i, a := range d.Allocations {
		rows[i] = []string{a.InvoiceNumber, p.Date(a.InvoiceDate), p.Money(a.InvoiceTotal), p.Money(a.Amount)}
	}
	p.Heading(t("receipt.allocations"))
	p.Table([]printing.Column{{Title: t("receipt.invoice"), Share: 0.3}, {Title: t("receipt.invoice_date"), Share: 0.2},
		{Title: t("receipt.invoice_total"), Share: 0.25, Right: true}, {Title: t("receipt.settled"), Share: 0.25, Right: true}}, rows, false)
	p.Text(t("currency"))
	p.Gap()
	p.Block("footer", nil)
	return nil
}
