package sales

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/partner"
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

// Customer returns one customer with its partner's identity; one the actor may not view
// does not exist.
func (s *Service) Customer(ctx context.Context, id int64) (Customer, error) {
	if err := s.d.Record.Visible(ctx, record.Ref{Type: customerType, ID: id}, record.View); err != nil {
		return Customer{}, err
	}
	c, err := store.GetCustomer.Run(ctx, id)
	if err != nil {
		return Customer{}, err
	}
	actions, err := s.d.Record.Actions(ctx, record.Doc{Ref: record.Ref{Type: customerType, ID: id}})
	return Customer{ID: id, Code: c.Code, PartnerID: c.PartnerID, PartnerCode: c.PartnerCode, Partner: partnerFields(c),
		CustomerFields: customerFields(c), OrgUnitName: c.OrgUnitName, AllowedActions: actions}, err
}

// CreateCustomer adds a customer owned by an org unit where the actor may edit customers,
// over an existing partner or one it creates in the same transaction.
func (s *Service) CreateCustomer(ctx context.Context, in NewCustomer) (int64, error) {
	if err := platform.ProductGate(ctx, "sales", platform.ClassWrite); err != nil {
		return 0, err
	}
	if (in.PartnerID == nil) == (in.Partner == nil) {
		return 0, ErrPartnerRequired
	}
	if err := s.require(ctx, PermCustomerEdit, in.OrgUnitID); err != nil {
		return 0, err
	}
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		var partnerID int64
		var err error
		if in.PartnerID != nil {
			// Only a partner the actor may see.
			p, err := s.d.Partner.Get(ctx, *in.PartnerID)
			if err != nil {
				return err
			}
			partnerID = p.ID
		} else if partnerID, err = s.d.Partner.Create(ctx, "", *in.Partner); err != nil {
			return err
		}
		if in.Code, err = s.d.Numbering.Code(ctx, customerType, in.Code, customerCodeTaken); err != nil {
			return err
		}
		id, err = store.CreateCustomer.Run(ctx, store.CreateCustomerParams{
			Code: in.Code, PartnerID: partnerID, PaymentTerms: platform.NullText(in.PaymentTerms), OrgUnitID: in.OrgUnitID, Active: in.Active,
		})
		if platform.Violates(err, "customers_org_unit_id_fkey") {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "sales.customer_created", audit.Ref{Type: customerType, ID: id},
			append([]audit.Change{{Field: "code", New: in.Code}, {Field: "partner_id", New: partnerID}}, customerChanges(CustomerFields{}, in.CustomerFields)...))
	})
	return id, err
}

// UpdateCustomer replaces a customer's fields and its partner's identity; moving it needs
// edit at both org units, changing the identity needs partner edit.
func (s *Service) UpdateCustomer(ctx context.Context, id int64, in CustomerUpdate) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		// Locked first, so the checks and the audit diff see what a concurrent edit left;
		// the partner row comes after.
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
		if err := s.d.Partner.Update(ctx, old.PartnerID, in.Partner); err != nil {
			return err
		}
		err = store.UpdateCustomer.Run(ctx, store.UpdateCustomerParams{
			ID: id, PaymentTerms: platform.NullText(in.PaymentTerms), OrgUnitID: in.OrgUnitID, Active: in.Active,
		})
		if platform.Violates(err, "customers_org_unit_id_fkey") {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "sales.customer_updated", audit.Ref{Type: customerType, ID: id}, customerChanges(customerFields(old), in.CustomerFields))
	})
}

func customerFields(c store.GetCustomerRow) CustomerFields {
	return CustomerFields{PaymentTerms: platform.TextPtr(c.PaymentTerms), OrgUnitID: c.OrgUnitID, Active: c.Active}
}

func partnerFields(c store.GetCustomerRow) partner.Fields {
	return partner.Fields{Name: c.Name, TaxCode: platform.TextPtr(c.TaxCode), Address: platform.TextPtr(c.Address),
		Phone: platform.TextPtr(c.Phone), Email: platform.TextPtr(c.Email), ContactName: platform.TextPtr(c.ContactName)}
}

// customerChanges are the customer role's fields; the identity is audited on the partner.
func customerChanges(old, n CustomerFields) []audit.Change {
	return []audit.Change{
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

// CustomerFields are the customer role's own fields; the identity is its partner's.
type CustomerFields struct {
	PaymentTerms *string `json:"payment_terms" maxLength:"1000" doc:"Prefilled on new quotations and orders"`
	OrgUnitID    int64   `json:"org_unit_id" doc:"Owning org unit: who may see the customer"`
	Active       bool    `json:"active"`
}

// NewCustomer names exactly one of an existing partner and a new partner's identity.
type NewCustomer struct {
	Code      string          `json:"code,omitempty" maxLength:"50" doc:"Empty or absent: issued from the sales.customer code series"`
	PartnerID *int64          `json:"partner_id,omitempty" doc:"An existing partner"`
	Partner   *partner.Fields `json:"partner,omitempty" doc:"A new partner's identity"`
	CustomerFields
}

// CustomerUpdate saves a customer and its partner's identity; an unchanged identity
// needs no partner edit.
type CustomerUpdate struct {
	CustomerFields
	Partner partner.Fields `json:"partner"`
}

type Customer struct {
	ID          int64          `json:"id"`
	Code        string         `json:"code"`
	PartnerID   int64          `json:"partner_id"`
	PartnerCode string         `json:"partner_code"`
	Partner     partner.Fields `json:"partner"`
	CustomerFields
	OrgUnitName string `json:"org_unit_name"`
	// edit, attach.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type CustomerFilter struct {
	Q      string `query:"q" maxLength:"100"`
	Active string `query:"active" enum:"true,false"`
	Sort   string `query:"sort" enum:"code,-code,name,-name" default:"code"`
	// The customers of one partner.
	PartnerID int64 `query:"partner_id"`
	platform.Paging
}

type CustomerListItem struct {
	ID          int64   `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	TaxCode     *string `json:"tax_code"`
	Phone       *string `json:"phone"`
	OrgUnitName string  `json:"org_unit_name"`
	Active      bool    `json:"active"`
	// Prefilled on a new quotation or order of the customer.
	PaymentTerms *string `json:"payment_terms"`
}

type CustomerList struct {
	Items []CustomerListItem `json:"items" nullable:"false"`
	Total int64              `json:"total"`
}

// CustomerCopy is the customer as the document copied it on its last draft save.
type CustomerCopy struct {
	ID          int64   `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	TaxCode     *string `json:"tax_code"`
	Address     *string `json:"address"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	ContactName *string `json:"contact_name"`
}
