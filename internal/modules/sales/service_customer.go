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

// canCustomer goes by the customer's owning org unit; its files are as open as the customer.
func (s *Service) canCustomer(ctx context.Context, id int64, action record.Action) (bool, error) {
	c, err := store.New(platform.DBFrom(ctx)).GetCustomer(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	switch action {
	case record.View, record.ViewFiles:
		return s.allowed(ctx, PermCustomerView, c.OrgUnitID)
	case record.Edit, record.Attach:
		return s.allowed(ctx, PermCustomerEdit, c.OrgUnitID)
	}
	return false, nil
}

// CustomerActions: create when the actor may edit customers somewhere and Sales is on.
func (s *Service) CustomerActions(ctx context.Context) (Actions, error) {
	return s.createActions(ctx, PermCustomerEdit)
}

func (s *Service) createActions(ctx context.Context, perm string) (Actions, error) {
	out := Actions{AllowedActions: []string{}}
	sc, err := s.d.IAM.Scope(ctx, "sales", perm)
	if err != nil {
		return out, err
	}
	if sc.Any() && platform.ProductGate(ctx, "sales", platform.ClassWrite) == nil {
		out.AllowedActions = append(out.AllowedActions, "create")
	}
	return out, nil
}

// Customers lists the customers under the org units where the actor may view them.
func (s *Service) Customers(ctx context.Context, f CustomerFilter) (CustomerList, error) {
	out := CustomerList{Items: []CustomerListItem{}}
	sc, err := s.d.IAM.Scope(ctx, "sales", PermCustomerView)
	if err != nil || !sc.Any() {
		return out, err
	}
	rows, err := store.New(platform.DBFrom(ctx)).ListCustomers(ctx, store.ListCustomersParams{
		AllUnits: sc.All, Units: sc.Units, Q: strings.TrimSpace(f.Q), Active: boolFilter(f.Active),
		Sort: f.Sort, Lim: int32(f.PageSize), Off: int32((f.Page - 1) * f.PageSize),
	})
	for _, r := range rows {
		out.Total = r.Total
		out.Items = append(out.Items, CustomerListItem{ID: r.ID, Code: r.Code, Name: r.Name, TaxCode: platform.TextPtr(r.TaxCode),
			Phone: platform.TextPtr(r.Phone), OrgUnitName: r.OrgUnitName, Active: r.Active, PaymentTerms: platform.TextPtr(r.PaymentTerms)})
	}
	return out, err
}

// Customer returns one customer; one the actor may not view does not exist.
func (s *Service) Customer(ctx context.Context, id int64) (Customer, error) {
	c, err := s.visibleCustomer(ctx, id)
	if err != nil {
		return Customer{}, err
	}
	actions, err := s.d.Record.AllowedActions(ctx, customerType, id, record.Edit, record.Attach)
	return Customer{ID: id, CustomerFields: customerFields(c), OrgUnitName: c.OrgUnitName, AllowedActions: actions}, err
}

// CreateCustomer adds a customer owned by an org unit where the actor may edit customers.
func (s *Service) CreateCustomer(ctx context.Context, in CustomerFields) (int64, error) {
	if err := platform.ProductGate(ctx, "sales", platform.ClassWrite); err != nil {
		return 0, err
	}
	if err := s.require(ctx, PermCustomerEdit, in.OrgUnitID); err != nil {
		return 0, err
	}
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		var err error
		id, err = store.New(platform.DBFrom(ctx)).CreateCustomer(ctx, store.CreateCustomerParams{
			Code: strings.TrimSpace(in.Code), Name: strings.TrimSpace(in.Name), TaxCode: platform.NullText(in.TaxCode),
			Address: platform.NullText(in.Address), Phone: platform.NullText(in.Phone), Email: platform.NullText(in.Email),
			ContactName: platform.NullText(in.ContactName), PaymentTerms: platform.NullText(in.PaymentTerms), OrgUnitID: in.OrgUnitID, Active: in.Active,
		})
		if platform.Violates(err, "customers_code_key") {
			return ErrCustomerCodeTaken
		}
		if platform.Violates(err, "customers_org_unit_id_fkey") {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "sales.customer_created", audit.Ref{Type: customerType, ID: id}, customerChanges(CustomerFields{}, in))
	})
	return id, err
}

// UpdateCustomer replaces a customer's fields; moving it needs edit at both org units.
func (s *Service) UpdateCustomer(ctx context.Context, id int64, in CustomerFields) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		// Locked first, so the checks and the audit diff see what a concurrent edit left.
		if err := store.New(platform.DBFrom(ctx)).LockCustomer(ctx, id); err != nil {
			return err
		}
		old, err := s.visibleCustomer(ctx, id)
		if err != nil {
			return err
		}
		if err := s.d.Record.WriteGate(ctx, record.Ref{Type: customerType, ID: id}); err != nil {
			return err
		}
		if err := s.require(ctx, PermCustomerEdit, old.OrgUnitID, in.OrgUnitID); err != nil {
			return err
		}
		err = store.New(platform.DBFrom(ctx)).UpdateCustomer(ctx, store.UpdateCustomerParams{
			ID: id, Code: strings.TrimSpace(in.Code), Name: strings.TrimSpace(in.Name), TaxCode: platform.NullText(in.TaxCode),
			Address: platform.NullText(in.Address), Phone: platform.NullText(in.Phone), Email: platform.NullText(in.Email),
			ContactName: platform.NullText(in.ContactName), PaymentTerms: platform.NullText(in.PaymentTerms), OrgUnitID: in.OrgUnitID, Active: in.Active,
		})
		if platform.Violates(err, "customers_code_key") {
			return ErrCustomerCodeTaken
		}
		if platform.Violates(err, "customers_org_unit_id_fkey") {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "sales.customer_updated", audit.Ref{Type: customerType, ID: id}, customerChanges(customerFields(old), in))
	})
}

func (s *Service) visibleCustomer(ctx context.Context, id int64) (store.GetCustomerRow, error) {
	if ok, err := s.d.Record.Can(ctx, customerType, id, record.View); err != nil || !ok {
		return store.GetCustomerRow{}, platform.OrErr(err, platform.ErrNotFound)
	}
	return store.New(platform.DBFrom(ctx)).GetCustomer(ctx, id)
}

func customerFields(c store.GetCustomerRow) CustomerFields {
	return CustomerFields{Code: c.Code, Name: c.Name, TaxCode: platform.TextPtr(c.TaxCode), Address: platform.TextPtr(c.Address),
		Phone: platform.TextPtr(c.Phone), Email: platform.TextPtr(c.Email), ContactName: platform.TextPtr(c.ContactName),
		PaymentTerms: platform.TextPtr(c.PaymentTerms), OrgUnitID: c.OrgUnitID, Active: c.Active}
}

func customerChanges(old, n CustomerFields) []audit.Change {
	return []audit.Change{
		{Field: "code", Old: old.Code, New: n.Code},
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
