package purchase

import (
	"context"
	"slices"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/purchase/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

const paymentType = "purchase.payment"

// A payment is as open as the orders of its org unit. It has no supplier check on sending:
// an inactive supplier still pays what it owes.
func (s *Service) registerPayment() {
	s.d.Record.Register(record.Type{
		Code: paymentType, Product: "purchase", Kind: record.Document, NumberPrefix: "PC",
		Perms:        record.Perms{View: PermOrderView, Edit: PermOrderEdit, ViewFiles: PermOrderView, Attach: PermOrderEdit},
		OnTransition: s.paymentTransition,
		OnDelete:     func(ctx context.Context, id int64) error { return store.DeletePayment.Run(ctx, id) },
	})
}

// paymentTransition checks the allocations against the invoices again and writes the money
// in and the payable down on posting; cancelling voids them.
func (s *Service) paymentTransition(ctx context.Context, d record.Doc, _ record.Status) error {
	switch d.Status {
	case record.Posted:
		h, err := store.GetPayment.Run(ctx, d.ID)
		if err != nil {
			return err
		}
		saved, err := store.GetAllocations.Run(ctx, d.ID)
		if err != nil {
			return err
		}
		f := PaymentFields{Date: d.Date, OrgUnitID: d.OrgUnitID, SupplierID: h.SupplierID, Method: h.Method,
			Allocations: make([]AllocationInput, len(saved))}
		for i, a := range saved {
			f.Allocations[i] = AllocationInput{InvoiceID: a.InvoiceID, Amount: a.Amount}
		}
		if _, err := s.checkPayment(ctx, d.ID, f); err != nil {
			return err
		}
		money := KindCashPayment
		if h.Method == methodBank {
			money = KindBankPayment
		}
		return s.d.Posting.Record(ctx, d.Ref, []posting.Line{
			{Kind: money, OrgUnitID: d.OrgUnitID, Amount: h.Amount},
			{Kind: KindPayable, OrgUnitID: d.OrgUnitID, PartnerID: &h.PartnerID, Amount: -h.Amount},
		})
	case record.Cancelled:
		return s.d.Posting.Void(ctx, d.Ref)
	}
	return nil
}

// PaymentActions lists what the actor may do before picking a payment: create.
func (s *Service) PaymentActions(ctx context.Context) (Actions, error) {
	return s.typeActions(ctx, paymentType)
}

// Payment returns one payment; one the actor may not view does not exist.
func (s *Service) Payment(ctx context.Context, id int64) (Payment, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: paymentType, ID: id})
	if err != nil {
		return Payment{}, err
	}
	h, err := store.GetPayment.Run(ctx, id)
	if err != nil {
		return Payment{}, err
	}
	rows, err := store.GetAllocations.Run(ctx, id)
	if err != nil {
		return Payment{}, err
	}
	out := Payment{ID: id, Number: h.Number, Status: h.Status, Version: h.Version, Date: *platform.DatePtr(h.Date),
		OrgUnitID: h.OrgUnitID, OrgUnitName: h.OrgUnitName, SupplierID: h.SupplierID, SupplierCode: h.SupplierCode,
		SupplierName: h.SupplierName, Method: h.Method, Amount: h.Amount, Note: platform.TextPtr(h.Note),
		Allocations: make([]Allocation, len(rows))}
	for i, r := range rows {
		out.Allocations[i] = Allocation{AllocationInput: AllocationInput{InvoiceID: r.InvoiceID, Amount: r.Amount},
			InvoiceNumber: r.InvoiceNumber, InvoiceDate: *platform.DatePtr(r.InvoiceDate), InvoiceTotal: r.InvoiceTotal, Remaining: r.Remaining}
	}
	out.AllowedActions, err = s.d.Record.Actions(ctx, d)
	return out, err
}

// OpenInvoices lists the posted invoices of a supplier at an org unit that other payments,
// drafts included, leave something of; paymentID is the payment being edited, 0 for none.
func (s *Service) OpenInvoices(ctx context.Context, supplierID, orgUnitID, paymentID int64) ([]Allocation, error) {
	if err := s.d.Record.Visible(ctx, record.Ref{Type: supplierType, ID: supplierID}, record.View); err != nil {
		return nil, err
	}
	if err := s.require(ctx, PermOrderView, orgUnitID); err != nil {
		return nil, err
	}
	rows, err := store.OpenInvoices.Run(ctx, store.OpenInvoicesParams{PaymentID: paymentID, SupplierID: supplierID, OrgUnitID: orgUnitID})
	if err != nil {
		return nil, err
	}
	out := []Allocation{}
	for _, r := range rows {
		if r.Remaining > 0 {
			out = append(out, Allocation{AllocationInput: AllocationInput{InvoiceID: r.ID, Amount: r.Remaining}, InvoiceNumber: r.Number,
				InvoiceDate: *platform.DatePtr(r.Date), InvoiceTotal: r.Total, Remaining: r.Remaining})
		}
	}
	return out, nil
}

// CreatePayment adds a draft payment; the same request id returns the first one.
func (s *Service) CreatePayment(ctx context.Context, in NewPayment) (id int64, err error) {
	return id, platform.InTx(ctx, func(ctx context.Context) error {
		amount, err := s.paymentAmount(ctx, in.PaymentFields)
		if err != nil {
			return err
		}
		d, err := s.d.Record.Create(ctx, paymentType, record.Header{Date: in.Date, OrgUnitID: in.OrgUnitID, RequestID: in.RequestID, Amount: &amount})
		if id = d.ID; err != nil || d.Reused {
			return err
		}
		if _, err := s.checkPayment(ctx, id, in.PaymentFields); err != nil {
			return err
		}
		if err := store.InsertPayment.Run(ctx, store.InsertPaymentParams{ID: id, SupplierID: in.SupplierID, Method: in.Method,
			Amount: amount, Note: platform.NullText(in.Note)}); err != nil {
			return err
		}
		return s.writeAllocations(ctx, id, in.Allocations)
	})
}

// UpdatePayment saves a draft payment again, checked as on creation.
func (s *Service) UpdatePayment(ctx context.Context, id int64, in PaymentUpdate) error {
	ref := record.Ref{Type: paymentType, ID: id}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.d.Record.Read(ctx, ref); err != nil {
			return err
		}
		amount, err := s.paymentAmount(ctx, in.PaymentFields)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, ref, in.Version, record.Header{Date: in.Date, OrgUnitID: in.OrgUnitID, Amount: &amount}); err != nil {
			return err
		}
		if _, err := s.checkPayment(ctx, id, in.PaymentFields); err != nil {
			return err
		}
		if err := store.UpdatePayment.Run(ctx, store.UpdatePaymentParams{ID: id, SupplierID: in.SupplierID, Method: in.Method,
			Amount: amount, Note: platform.NullText(in.Note)}); err != nil {
			return err
		}
		if err := store.DeleteAllocations.Run(ctx, id); err != nil {
			return err
		}
		return s.writeAllocations(ctx, id, in.Allocations)
	})
}

// paymentAmount checks the supplier may be seen, refuses an invoice named twice, and sums the
// allocations. An inactive supplier still pays what it owes.
func (s *Service) paymentAmount(ctx context.Context, f PaymentFields) (int64, error) {
	if err := s.d.Record.Visible(ctx, record.Ref{Type: supplierType, ID: f.SupplierID}, record.View); err != nil {
		return 0, err
	}
	var sum int64
	seen := map[int64]bool{}
	for i, a := range f.Allocations {
		if seen[a.InvoiceID] {
			return 0, lineErr(ErrPaymentDuplicateInvoice, i+1, nil)
		}
		seen[a.InvoiceID] = true
		sum += a.Amount
	}
	return sum, nil
}

// checkPayment locks the invoices, in id order and after the payment's own row, so payments
// settling the same invoice take turns; then checks each allocation against its invoice.
func (s *Service) checkPayment(ctx context.Context, id int64, f PaymentFields) ([]store.AllocatableInvoicesRow, error) {
	ids := make([]int64, len(f.Allocations))
	for i, a := range f.Allocations {
		if err := s.d.Record.Visible(ctx, record.Ref{Type: invoiceType, ID: a.InvoiceID}, record.View); err != nil {
			return nil, err
		}
		ids[i] = a.InvoiceID
	}
	sorted := slices.Sorted(slices.Values(ids))
	for _, inv := range slices.Compact(sorted) {
		if _, err := s.d.Record.LockToDepend(ctx, record.Ref{Type: invoiceType, ID: inv}); err != nil {
			return nil, err
		}
	}
	rows, err := store.AllocatableInvoices.Run(ctx, store.AllocatableInvoicesParams{PaymentID: id, InvoiceIds: ids})
	if err != nil {
		return nil, err
	}
	byID := map[int64]store.AllocatableInvoicesRow{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	for i, a := range f.Allocations {
		r := byID[a.InvoiceID]
		switch {
		case r.Status != string(record.Posted) || r.SupplierID != f.SupplierID:
			return nil, lineErr(ErrPaymentInvoice, i+1, nil)
		case r.OrgUnitID != f.OrgUnitID:
			return nil, lineErr(ErrPaymentInvoiceOrgUnit, i+1, nil)
		case f.Date < *platform.DatePtr(r.Date):
			return nil, lineErr(ErrPaymentBeforeInvoice, i+1, nil)
		case a.Amount > r.Remaining:
			return nil, lineErr(ErrPaymentOverInvoice, i+1, map[string]any{"invoice_id": a.InvoiceID, "remaining": max(r.Remaining, 0)})
		}
	}
	return rows, nil
}

func (s *Service) writeAllocations(ctx context.Context, id int64, allocs []AllocationInput) error {
	a := store.InsertAllocationsParams{PaymentID: id}
	for i, l := range allocs {
		a.Positions = append(a.Positions, int32(i+1))
		a.InvoiceIds = append(a.InvoiceIds, l.InvoiceID)
		a.Amounts = append(a.Amounts, l.Amount)
	}
	if err := store.InsertAllocations.Run(ctx, a); err != nil {
		return err
	}
	return s.d.Audit.RecordFor(ctx, "purchase.payment_saved", audit.Ref{Type: paymentType, ID: id}, map[string]any{"allocations": len(allocs)})
}
