// Package partner owns partners: the companies and people every product deals with.
// Products add their roles (Sales: customer) on top.
package partner

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/shared/partner/internal/store"
)

const partnerType = "partner.partner"

type Service struct{ d Deps }

// NewService registers the partner roles, code series and record type; no product gates them.
func NewService(d Deps) *Service {
	data.RegisterConstraints(data.Constraints{
		"partner.partners_code_key": ErrCodeTaken,
		// Only a race reaches the index; the check before writing names the partner.
		"partner.partners_tax_code_key": ErrTaxCodeTaken,
	})
	d.IAM.RegisterRoles("partner", map[string][]string{
		"viewer": {PermView},
		"editor": {PermView, PermEdit},
	})
	d.Numbering.RegisterSeries(partnerType, "", "DT")
	// A partner's files are as open as the partner.
	d.Record.Register(record.Type{Code: partnerType, Product: "partner", Kind: record.Catalog, Shared: true, Anywhere: true,
		Perms:  record.Perms{View: PermView, Edit: PermEdit, ViewFiles: PermView, Attach: PermEdit},
		Locate: locate})
	return &Service{d: d}
}

func locate(ctx context.Context, id int64) (int64, error) {
	_, err := store.GetPartner.Run(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, platform.ErrNotFound
	}
	return 0, err
}

// List lists every partner to anyone who may view partners somewhere.
func (s *Service) List(ctx context.Context, f PartnerFilter) (PartnerList, error) {
	out := PartnerList{Items: []PartnerListItem{}}
	if err := s.requireAnywhere(ctx, PermView); err != nil {
		return out, err
	}
	page, err := partnerList.Run(ctx, partnerQuery(f))
	if err != nil {
		return out, err
	}
	out.Total = page.Total
	for _, r := range page.Items {
		out.Items = append(out.Items, PartnerListItem{ID: r.ID, Code: r.Code, PartnerFields: fields(r)})
	}
	return out, nil
}

// Visible fails with not found when the partner is missing or the actor may not view it.
func (s *Service) Visible(ctx context.Context, id int64) error {
	return s.d.Record.Visible(ctx, record.Ref{Type: partnerType, ID: id}, record.View)
}

// Get returns one partner; one the actor may not view does not exist.
func (s *Service) Get(ctx context.Context, id int64) (Partner, error) {
	if err := s.Visible(ctx, id); err != nil {
		return Partner{}, err
	}
	r, err := store.GetPartner.Run(ctx, id)
	if err != nil {
		return Partner{}, err
	}
	actions, err := s.d.Record.Actions(ctx, record.Doc{Ref: record.Ref{Type: partnerType, ID: id}})
	return Partner{ID: r.ID, Code: r.Code, PartnerFields: fields(r), AllowedActions: actions}, err
}

// Create adds a partner, in the caller's transaction if there is one.
func (s *Service) Create(ctx context.Context, code string, f PartnerFields) (int64, error) {
	if err := s.requireAnywhere(ctx, PermEdit); err != nil {
		return 0, err
	}
	f = clean(f)
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		if err := taxCodeFree(ctx, 0, f.TaxCode); err != nil {
			return err
		}
		var err error
		if code, err = s.d.Numbering.Code(ctx, partnerType, code, store.PartnerCodeTaken.Run); err != nil {
			return err
		}
		id, err = store.CreatePartner.Run(ctx, store.CreatePartnerParams{Code: code, Name: f.Name, TaxCode: platform.NullText(f.TaxCode),
			Address: platform.NullText(f.Address), Phone: platform.NullText(f.Phone), Email: platform.NullText(f.Email), ContactName: platform.NullText(f.ContactName)})
		if err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "partner.partner_created", audit.Ref{Type: partnerType, ID: id},
			append([]audit.Change{{Field: "code", New: code}}, changes(PartnerFields{}, f)...))
	})
	return id, err
}

// Update replaces a partner's identity, in the caller's transaction if there is one. An
// unchanged identity saves without partner edit, so a product may save its role alone.
func (s *Service) Update(ctx context.Context, id int64, f PartnerFields) error {
	f = clean(f)
	return platform.InTx(ctx, func(ctx context.Context) error {
		r, err := store.LockPartner.Run(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		old := fields(r)
		if reflect.DeepEqual(old, f) {
			return nil
		}
		if err := s.requireAnywhere(ctx, PermEdit); err != nil {
			return err
		}
		if err := taxCodeFree(ctx, id, f.TaxCode); err != nil {
			return err
		}
		if err := store.UpdatePartner.Run(ctx, store.UpdatePartnerParams{ID: id, Name: f.Name, TaxCode: platform.NullText(f.TaxCode),
			Address: platform.NullText(f.Address), Phone: platform.NullText(f.Phone), Email: platform.NullText(f.Email), ContactName: platform.NullText(f.ContactName)}); err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "partner.partner_updated", audit.Ref{Type: partnerType, ID: id}, changes(old, f))
	})
}

// Edit is Update for the partner routes: it always needs partner edit, so an unchanged
// identity tells nobody else whether the partner exists.
func (s *Service) Edit(ctx context.Context, id int64, f PartnerFields) error {
	if err := s.requireAnywhere(ctx, PermEdit); err != nil {
		return err
	}
	return s.Update(ctx, id, f)
}

func (s *Service) requireAnywhere(ctx context.Context, perm string) error {
	sc, err := s.d.IAM.Scope(ctx, "partner", perm)
	if err == nil && !sc.Any() {
		return platform.ErrForbidden
	}
	return err
}

// taxCodeFree refuses a tax code another partner holds, naming that partner. Checked
// before writing, so a caller's transaction stays usable.
func taxCodeFree(ctx context.Context, self int64, tax *string) error {
	if tax == nil {
		return nil
	}
	other, err := store.PartnerByTaxCode.Run(ctx, *tax)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && other == self {
		return nil
	}
	if err != nil {
		return err
	}
	return &platform.Error{Status: ErrTaxCodeTaken.Status, Code: ErrTaxCodeTaken.Code, Params: map[string]any{"partner_id": other}}
}

// clean trims the identity; a blank optional field is none.
func clean(f PartnerFields) PartnerFields {
	f.Name = strings.TrimSpace(f.Name)
	for _, p := range []**string{&f.TaxCode, &f.Address, &f.Phone, &f.Email, &f.ContactName} {
		if *p == nil {
			continue
		}
		if v := strings.TrimSpace(**p); v != "" {
			*p = &v
		} else {
			*p = nil
		}
	}
	return f
}

func fields(r store.PartnerPartner) PartnerFields {
	return PartnerFields{Name: r.Name, TaxCode: platform.TextPtr(r.TaxCode), Address: platform.TextPtr(r.Address),
		Phone: platform.TextPtr(r.Phone), Email: platform.TextPtr(r.Email), ContactName: platform.TextPtr(r.ContactName)}
}

func changes(old, n PartnerFields) []audit.Change {
	return []audit.Change{
		{Field: "name", Old: old.Name, New: n.Name},
		{Field: "tax_code", Old: old.TaxCode, New: n.TaxCode},
		{Field: "address", Old: old.Address, New: n.Address},
		{Field: "phone", Old: old.Phone, New: n.Phone},
		{Field: "email", Old: old.Email, New: n.Email},
		{Field: "contact_name", Old: old.ContactName, New: n.ContactName},
	}
}
