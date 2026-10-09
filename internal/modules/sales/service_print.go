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
		s.d.Printing.Register(printing.Template{Code: k.docType, Name: "sales.print." + k.name + ".name", DocType: k.docType, Product: "sales",
			Blocks: []printing.Block{
				{Key: "terms", Label: "sales.print.block.terms", Default: "sales.print." + k.name + ".block.terms.default",
					Placeholders: []string{"customer_name", "seller_name"}},
				{Key: "footer", Label: "sales.print.block.footer", Default: "sales.print.block.footer.default"},
			},
			Data: s.printData, Layouts: []printing.Layout{docLayout1}})
	}
}

type sellerPrint struct {
	Name    string  `json:"name"`
	TaxCode *string `json:"tax_code"`
	Address *string `json:"address"`
}

// docPrint is what a quotation or order prints; frozen as is once it is posted.
type docPrint struct {
	Kind          string       `json:"kind"`
	Number        string       `json:"number"`
	Date          string       `json:"date"`
	QuoteNumber   *string      `json:"quote_number"`
	Seller        sellerPrint  `json:"seller"`
	Customer      CustomerCopy `json:"customer"`
	ValidUntil    *string      `json:"valid_until"`
	DeliveryDate  *string      `json:"delivery_date"`
	PaymentTerms  *string      `json:"payment_terms"`
	DeliveryTerms *string      `json:"delivery_terms"`
	Note          *string      `json:"note"`
	Lines         []Line       `json:"lines"`
	Totals        Totals       `json:"totals"`
}

// printData reads a quotation or order as printed; printing checked the actor may print it.
func (s *Service) printData(ctx context.Context, id int64) ([]printing.Part, error) {
	q := store.New(platform.DBFrom(ctx))
	h, err := q.GetHeader(ctx, id)
	if err != nil {
		return nil, err
	}
	lines, err := q.GetLines(ctx, id)
	if err != nil {
		return nil, err
	}
	le, err := q.PrintLegalEntity(ctx, h.LegalEntityID)
	if err != nil {
		return nil, err
	}
	p := docPrint{Kind: h.Kind, Number: h.Number, Date: *platform.DatePtr(h.Date), QuoteNumber: platform.TextPtr(h.QuoteNumber),
		Seller:   sellerPrint{Name: le.Name, TaxCode: platform.TextPtr(le.TaxCode), Address: platform.TextPtr(le.Address)},
		Customer: customerCopy(h), ValidUntil: platform.DatePtr(h.ValidUntil), DeliveryDate: platform.DatePtr(h.DeliveryDate),
		PaymentTerms: platform.TextPtr(h.PaymentTerms), DeliveryTerms: platform.TextPtr(h.DeliveryTerms), Note: platform.TextPtr(h.Note),
		Lines:  make([]Line, len(lines)),
		Totals: Totals{Subtotal: h.Subtotal, DiscountTotal: h.DiscountTotal, VatTotal: h.VatTotal, Total: h.Total}}
	for i, l := range lines {
		p.Lines[i] = Line{LineInput: LineInput{ItemID: l.ItemID, Description: l.Description, Quantity: plain(l.Quantity), UnitPrice: l.UnitPrice,
			DiscountPercent: plain(l.DiscountPercent), VatRate: l.VatRate}, ItemCode: l.ItemCode, Unit: l.Unit, Amount: l.Amount, Discount: l.Discount, Vat: l.Vat}
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
	p.Heading(t("seller"))
	p.Field(t("name"), d.Seller.Name)
	p.Field(t("tax_code"), or(d.Seller.TaxCode))
	p.Field(t("address"), or(d.Seller.Address))
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
