package purchase

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/purchase/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

const invoiceType = "purchase.invoice"

// An invoice is as open as its order: same org unit, same permissions.
func (s *Service) registerInvoice() {
	s.d.Record.Register(record.Type{
		Code: invoiceType, Product: "purchase", Kind: record.Document, NumberPrefix: "HM",
		Perms:        record.Perms{View: PermOrderView, Edit: PermOrderEdit, ViewFiles: PermOrderView, Attach: PermOrderEdit},
		OnTransition: s.invoiceTransition,
		BeforeSubmit: s.invoiceBeforeSubmit,
		OnDelete:     func(ctx context.Context, id int64) error { return store.DeleteInvoice.Run(ctx, id) },
	})
}

// invoiceBeforeSubmit needs the supplier's invoice reference and refuses a bill already booked.
func (s *Service) invoiceBeforeSubmit(ctx context.Context, d record.Doc) error {
	h, err := store.GetInvoice.Run(ctx, d.ID)
	if err != nil {
		return err
	}
	return s.checkSupplierRef(ctx, d.ID, h)
}

// checkSupplierRef refuses a missing reference, then, under the supplier's row lock so two
// sends of the same paper take turns, another pending or posted invoice of the supplier with it.
func (s *Service) checkSupplierRef(ctx context.Context, id int64, h store.GetInvoiceRow) error {
	symbol, number := platform.TextPtr(h.SupplierInvoiceSymbol), platform.TextPtr(h.SupplierInvoiceNo)
	if trimmed(symbol) == nil || trimmed(number) == nil || !h.SupplierInvoiceDate.Valid {
		return ErrInvoiceSupplierRef
	}
	if err := store.LockSupplier.Run(ctx, h.SupplierID); err != nil {
		return err
	}
	other, err := store.DuplicateInvoice.Run(ctx, store.DuplicateInvoiceParams{SupplierID: h.SupplierID, InvoiceID: id, Symbol: *symbol, Number: *number})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return &platform.Error{Status: http.StatusConflict, Code: "purchase_invoice_duplicate", Params: map[string]any{"number": other}}
}

// invoiceTransition checks the lines against the order and the supplier reference again and
// writes the business lines on posting; cancelling is refused while a payment settles the
// invoice, then voids them.
func (s *Service) invoiceTransition(ctx context.Context, d record.Doc, _ record.Status) error {
	switch d.Status {
	case record.Posted:
		h, err := store.GetInvoice.Run(ctx, d.ID)
		if err != nil {
			return err
		}
		saved, err := store.GetInvoiceLines.Run(ctx, d.ID)
		if err != nil {
			return err
		}
		f := InvoiceFields{Date: d.Date, Lines: make([]InvoiceLineInput, len(saved))}
		for i, l := range saved {
			f.Lines[i] = InvoiceLineInput{OrderPosition: int(l.OrderPosition), Quantity: l.Quantity, UnitPrice: l.UnitPrice,
				DiscountPercent: l.DiscountPercent, VatRate: l.VatRate}
		}
		if _, _, err := s.checkInvoice(ctx, d.ID, h.OrderID, d.OrgUnitID, f); err != nil {
			return err
		}
		if err := s.checkSupplierRef(ctx, d.ID, h); err != nil {
			return err
		}
		// The amounts are those saved, which the approver saw.
		var goods, expense int64
		for _, l := range saved {
			if l.Stocked {
				goods += l.Amount - l.Discount
			} else {
				expense += l.Amount - l.Discount
			}
		}
		return s.d.Posting.Record(ctx, d.Ref, nonZero([]posting.Line{
			{Kind: KindGoods, OrgUnitID: d.OrgUnitID, Amount: goods},
			{Kind: KindExpense, OrgUnitID: d.OrgUnitID, Amount: expense},
			{Kind: KindInputVAT, OrgUnitID: d.OrgUnitID, Amount: h.VatTotal},
			{Kind: KindPayable, OrgUnitID: d.OrgUnitID, PartnerID: &h.PartnerID, Amount: h.Total},
		}))
	case record.Cancelled:
		if err := refuseIfLive(ctx, d.ID, store.LivePaymentOfInvoice.Run, ErrInvoiceHasPayment); err != nil {
			return err
		}
		return s.d.Posting.Void(ctx, d.Ref)
	}
	return nil
}

func nonZero(lines []posting.Line) []posting.Line {
	out := lines[:0]
	for _, l := range lines {
		if l.Amount != 0 {
			out = append(out, l)
		}
	}
	return out
}

// Invoice returns one invoice; one the actor may not view does not exist.
func (s *Service) Invoice(ctx context.Context, id int64) (Invoice, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: invoiceType, ID: id})
	if err != nil {
		return Invoice{}, err
	}
	h, err := store.GetInvoice.Run(ctx, id)
	if err != nil {
		return Invoice{}, err
	}
	o, err := store.GetOrder.Run(ctx, h.OrderID)
	if err != nil {
		return Invoice{}, err
	}
	lines, err := store.GetInvoiceLines.Run(ctx, id)
	if err != nil {
		return Invoice{}, err
	}
	out := Invoice{ID: id, OrderID: h.OrderID, OrderNumber: h.OrderNumber, Number: h.Number, Status: h.Status, Version: h.Version,
		Date: *platform.DatePtr(h.Date), OrgUnitID: h.OrgUnitID, OrgUnitName: h.OrgUnitName, Supplier: supplierCopy(o),
		SupplierInvoiceSymbol: platform.TextPtr(h.SupplierInvoiceSymbol), SupplierInvoiceNo: platform.TextPtr(h.SupplierInvoiceNo),
		SupplierInvoiceDate: platform.DatePtr(h.SupplierInvoiceDate), Note: platform.TextPtr(h.Note), Lines: make([]InvoiceLine, len(lines)),
		Totals: Totals{Subtotal: h.Subtotal, DiscountTotal: h.DiscountTotal, VatTotal: h.VatTotal, Total: h.Total}}
	for i, l := range lines {
		out.Lines[i] = InvoiceLine{
			InvoiceLineInput: InvoiceLineInput{OrderPosition: int(l.OrderPosition), Quantity: platform.PlainDecimal(l.Quantity), UnitPrice: l.UnitPrice,
				DiscountPercent: platform.PlainDecimal(l.DiscountPercent), VatRate: l.VatRate},
			ItemID: l.ItemID, ItemCode: l.ItemCode, Description: l.Description, Unit: l.Unit, Stocked: l.Stocked,
			Amount: l.Amount, Discount: l.Discount, Vat: l.Vat,
		}
	}
	out.AllowedActions, err = s.d.Record.Actions(ctx, d)
	return out, err
}

// InvoiceOptions lists the order's lines with what non-cancelled invoices leave of each and the
// order line's price, discount and VAT rate to pre-fill.
func (s *Service) InvoiceOptions(ctx context.Context, orderID int64) (InvoiceOptions, error) {
	if _, err := s.d.Record.Read(ctx, record.Ref{Type: orderType, ID: orderID}); err != nil {
		return InvoiceOptions{}, err
	}
	rows, err := store.InvoiceableLines.Run(ctx, store.InvoiceableLinesParams{OrderID: orderID})
	if err != nil {
		return InvoiceOptions{}, err
	}
	out := InvoiceOptions{Lines: make([]InvoiceOptionLine, len(rows))}
	for i, r := range rows {
		ordered, taken, err := decimals(r.Quantity, r.Taken)
		if err != nil {
			return InvoiceOptions{}, err
		}
		out.Lines[i] = InvoiceOptionLine{OrderPosition: int(r.Position), ItemID: r.ItemID, ItemCode: r.ItemCode, Description: r.Description,
			Unit: r.Unit, UnitPrice: r.UnitPrice, DiscountPercent: platform.PlainDecimal(r.DiscountPercent), VatRate: r.VatRate,
			Ordered: ordered.StringFixed(3), Remaining: decimal.Max(ordered.Sub(taken), decimal.Zero).StringFixed(3)}
	}
	return out, nil
}

// CreateInvoice adds a draft invoice of a posted order, at the order's org unit; the same
// request id returns the first one.
func (s *Service) CreateInvoice(ctx context.Context, in NewInvoice) (id int64, err error) {
	return id, platform.InTx(ctx, func(ctx context.Context) error {
		o, err := s.d.Record.Read(ctx, record.Ref{Type: orderType, ID: in.OrderID})
		if err != nil {
			return err
		}
		// The amounts come from the input alone, so pricing before the locks holds.
		_, t, err := s.priceInvoice(ctx, 0, in.OrderID, o.OrgUnitID, in.InvoiceFields, false)
		if err != nil {
			return err
		}
		d, err := s.d.Record.Create(ctx, invoiceType, record.Header{Date: in.Date, OrgUnitID: o.OrgUnitID, RequestID: in.RequestID, Amount: &t.Total})
		if id = d.ID; err != nil || d.Reused {
			return err
		}
		lines, t, err := s.checkInvoice(ctx, id, in.OrderID, o.OrgUnitID, in.InvoiceFields)
		if err != nil {
			return err
		}
		if err := store.InsertInvoice.Run(ctx, store.InsertInvoiceParams{ID: id, OrderID: in.OrderID,
			SupplierInvoiceSymbol: platform.NullText(trimmed(in.SupplierInvoiceSymbol)), SupplierInvoiceNo: platform.NullText(trimmed(in.SupplierInvoiceNo)),
			SupplierInvoiceDate: platform.NullDate(in.SupplierInvoiceDate),
			Subtotal:            t.Subtotal, DiscountTotal: t.DiscountTotal, VatTotal: t.VatTotal, Total: t.Total, Note: platform.NullText(in.Note)}); err != nil {
			return err
		}
		return s.writeInvoiceLines(ctx, id, lines)
	})
}

// UpdateInvoice saves a draft invoice again, checked as on creation.
func (s *Service) UpdateInvoice(ctx context.Context, id int64, in InvoiceUpdate) error {
	ref := record.Ref{Type: invoiceType, ID: id}
	return platform.InTx(ctx, func(ctx context.Context) error {
		d, err := s.d.Record.Read(ctx, ref)
		if err != nil {
			return err
		}
		order, err := store.InvoiceOrder.Run(ctx, id)
		if err != nil {
			return err
		}
		_, t, err := s.priceInvoice(ctx, id, order, d.OrgUnitID, in.InvoiceFields, false)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, ref, in.Version, record.Header{Date: in.Date, OrgUnitID: d.OrgUnitID, Amount: &t.Total}); err != nil {
			return err
		}
		lines, t, err := s.checkInvoice(ctx, id, order, d.OrgUnitID, in.InvoiceFields)
		if err != nil {
			return err
		}
		if err := store.UpdateInvoice.Run(ctx, store.UpdateInvoiceParams{ID: id,
			SupplierInvoiceSymbol: platform.NullText(trimmed(in.SupplierInvoiceSymbol)), SupplierInvoiceNo: platform.NullText(trimmed(in.SupplierInvoiceNo)),
			SupplierInvoiceDate: platform.NullDate(in.SupplierInvoiceDate),
			Subtotal:            t.Subtotal, DiscountTotal: t.DiscountTotal, VatTotal: t.VatTotal, Total: t.Total, Note: platform.NullText(in.Note)}); err != nil {
			return err
		}
		if err := store.DeleteInvoiceLines.Run(ctx, id); err != nil {
			return err
		}
		return s.writeInvoiceLines(ctx, id, lines)
	})
}

// checkInvoice locks the order, after the invoice's own row, so saves and posting of its
// invoices take turns, and refuses while an action on the order awaits approval; then it
// checks the order and prices every line.
func (s *Service) checkInvoice(ctx context.Context, id, orderID, unit int64, f InvoiceFields) ([]InvoiceLine, Totals, error) {
	o, err := s.d.Record.LockToDepend(ctx, record.Ref{Type: orderType, ID: orderID})
	if err != nil {
		return nil, Totals{}, err
	}
	if o.Status != record.Posted {
		return nil, Totals{}, ErrOrderNotPosted
	}
	if f.Date < o.Date {
		return nil, Totals{}, ErrInvoiceBeforeOrder
	}
	return s.priceInvoice(ctx, id, orderID, unit, f, true)
}

// priceInvoice takes each line's item, texts and stocked from the order line, its price, discount
// and VAT rate from the input, and computes the amounts; with held, it also refuses a quantity
// beyond what other non-cancelled invoices leave. Before the locks it only prices: a retried
// create must reach its reuse check.
func (s *Service) priceInvoice(ctx context.Context, id, orderID, unit int64, f InvoiceFields, held bool) ([]InvoiceLine, Totals, error) {
	rows, err := store.InvoiceableLines.Run(ctx, store.InvoiceableLinesParams{InvoiceID: id, OrderID: orderID})
	if err != nil {
		return nil, Totals{}, err
	}
	byPos := map[int]store.InvoiceableLinesRow{}
	for _, r := range rows {
		byPos[int(r.Position)] = r
	}
	used := map[int]decimal.Decimal{}
	in := make([]LineInput, len(f.Lines))
	for i, l := range f.Lines {
		r, ok := byPos[l.OrderPosition]
		if !ok {
			return nil, Totals{}, lineErr(ErrInvoicePosition, i+1, nil)
		}
		q, ok := record.ParseNumber(l.Quantity)
		if !ok || !q.IsPositive() || !q.Equal(q.Truncate(3)) {
			return nil, Totals{}, errLine("invalid_quantity", i+1)
		}
		ordered, taken, err := decimals(r.Quantity, r.Taken)
		if err != nil {
			return nil, Totals{}, err
		}
		used[l.OrderPosition] = used[l.OrderPosition].Add(q)
		if held && taken.Add(used[l.OrderPosition]).GreaterThan(ordered) {
			return nil, Totals{}, lineErr(ErrInvoiceOverOrder, i+1, map[string]any{
				"order_position": l.OrderPosition, "remaining": ordered.Sub(taken).StringFixed(3)})
		}
		in[i] = LineInput{ItemID: r.ItemID, Description: r.Description, Quantity: q.String(), UnitPrice: l.UnitPrice,
			DiscountPercent: strings.TrimSpace(l.DiscountPercent), VatRate: l.VatRate}
	}
	mode, err := s.d.Setting.RoundingOf(ctx, unit)
	if err != nil {
		return nil, Totals{}, err
	}
	amts, t, err := compute(in, mode)
	if err != nil {
		return nil, Totals{}, err
	}
	out := make([]InvoiceLine, len(in))
	for i, l := range in {
		r := byPos[f.Lines[i].OrderPosition]
		out[i] = InvoiceLine{InvoiceLineInput: InvoiceLineInput{OrderPosition: f.Lines[i].OrderPosition, Quantity: l.Quantity, UnitPrice: l.UnitPrice,
			DiscountPercent: l.DiscountPercent, VatRate: l.VatRate},
			ItemID: r.ItemID, ItemCode: r.ItemCode, Description: r.Description, Unit: r.Unit, Stocked: r.Stocked,
			Amount: amts[i].amount, Discount: amts[i].discount, Vat: amts[i].vat}
	}
	return out, t, nil
}

func (s *Service) writeInvoiceLines(ctx context.Context, id int64, lines []InvoiceLine) error {
	a := store.InsertInvoiceLinesParams{InvoiceID: id}
	for i, l := range lines {
		a.Positions = append(a.Positions, int32(i+1))
		a.OrderPositions = append(a.OrderPositions, int32(l.OrderPosition))
		a.ItemIds = append(a.ItemIds, l.ItemID)
		a.ItemCodes = append(a.ItemCodes, l.ItemCode)
		a.Descriptions = append(a.Descriptions, l.Description)
		a.Units = append(a.Units, l.Unit)
		a.Stocked = append(a.Stocked, l.Stocked)
		a.Quantities = append(a.Quantities, l.Quantity)
		a.UnitPrices = append(a.UnitPrices, l.UnitPrice)
		a.DiscountPercents = append(a.DiscountPercents, l.DiscountPercent)
		a.VatRates = append(a.VatRates, l.VatRate)
		a.Amounts = append(a.Amounts, l.Amount)
		a.Discounts = append(a.Discounts, l.Discount)
		a.Vats = append(a.Vats, l.Vat)
	}
	if err := store.InsertInvoiceLines.Run(ctx, a); err != nil {
		return err
	}
	return s.d.Audit.RecordFor(ctx, "purchase.invoice_saved", audit.Ref{Type: invoiceType, ID: id}, map[string]any{"lines": len(lines)})
}
