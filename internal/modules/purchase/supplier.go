package purchase

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/purchase/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/partner"
)

// locateSupplier: a supplier is owned by its org unit.
func locateSupplier(ctx context.Context, id int64) (int64, error) {
	c, err := store.GetSupplier.Run(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, platform.ErrNotFound
	}
	return c.OrgUnitID, err
}

// SupplierActions: create when the actor may edit suppliers somewhere and Sales is on.
func (s *Service) SupplierActions(ctx context.Context) (PurchaseActions, error) {
	return s.typeActions(ctx, supplierType)
}

func (s *Service) typeActions(ctx context.Context, code string) (PurchaseActions, error) {
	acts, err := s.d.Record.TypeActions(ctx, code)
	return PurchaseActions{AllowedActions: acts}, err
}

// Suppliers lists the suppliers under the org units where the actor may view them.
func (s *Service) Suppliers(ctx context.Context, f SupplierFilter) (SupplierList, error) {
	out := SupplierList{Items: []SupplierListItem{}}
	sc, err := s.d.IAM.DataScope(ctx, "purchase", PermSupplierView)
	if err != nil {
		return out, err
	}
	page, err := supplierList.Run(ctx, supplierQuery(sc, f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, SupplierListItem{ID: r.ID, Code: r.Code, Name: r.Name, TaxCode: platform.TextPtr(r.TaxCode),
			Phone: platform.TextPtr(r.Phone), OrgUnitName: r.OrgUnitName, Active: r.Active, PaymentTerms: platform.TextPtr(r.PaymentTerms)})
	}
	return out, nil
}

// Supplier returns one supplier with its partner's identity; one the actor may not view
// does not exist.
func (s *Service) Supplier(ctx context.Context, id int64) (Supplier, error) {
	if err := s.d.Record.Visible(ctx, record.Ref{Type: supplierType, ID: id}, record.View); err != nil {
		return Supplier{}, err
	}
	c, err := store.GetSupplier.Run(ctx, id)
	if err != nil {
		return Supplier{}, err
	}
	actions, err := s.d.Record.Actions(ctx, record.Doc{Ref: record.Ref{Type: supplierType, ID: id}})
	if err != nil {
		return Supplier{}, err
	}
	// The identity is the partner's, so its own actions decide who edits it.
	partnerActions, err := s.d.Partner.Actions(ctx, c.PartnerID)
	return Supplier{ID: id, Code: c.Code, PartnerID: c.PartnerID, PartnerCode: c.PartnerCode, Partner: partnerFields(c),
		SupplierFields: supplierFields(c), OrgUnitName: c.OrgUnitName, AllowedActions: actions, PartnerActions: partnerActions}, err
}

// CreateSupplier adds a supplier owned by an org unit where the actor may edit suppliers,
// over an existing partner or one it creates in the same transaction.
func (s *Service) CreateSupplier(ctx context.Context, in NewSupplier) (int64, error) {
	if err := platform.ProductGate(ctx, "purchase", platform.ClassWrite); err != nil {
		return 0, err
	}
	if (in.PartnerID == nil) == (in.Partner == nil) {
		return 0, ErrPartnerRequired
	}
	if err := s.require(ctx, PermSupplierEdit, in.OrgUnitID); err != nil {
		return 0, err
	}
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		var partnerID int64
		var err error
		if in.PartnerID != nil {
			// Only a partner the actor may see.
			if err := s.d.Partner.Visible(ctx, *in.PartnerID); err != nil {
				return err
			}
			partnerID = *in.PartnerID
		} else if partnerID, err = s.d.Partner.Create(ctx, "", *in.Partner); err != nil {
			return err
		}
		if in.Code, err = s.d.Numbering.Code(ctx, supplierType, in.Code, supplierCodeTaken); err != nil {
			return err
		}
		id, err = store.CreateSupplier.Run(ctx, store.CreateSupplierParams{
			Code: in.Code, PartnerID: partnerID, PaymentTerms: platform.NullText(in.PaymentTerms), OrgUnitID: in.OrgUnitID, Active: in.Active,
		})
		if platform.Violates(err, "suppliers_org_unit_id_fkey") {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "purchase.supplier_created", audit.Ref{Type: supplierType, ID: id},
			append([]audit.Change{{Field: "code", New: in.Code}, {Field: "partner_id", New: partnerID}}, supplierChanges(SupplierFields{}, in.SupplierFields)...))
	})
	return id, err
}

// UpdateSupplier replaces a supplier's fields and, when given, its partner's identity;
// moving it needs edit at both org units, changing the identity needs partner edit.
func (s *Service) UpdateSupplier(ctx context.Context, id int64, in SupplierUpdate) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		// Locked first, so the checks and the audit diff see what a concurrent edit left;
		// the partner row comes after.
		if err := store.LockSupplier.Run(ctx, id); err != nil {
			return err
		}
		ref := record.Ref{Type: supplierType, ID: id}
		if err := s.d.Record.Visible(ctx, ref, record.View); err != nil {
			return err
		}
		old, err := store.GetSupplier.Run(ctx, id)
		if err != nil {
			return err
		}
		if err := s.d.Record.WriteGate(ctx, ref); err != nil {
			return err
		}
		if err := s.require(ctx, PermSupplierEdit, old.OrgUnitID, in.OrgUnitID); err != nil {
			return err
		}
		if in.Partner != nil {
			if err := s.d.Partner.Update(ctx, old.PartnerID, *in.Partner); err != nil {
				return err
			}
		}
		err = store.UpdateSupplier.Run(ctx, store.UpdateSupplierParams{
			ID: id, PaymentTerms: platform.NullText(in.PaymentTerms), OrgUnitID: in.OrgUnitID, Active: in.Active,
		})
		if platform.Violates(err, "suppliers_org_unit_id_fkey") {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "purchase.supplier_updated", audit.Ref{Type: supplierType, ID: id}, supplierChanges(supplierFields(old), in.SupplierFields))
	})
}

func supplierFields(c store.GetSupplierRow) SupplierFields {
	return SupplierFields{PaymentTerms: platform.TextPtr(c.PaymentTerms), OrgUnitID: c.OrgUnitID, Active: c.Active}
}

func partnerFields(c store.GetSupplierRow) partner.PartnerFields {
	return partner.PartnerFields{Name: c.Name, TaxCode: platform.TextPtr(c.TaxCode), Address: platform.TextPtr(c.Address),
		Phone: platform.TextPtr(c.Phone), Email: platform.TextPtr(c.Email), ContactName: platform.TextPtr(c.ContactName)}
}

// supplierChanges are the supplier role's fields; the identity is audited on the partner.
func supplierChanges(old, n SupplierFields) []audit.Change {
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

// supplierCodeTaken reports whether a code is used, for numbering to skip it.
func supplierCodeTaken(ctx context.Context, code string) (bool, error) {
	return store.SupplierCodeTaken.Run(ctx, code)
}

// SupplierFields are the supplier role's own fields; the identity is its partner's.
type SupplierFields struct {
	PaymentTerms *string `json:"payment_terms" maxLength:"1000" doc:"Prefilled on new purchase orders"`
	OrgUnitID    int64   `json:"org_unit_id" doc:"Owning org unit: who may see the supplier"`
	Active       bool    `json:"active"`
}

// NewSupplier names exactly one of an existing partner and a new partner's identity.
type NewSupplier struct {
	Code      string                 `json:"code,omitempty" maxLength:"50" doc:"Empty or absent: issued from the purchase.supplier code series"`
	PartnerID *int64                 `json:"partner_id,omitempty" doc:"An existing partner"`
	Partner   *partner.PartnerFields `json:"partner,omitempty" doc:"A new partner's identity"`
	SupplierFields
}

// SupplierUpdate saves a supplier and, when given, its partner's identity; an unchanged
// identity needs no partner edit.
type SupplierUpdate struct {
	SupplierFields
	Partner *partner.PartnerFields `json:"partner,omitempty" doc:"Absent: the identity is left as it is"`
}

type Supplier struct {
	ID          int64                 `json:"id"`
	Code        string                `json:"code"`
	PartnerID   int64                 `json:"partner_id"`
	PartnerCode string                `json:"partner_code"`
	Partner     partner.PartnerFields `json:"partner"`
	SupplierFields
	OrgUnitName string `json:"org_unit_name"`
	// edit, attach.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
	// The partner's allowed_actions: edit means the identity may change.
	PartnerActions []string `json:"partner_actions" nullable:"false"`
}

type SupplierFilter struct {
	Q      string `query:"q" maxLength:"100"`
	Active string `query:"active" enum:"true,false"`
	Sort   string `query:"sort" enum:"code,-code,name,-name" default:"code"`
	// The suppliers of one partner.
	PartnerID int64 `query:"partner_id"`
	platform.Paging
}

type SupplierListItem struct {
	ID          int64   `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	TaxCode     *string `json:"tax_code"`
	Phone       *string `json:"phone"`
	OrgUnitName string  `json:"org_unit_name"`
	Active      bool    `json:"active"`
	// Prefilled on a new purchase order of the supplier.
	PaymentTerms *string `json:"payment_terms"`
}

type SupplierList struct {
	Items []SupplierListItem `json:"items" nullable:"false"`
	Total int64              `json:"total"`
}

// SupplierCopy is the supplier as the document copied it on its last draft save.
type SupplierCopy struct {
	ID          int64   `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	TaxCode     *string `json:"tax_code"`
	Address     *string `json:"address"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	ContactName *string `json:"contact_name"`
}
