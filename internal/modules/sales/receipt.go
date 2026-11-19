package sales

import (
	"context"
	"slices"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

const receiptType = "sales.receipt"

// A receipt is as open as the orders of its org unit. It has no customer check on sending:
// an inactive customer still pays what it owes.
func (s *Service) registerReceipt() {
	s.d.Record.Register(record.Type{
		Code: receiptType, Product: "sales", Kind: record.Document, NumberPrefix: "PT",
		Perms:        record.Perms{View: PermOrderView, Edit: PermOrderEdit, ViewFiles: PermOrderView, Attach: PermOrderEdit},
		OnTransition: s.receiptTransition,
		OnDelete:     func(ctx context.Context, id int64) error { return store.DeleteReceipt.Run(ctx, id) },
	})
}

// receiptTransition checks the allocations against the invoices again and writes the money
// in and the receivable down on posting; cancelling voids them.
func (s *Service) receiptTransition(ctx context.Context, d record.Doc, _ record.Status) error {
	switch d.Status {
	case record.Posted:
		h, err := store.GetReceipt.Run(ctx, d.ID)
		if err != nil {
			return err
		}
		saved, err := store.GetAllocations.Run(ctx, d.ID)
		if err != nil {
			return err
		}
		f := ReceiptFields{Date: d.Date, OrgUnitID: d.OrgUnitID, CustomerID: h.CustomerID, Method: h.Method,
			Allocations: make([]AllocationInput, len(saved))}
		for i, a := range saved {
			f.Allocations[i] = AllocationInput{InvoiceID: a.InvoiceID, Amount: a.Amount}
		}
		if _, err := s.checkReceipt(ctx, d.ID, f); err != nil {
			return err
		}
		money := KindCashReceipt
		if h.Method == methodBank {
			money = KindBankReceipt
		}
		return s.d.Posting.Record(ctx, d.Ref, []posting.Line{
			{Kind: money, OrgUnitID: d.OrgUnitID, Amount: h.Amount},
			{Kind: KindReceivable, OrgUnitID: d.OrgUnitID, PartnerID: &h.PartnerID, Amount: -h.Amount},
		})
	case record.Cancelled:
		return s.d.Posting.Void(ctx, d.Ref)
	}
	return nil
}

// ReceiptActions lists what the actor may do before picking a receipt: create.
func (s *Service) ReceiptActions(ctx context.Context) (Actions, error) {
	return s.typeActions(ctx, receiptType)
}

// Receipt returns one receipt; one the actor may not view does not exist.
func (s *Service) Receipt(ctx context.Context, id int64) (Receipt, error) {
	d, err := s.d.Record.Read(ctx, record.Ref{Type: receiptType, ID: id})
	if err != nil {
		return Receipt{}, err
	}
	h, err := store.GetReceipt.Run(ctx, id)
	if err != nil {
		return Receipt{}, err
	}
	rows, err := store.GetAllocations.Run(ctx, id)
	if err != nil {
		return Receipt{}, err
	}
	out := Receipt{ID: id, Number: h.Number, Status: h.Status, Version: h.Version, Date: *platform.DatePtr(h.Date),
		OrgUnitID: h.OrgUnitID, OrgUnitName: h.OrgUnitName, CustomerID: h.CustomerID, CustomerCode: h.CustomerCode,
		CustomerName: h.CustomerName, Method: h.Method, Amount: h.Amount, Note: platform.TextPtr(h.Note),
		Allocations: make([]Allocation, len(rows))}
	for i, r := range rows {
		out.Allocations[i] = Allocation{AllocationInput: AllocationInput{InvoiceID: r.InvoiceID, Amount: r.Amount},
			InvoiceNumber: r.InvoiceNumber, InvoiceDate: *platform.DatePtr(r.InvoiceDate), InvoiceTotal: r.InvoiceTotal, Remaining: r.Remaining}
	}
	out.AllowedActions, err = s.d.Record.Actions(ctx, d)
	return out, err
}

// OpenInvoices lists the posted invoices of a customer at an org unit that other receipts,
// drafts included, leave something of; receiptID is the receipt being edited, 0 for none.
func (s *Service) OpenInvoices(ctx context.Context, customerID, orgUnitID, receiptID int64) ([]Allocation, error) {
	if err := s.d.Record.Visible(ctx, record.Ref{Type: customerType, ID: customerID}, record.View); err != nil {
		return nil, err
	}
	if err := s.require(ctx, PermOrderView, orgUnitID); err != nil {
		return nil, err
	}
	rows, err := store.OpenInvoices.Run(ctx, store.OpenInvoicesParams{ReceiptID: receiptID, CustomerID: customerID, OrgUnitID: orgUnitID})
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

// CreateReceipt adds a draft receipt; the same request id returns the first one.
func (s *Service) CreateReceipt(ctx context.Context, in NewReceipt) (id int64, err error) {
	return id, platform.InTx(ctx, func(ctx context.Context) error {
		amount, err := s.receiptAmount(ctx, in.ReceiptFields)
		if err != nil {
			return err
		}
		d, err := s.d.Record.Create(ctx, receiptType, record.Header{Date: in.Date, OrgUnitID: in.OrgUnitID, RequestID: in.RequestID, Amount: &amount})
		if id = d.ID; err != nil || d.Reused {
			return err
		}
		if _, err := s.checkReceipt(ctx, id, in.ReceiptFields); err != nil {
			return err
		}
		if err := store.InsertReceipt.Run(ctx, store.InsertReceiptParams{ID: id, CustomerID: in.CustomerID, Method: in.Method,
			Amount: amount, Note: platform.NullText(in.Note)}); err != nil {
			return err
		}
		return s.writeAllocations(ctx, id, in.Allocations)
	})
}

// UpdateReceipt saves a draft receipt again, checked as on creation.
func (s *Service) UpdateReceipt(ctx context.Context, id int64, in ReceiptUpdate) error {
	ref := record.Ref{Type: receiptType, ID: id}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.d.Record.Read(ctx, ref); err != nil {
			return err
		}
		amount, err := s.receiptAmount(ctx, in.ReceiptFields)
		if err != nil {
			return err
		}
		if _, err := s.d.Record.Edit(ctx, ref, in.Version, record.Header{Date: in.Date, OrgUnitID: in.OrgUnitID, Amount: &amount}); err != nil {
			return err
		}
		if _, err := s.checkReceipt(ctx, id, in.ReceiptFields); err != nil {
			return err
		}
		if err := store.UpdateReceipt.Run(ctx, store.UpdateReceiptParams{ID: id, CustomerID: in.CustomerID, Method: in.Method,
			Amount: amount, Note: platform.NullText(in.Note)}); err != nil {
			return err
		}
		if err := store.DeleteAllocations.Run(ctx, id); err != nil {
			return err
		}
		return s.writeAllocations(ctx, id, in.Allocations)
	})
}

// receiptAmount checks the customer may be seen, refuses an invoice named twice, and sums the
// allocations. An inactive customer still pays what it owes.
func (s *Service) receiptAmount(ctx context.Context, f ReceiptFields) (int64, error) {
	if err := s.d.Record.Visible(ctx, record.Ref{Type: customerType, ID: f.CustomerID}, record.View); err != nil {
		return 0, err
	}
	var sum int64
	seen := map[int64]bool{}
	for i, a := range f.Allocations {
		if seen[a.InvoiceID] {
			return 0, lineErr(ErrReceiptDuplicateInvoice, i+1, nil)
		}
		seen[a.InvoiceID] = true
		sum += a.Amount
	}
	return sum, nil
}

// checkReceipt locks the invoices, in id order and after the receipt's own row, so receipts
// settling the same invoice take turns; then checks each allocation against its invoice.
func (s *Service) checkReceipt(ctx context.Context, id int64, f ReceiptFields) ([]store.AllocatableInvoicesRow, error) {
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
	rows, err := store.AllocatableInvoices.Run(ctx, store.AllocatableInvoicesParams{ReceiptID: id, InvoiceIds: ids})
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
		case r.Status != string(record.Posted) || r.CustomerID != f.CustomerID:
			return nil, lineErr(ErrReceiptInvoice, i+1, nil)
		case r.OrgUnitID != f.OrgUnitID:
			return nil, lineErr(ErrReceiptInvoiceOrgUnit, i+1, nil)
		case f.Date < *platform.DatePtr(r.Date):
			return nil, lineErr(ErrReceiptBeforeInvoice, i+1, nil)
		case a.Amount > r.Remaining:
			return nil, lineErr(ErrReceiptOverInvoice, i+1, map[string]any{"invoice_id": a.InvoiceID, "remaining": max(r.Remaining, 0)})
		}
	}
	return rows, nil
}

func (s *Service) writeAllocations(ctx context.Context, id int64, allocs []AllocationInput) error {
	a := store.InsertAllocationsParams{ReceiptID: id}
	for i, l := range allocs {
		a.Positions = append(a.Positions, int32(i+1))
		a.InvoiceIds = append(a.InvoiceIds, l.InvoiceID)
		a.Amounts = append(a.Amounts, l.Amount)
	}
	if err := store.InsertAllocations.Run(ctx, a); err != nil {
		return err
	}
	return s.d.Audit.RecordFor(ctx, "sales.receipt_saved", audit.Ref{Type: receiptType, ID: id}, map[string]any{"allocations": len(allocs)})
}
