package sales

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// QuoteActions and OrderActions list what the actor may do before picking a document: create.
func (s *Service) QuoteActions(ctx context.Context) (Actions, error) {
	return s.createActions(ctx, PermQuoteEdit)
}
func (s *Service) OrderActions(ctx context.Context) (Actions, error) {
	return s.createActions(ctx, PermOrderEdit)
}

func (s *Service) Quotes(ctx context.Context, f DocFilter) (DocList, error) {
	return s.docs(ctx, quoteKind, f)
}
func (s *Service) Orders(ctx context.Context, f DocFilter) (DocList, error) {
	return s.docs(ctx, orderKind, f)
}
func (s *Service) Quote(ctx context.Context, id int64) (Doc, error) { return s.doc(ctx, quoteKind, id) }
func (s *Service) Order(ctx context.Context, id int64) (Doc, error) { return s.doc(ctx, orderKind, id) }

// CreateQuote and CreateOrder add a draft; the same request id returns the first one.
func (s *Service) CreateQuote(ctx context.Context, in NewDoc) (int64, error) {
	return s.createOnce(ctx, quoteKind, in)
}

func (s *Service) CreateOrder(ctx context.Context, in NewDoc) (int64, error) {
	return s.createOnce(ctx, orderKind, in)
}

func (s *Service) UpdateQuote(ctx context.Context, id int64, in DocUpdate) error {
	return s.update(ctx, quoteKind, id, in)
}

func (s *Service) UpdateOrder(ctx context.Context, id int64, in DocUpdate) error {
	return s.update(ctx, orderKind, id, in)
}

func (s *Service) DeleteQuote(ctx context.Context, id int64, version int32) error {
	return s.delete(ctx, quoteKind, id, version)
}

func (s *Service) DeleteOrder(ctx context.Context, id int64, version int32) error {
	return s.delete(ctx, orderKind, id, version)
}

func (s *Service) docs(ctx context.Context, k kind, f DocFilter) (DocList, error) {
	out := DocList{Items: []DocListItem{}}
	sc, err := s.d.IAM.Scope(ctx, "sales", k.view)
	if err != nil || !sc.Any() {
		return out, err
	}
	today, err := s.today(ctx)
	if err != nil {
		return out, err
	}
	rows, err := store.New(platform.DBFrom(ctx)).ListHeaders(ctx, store.ListHeadersParams{
		Kind: k.name, AllUnits: sc.All, Units: sc.Units, Status: f.Status, Q: f.Q,
		CustomerID: pgtype.Int8{Int64: f.CustomerID, Valid: f.CustomerID != 0},
		Sort:       f.Sort, Lim: int32(f.PageSize), Off: int32((f.Page - 1) * f.PageSize),
	})
	for _, r := range rows {
		out.Total = r.TotalRows
		out.Items = append(out.Items, DocListItem{
			ID: r.ID, Number: r.Number, Status: r.Status, Date: *platform.DatePtr(r.Date), CustomerID: r.CustomerID,
			CustomerCode: r.CustomerCode, CustomerName: r.CustomerName, OrgUnitName: r.OrgUnitName, Total: r.Total,
			ValidUntil: platform.DatePtr(r.ValidUntil), Expired: expired(r.ValidUntil, today),
			Ordered: r.Ordered, FromQuote: r.FromQuote,
		})
	}
	return out, err
}

// expired: a quotation is valid through its validity date, in the tenant time zone.
func expired(validUntil, today pgtype.Date) bool {
	return validUntil.Valid && validUntil.Time.Before(today.Time)
}

// doc returns one document; one the actor may not view does not exist.
func (s *Service) doc(ctx context.Context, k kind, id int64) (Doc, error) {
	h, err := s.visible(ctx, k, id)
	if err != nil {
		return Doc{}, err
	}
	q := store.New(platform.DBFrom(ctx))
	lines, err := q.GetLines(ctx, id)
	if err != nil {
		return Doc{}, err
	}
	d, err := s.d.Record.Get(ctx, record.Ref{Type: k.docType, ID: id})
	if err != nil {
		return Doc{}, err
	}
	actions, err := s.d.Record.DocumentActions(ctx, d)
	if err != nil {
		return Doc{}, err
	}
	if ok, err := s.d.Record.Can(ctx, k.docType, id, record.Print); err != nil {
		return Doc{}, err
	} else if ok {
		actions = append(actions, "print")
	}
	out := Doc{
		ID: id, Kind: k.name, Number: h.Number, Status: h.Status, Version: h.Version, Date: *platform.DatePtr(h.Date),
		OrgUnitID: h.OrgUnitID, OrgUnitName: h.OrgUnitName, Customer: customerCopy(h),
		ValidUntil: platform.DatePtr(h.ValidUntil), DeliveryDate: platform.DatePtr(h.DeliveryDate),
		PaymentTerms: platform.TextPtr(h.PaymentTerms), DeliveryTerms: platform.TextPtr(h.DeliveryTerms), Note: platform.TextPtr(h.Note),
		Lines:       make([]Line, len(lines)),
		Totals:      Totals{Subtotal: h.Subtotal, DiscountTotal: h.DiscountTotal, VatTotal: h.VatTotal, Total: h.Total},
		MaxDiscount: d.Fields["max_discount"],
	}
	for i, l := range lines {
		out.Lines[i] = Line{
			LineInput: LineInput{ItemID: l.ItemID, Description: l.Description, Quantity: plain(l.Quantity), UnitPrice: l.UnitPrice,
				DiscountPercent: plain(l.DiscountPercent), VatRate: l.VatRate},
			ItemCode: l.ItemCode, Unit: l.Unit, Amount: l.Amount, Discount: l.Discount, Vat: l.Vat,
		}
	}
	// The other document shows only to who may view it.
	if h.QuoteID.Valid {
		if ok, err := s.d.Record.Can(ctx, quoteType, h.QuoteID.Int64, record.View); err != nil {
			return Doc{}, err
		} else if ok {
			qd, err := s.d.Record.Get(ctx, record.Ref{Type: quoteType, ID: h.QuoteID.Int64})
			if err != nil {
				return Doc{}, err
			}
			out.Quote = &DocRef{ID: qd.ID, Number: qd.Number, Status: string(qd.Status)}
		}
	}
	if k == quoteKind {
		today, err := s.today(ctx)
		if err != nil {
			return Doc{}, err
		}
		out.Expired = expired(h.ValidUntil, today)
		o, err := q.LiveOrder(ctx, pgtype.Int8{Int64: id, Valid: true})
		switch {
		case err == nil:
			out.Ordered = true
			if ok, err := s.d.Record.Can(ctx, orderType, o.ID, record.View); err != nil {
				return Doc{}, err
			} else if ok {
				out.Order = &DocRef{ID: o.ID, Number: o.Number, Status: o.Status}
			}
		case !errors.Is(err, pgx.ErrNoRows):
			return Doc{}, err
		}
		if d.Status == record.Posted && !out.Expired && !out.Ordered {
			if ok, err := s.canMakeOrder(ctx, h.OrgUnitID); err != nil {
				return Doc{}, err
			} else if ok {
				out.AllowedActions = append(actions, "create_order")
			}
		}
	}
	if out.AllowedActions == nil {
		out.AllowedActions = actions
	}
	return out, nil
}

func (s *Service) canMakeOrder(ctx context.Context, unit int64) (bool, error) {
	if platform.ProductGate(ctx, "sales", platform.ClassWrite) != nil {
		return false, nil
	}
	return s.allowed(ctx, PermOrderEdit, unit)
}

func (s *Service) visible(ctx context.Context, k kind, id int64) (store.GetHeaderRow, error) {
	if ok, err := s.d.Record.Can(ctx, k.docType, id, record.View); err != nil || !ok {
		return store.GetHeaderRow{}, platform.OrErr(err, platform.ErrNotFound)
	}
	return store.New(platform.DBFrom(ctx)).GetHeader(ctx, id)
}

// createOnce creates a draft unless its request id was seen: then it returns that draft,
// also when a concurrent twin wins the unique index first.
func (s *Service) createOnce(ctx context.Context, k kind, in NewDoc) (int64, error) {
	if id, found, err := s.byRequest(ctx, k, in.RequestID); err != nil || found {
		return id, err
	}
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		var err error
		id, err = s.create(ctx, k, in.RequestID, in.DocFields, nil)
		return err
	})
	if isUnique(err, "headers_request_id_key") {
		id, _, err = s.byRequest(ctx, k, in.RequestID)
	}
	return id, err
}

func (s *Service) byRequest(ctx context.Context, k kind, requestID string) (int64, bool, error) {
	r, err := store.New(platform.DBFrom(ctx)).HeaderByRequest(ctx, requestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if r.Kind != k.name {
		return 0, false, ErrRequestReused
	}
	if ok, err := s.d.Record.Can(ctx, k.docType, r.ID, record.View); err != nil || !ok {
		return 0, false, platform.OrErr(err, ErrRequestReused)
	}
	return r.ID, true, nil
}

// create checks the actor may create at the org unit and see the customer, then adds the draft.
func (s *Service) create(ctx context.Context, k kind, requestID string, f DocFields, quoteID *int64) (int64, error) {
	p, err := s.prepare(ctx, k, f)
	if err != nil {
		return 0, err
	}
	d, err := s.d.Record.Create(ctx, k.docType, p.header)
	if err != nil {
		return 0, err
	}
	c := p.customer
	if err := store.New(platform.DBFrom(ctx)).InsertHeader(ctx, store.InsertHeaderParams{
		ID: d.ID, Kind: k.name, RequestID: requestID, CustomerID: c.ID, CustomerCode: c.Code, CustomerName: c.Name,
		CustomerTaxCode: platform.NullText(c.TaxCode), CustomerAddress: platform.NullText(c.Address), CustomerPhone: platform.NullText(c.Phone),
		CustomerEmail: platform.NullText(c.Email), ContactName: platform.NullText(c.ContactName),
		ValidUntil: platform.NullDate(f.ValidUntil), DeliveryDate: platform.NullDate(f.DeliveryDate), QuoteID: platform.NullInt8(quoteID),
		PaymentTerms: platform.NullText(f.PaymentTerms), DeliveryTerms: platform.NullText(f.DeliveryTerms), Note: platform.NullText(f.Note),
		Subtotal: p.totals.Subtotal, DiscountTotal: p.totals.DiscountTotal, VatTotal: p.totals.VatTotal, Total: p.totals.Total,
	}); err != nil {
		return 0, err
	}
	return d.ID, s.writeLines(ctx, k, d.ID, p)
}

func (s *Service) update(ctx context.Context, k kind, id int64, in DocUpdate) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.visible(ctx, k, id); err != nil {
			return err
		}
		p, err := s.prepare(ctx, k, in.DocFields)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, record.Ref{Type: k.docType, ID: id}, in.Version, p.header); err != nil {
			return err
		}
		c, f := p.customer, in.DocFields
		q := store.New(platform.DBFrom(ctx))
		if err := q.UpdateHeader(ctx, store.UpdateHeaderParams{
			ID: id, CustomerID: c.ID, CustomerCode: c.Code, CustomerName: c.Name,
			CustomerTaxCode: platform.NullText(c.TaxCode), CustomerAddress: platform.NullText(c.Address), CustomerPhone: platform.NullText(c.Phone),
			CustomerEmail: platform.NullText(c.Email), ContactName: platform.NullText(c.ContactName),
			ValidUntil: platform.NullDate(f.ValidUntil), DeliveryDate: platform.NullDate(f.DeliveryDate),
			PaymentTerms: platform.NullText(f.PaymentTerms), DeliveryTerms: platform.NullText(f.DeliveryTerms), Note: platform.NullText(f.Note),
			Subtotal: p.totals.Subtotal, DiscountTotal: p.totals.DiscountTotal, VatTotal: p.totals.VatTotal, Total: p.totals.Total,
		}); err != nil {
			return err
		}
		if err := q.DeleteLines(ctx, id); err != nil {
			return err
		}
		return s.writeLines(ctx, k, id, p)
	})
}

func (s *Service) delete(ctx context.Context, k kind, id int64, version int32) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.visible(ctx, k, id); err != nil {
			return err
		}
		if err := s.d.Record.Delete(ctx, record.Ref{Type: k.docType, ID: id}, version); err != nil {
			return err
		}
		return store.New(platform.DBFrom(ctx)).DeleteHeader(ctx, id)
	})
}

// prepared is a draft's content, checked against the current catalogues and computed.
type prepared struct {
	header   record.Header
	customer CustomerCopy
	lines    []Line
	totals   Totals
}

// prepare checks the fields, copies the customer and items as they are now, and computes
// every amount with the rounding mode of the org unit's legal entity.
func (s *Service) prepare(ctx context.Context, k kind, f DocFields) (prepared, error) {
	var p prepared
	if err := s.require(ctx, k.edit, f.OrgUnitID); err != nil {
		return p, err
	}
	switch {
	case k == quoteKind && (f.ValidUntil == nil || *f.ValidUntil < f.Date):
		return p, ErrValidUntil
	case k == orderKind && f.DeliveryDate != nil && *f.DeliveryDate < f.Date:
		return p, ErrDeliveryDate
	}
	if k == quoteKind {
		f.DeliveryDate = nil
	} else {
		f.ValidUntil = nil
	}
	c, err := s.visibleCustomer(ctx, f.CustomerID)
	if err != nil {
		return p, err
	}
	if !c.Active {
		return p, ErrCustomerInactive
	}
	p.customer = CustomerCopy{ID: c.ID, Code: c.Code, Name: c.Name, TaxCode: platform.TextPtr(c.TaxCode), Address: platform.TextPtr(c.Address),
		Phone: platform.TextPtr(c.Phone), Email: platform.TextPtr(c.Email), ContactName: platform.TextPtr(c.ContactName)}
	ids := make([]int64, len(f.Lines))
	for i, l := range f.Lines {
		ids[i] = l.ItemID
	}
	rows, err := store.New(platform.DBFrom(ctx)).ItemsByID(ctx, ids)
	if err != nil {
		return p, err
	}
	items := map[int64]store.SalesItem{}
	for _, r := range rows {
		items[r.ID] = r
	}
	le, err := s.d.IAM.LegalEntityOf(ctx, f.OrgUnitID)
	if err != nil {
		return p, err
	}
	mode, err := s.d.Setting.GetFor(ctx, le, setting.Rounding)
	if err != nil {
		return p, err
	}
	list := make([]int64, len(f.Lines))
	for i, l := range f.Lines {
		it, ok := items[l.ItemID]
		if !ok || !it.Active {
			return p, errLine("item_inactive", i+1)
		}
		list[i] = it.Price
	}
	amts, totals, maxDiscount, err := compute(f.Lines, list, platform.Rounding(mode))
	if err != nil {
		return p, err
	}
	p.lines = make([]Line, len(f.Lines))
	for i, l := range f.Lines {
		it := items[l.ItemID]
		p.lines[i] = Line{LineInput: l, ItemCode: it.Code, Unit: it.Unit, Amount: amts[i].amount, Discount: amts[i].discount, Vat: amts[i].vat}
	}
	p.totals = totals
	p.header = record.Header{Date: f.Date, OrgUnitID: f.OrgUnitID, Amount: &totals.Total, Fields: map[string]string{"max_discount": maxDiscount.String()}}
	return p, nil
}

func (s *Service) writeLines(ctx context.Context, k kind, id int64, p prepared) error {
	q := store.New(platform.DBFrom(ctx))
	for i, l := range p.lines {
		if err := q.InsertLine(ctx, store.InsertLineParams{
			DocID: id, Position: int32(i + 1), ItemID: l.ItemID, ItemCode: l.ItemCode, Description: l.Description, Unit: l.Unit,
			Quantity: l.Quantity, UnitPrice: l.UnitPrice, DiscountPercent: l.DiscountPercent, VatRate: l.VatRate,
			Amount: l.Amount, Discount: l.Discount, Vat: l.Vat,
		}); err != nil {
			return err
		}
	}
	return s.d.Audit.RecordFor(ctx, "sales.document_saved", audit.Ref{Type: k.docType, ID: id},
		map[string]any{"customer_id": p.customer.ID, "lines": len(p.lines), "total": p.totals.Total})
}

// CreateOrderFromQuote makes a draft order of a posted, unexpired quotation: its customer,
// terms and lines as quoted, dated today. While an order from it is not cancelled, it
// returns that order instead, so a retried click never makes a second one.
func (s *Service) CreateOrderFromQuote(ctx context.Context, quoteID int64) (int64, error) {
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		qh, err := s.visible(ctx, quoteKind, quoteID)
		if err != nil {
			return err
		}
		// Checked before an existing order is returned, so only who may make one learns of it.
		if err := s.d.Record.WriteGate(ctx, record.Ref{Type: quoteType, ID: quoteID}); err != nil {
			return err
		}
		if err := s.require(ctx, PermOrderEdit, qh.OrgUnitID); err != nil {
			return err
		}
		d, err := s.d.Record.Lock(ctx, record.Ref{Type: quoteType, ID: quoteID})
		if err != nil {
			return err
		}
		q := store.New(platform.DBFrom(ctx))
		live, err := q.LiveOrder(ctx, pgtype.Int8{Int64: quoteID, Valid: true})
		if err == nil {
			if ok, err := s.d.Record.Can(ctx, orderType, live.ID, record.View); err != nil || !ok {
				return platform.OrErr(err, ErrQuoteHasOrder)
			}
			id = live.ID
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if d.Status != record.Posted {
			return ErrQuoteNotPosted
		}
		h, err := q.GetHeader(ctx, quoteID)
		if err != nil {
			return err
		}
		today, err := s.today(ctx)
		if err != nil {
			return err
		}
		if expired(h.ValidUntil, today) {
			return ErrQuoteExpired
		}
		lines, err := q.GetLines(ctx, quoteID)
		if err != nil {
			return err
		}
		f := DocFields{Date: *platform.DatePtr(today), OrgUnitID: h.OrgUnitID, CustomerID: h.CustomerID,
			PaymentTerms: platform.TextPtr(h.PaymentTerms), DeliveryTerms: platform.TextPtr(h.DeliveryTerms), Note: platform.TextPtr(h.Note),
			Lines: make([]LineInput, len(lines))}
		for i, l := range lines {
			f.Lines[i] = LineInput{ItemID: l.ItemID, Description: l.Description, Quantity: l.Quantity, UnitPrice: l.UnitPrice,
				DiscountPercent: l.DiscountPercent, VatRate: l.VatRate}
		}
		id, err = s.create(ctx, orderKind, newRequestID(), f, &quoteID)
		return err
	})
	return id, err
}

// plain writes a stored numeric without trailing zeros: "2.000" is "2".
func plain(s string) string { return decimal.RequireFromString(s).String() }

func customerCopy(h store.GetHeaderRow) CustomerCopy {
	return CustomerCopy{ID: h.CustomerID, Code: h.CustomerCode, Name: h.CustomerName, TaxCode: platform.TextPtr(h.CustomerTaxCode),
		Address: platform.TextPtr(h.CustomerAddress), Phone: platform.TextPtr(h.CustomerPhone), Email: platform.TextPtr(h.CustomerEmail),
		ContactName: platform.TextPtr(h.ContactName)}
}

// newRequestID is a random UUID (version 4) for a create the server starts itself.
func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:]) // never fails (crypto/rand)
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
