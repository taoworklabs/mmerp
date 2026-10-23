package sales

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// locateCustomer: a customer is owned by its org unit.
func locateCustomer(ctx context.Context, id int64) (int64, error) {
	c, err := store.GetCustomer.Run(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, platform.ErrNotFound
	}
	return c.OrgUnitID, err
}

// CustomerActions: create when the actor may edit customers somewhere and Sales is on.
func (s *Service) CustomerActions(ctx context.Context) (Actions, error) {
	return s.typeActions(ctx, customerType)
}

func (s *Service) typeActions(ctx context.Context, code string) (Actions, error) {
	acts, err := s.d.Record.TypeActions(ctx, code)
	return Actions{AllowedActions: acts}, err
}

// Customers lists the customers under the org units where the actor may view them.
func (s *Service) Customers(ctx context.Context, f CustomerFilter) (CustomerList, error) {
	out := CustomerList{Items: []CustomerListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "sales", PermCustomerView)
	if err != nil {
		return out, err
	}
	page, err := customerList.Run(ctx, customerQuery(sc, f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, CustomerListItem{ID: r.ID, Code: r.Code, Name: r.Name, TaxCode: platform.TextPtr(r.TaxCode),
			Phone: platform.TextPtr(r.Phone), OrgUnitName: r.OrgUnitName, Active: r.Active, PaymentTerms: platform.TextPtr(r.PaymentTerms)})
	}
	return out, nil
}

// Customer returns one customer; one the actor may not view does not exist.
func (s *Service) Customer(ctx context.Context, id int64) (Customer, error) {
	if err := s.d.Record.Visible(ctx, record.Ref{Type: customerType, ID: id}, record.View); err != nil {
		return Customer{}, err
	}
	c, err := store.GetCustomer.Run(ctx, id)
	if err != nil {
		return Customer{}, err
	}
	actions, err := s.d.Record.Actions(ctx, record.Doc{Ref: record.Ref{Type: customerType, ID: id}})
	return Customer{ID: id, Code: c.Code, CustomerFields: customerFields(c), OrgUnitName: c.OrgUnitName, AllowedActions: actions}, err
}

// CreateCustomer adds a customer owned by an org unit where the actor may edit customers.
func (s *Service) CreateCustomer(ctx context.Context, in NewCustomer) (int64, error) {
	if err := platform.ProductGate(ctx, "sales", platform.ClassWrite); err != nil {
		return 0, err
	}
	if err := s.require(ctx, PermCustomerEdit, in.OrgUnitID); err != nil {
		return 0, err
	}
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		var err error
		if in.Code, err = s.d.Numbering.Code(ctx, customerType, in.Code, customerCodeTaken); err != nil {
			return err
		}
		id, err = store.CreateCustomer.Run(ctx, store.CreateCustomerParams{
			Code: in.Code, Name: strings.TrimSpace(in.Name), TaxCode: platform.NullText(in.TaxCode),
			Address: platform.NullText(in.Address), Phone: platform.NullText(in.Phone), Email: platform.NullText(in.Email),
			ContactName: platform.NullText(in.ContactName), PaymentTerms: platform.NullText(in.PaymentTerms), OrgUnitID: in.OrgUnitID, Active: in.Active,
		})
		if platform.Violates(err, "customers_org_unit_id_fkey") {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "sales.customer_created", audit.Ref{Type: customerType, ID: id}, append([]audit.Change{{Field: "code", New: in.Code}}, customerChanges(CustomerFields{}, in.CustomerFields)...))
	})
	return id, err
}

// UpdateCustomer replaces a customer's fields; moving it needs edit at both org units.
func (s *Service) UpdateCustomer(ctx context.Context, id int64, in CustomerFields) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		// Locked first, so the checks and the audit diff see what a concurrent edit left.
		if err := store.LockCustomer.Run(ctx, id); err != nil {
			return err
		}
		ref := record.Ref{Type: customerType, ID: id}
		if err := s.d.Record.Visible(ctx, ref, record.View); err != nil {
			return err
		}
		old, err := store.GetCustomer.Run(ctx, id)
		if err != nil {
			return err
		}
		if err := s.d.Record.WriteGate(ctx, ref); err != nil {
			return err
		}
		if err := s.require(ctx, PermCustomerEdit, old.OrgUnitID, in.OrgUnitID); err != nil {
			return err
		}
		err = store.UpdateCustomer.Run(ctx, store.UpdateCustomerParams{
			ID: id, Name: strings.TrimSpace(in.Name), TaxCode: platform.NullText(in.TaxCode),
			Address: platform.NullText(in.Address), Phone: platform.NullText(in.Phone), Email: platform.NullText(in.Email),
			ContactName: platform.NullText(in.ContactName), PaymentTerms: platform.NullText(in.PaymentTerms), OrgUnitID: in.OrgUnitID, Active: in.Active,
		})
		if platform.Violates(err, "customers_org_unit_id_fkey") {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "sales.customer_updated", audit.Ref{Type: customerType, ID: id}, customerChanges(customerFields(old), in))
	})
}

func customerFields(c store.GetCustomerRow) CustomerFields {
	return CustomerFields{Name: c.Name, TaxCode: platform.TextPtr(c.TaxCode), Address: platform.TextPtr(c.Address),
		Phone: platform.TextPtr(c.Phone), Email: platform.TextPtr(c.Email), ContactName: platform.TextPtr(c.ContactName),
		PaymentTerms: platform.TextPtr(c.PaymentTerms), OrgUnitID: c.OrgUnitID, Active: c.Active}
}

func customerChanges(old, n CustomerFields) []audit.Change {
	return []audit.Change{
		{Field: "name", Old: old.Name, New: n.Name},
		{Field: "tax_code", Old: old.TaxCode, New: n.TaxCode},
		{Field: "address", Old: old.Address, New: n.Address},
		{Field: "phone", Old: old.Phone, New: n.Phone},
		{Field: "email", Old: old.Email, New: n.Email},
		{Field: "contact_name", Old: old.ContactName, New: n.ContactName},
		{Field: "payment_terms", Old: old.PaymentTerms, New: n.PaymentTerms},
		{Field: "org_unit_id", Old: old.OrgUnitID, New: n.OrgUnitID},
		{Field: "active", Old: old.Active, New: n.Active},
	}
}

// boolFilter reads an optional "true"/"false" query value.
func boolFilter(v string) pgtype.Bool {
	return pgtype.Bool{Bool: v == "true", Valid: v != ""}
}

// customerCodeTaken reports whether a code is used, for numbering to skip it.
func customerCodeTaken(ctx context.Context, code string) (bool, error) {
	return store.CustomerCodeTaken.Run(ctx, code)
}
