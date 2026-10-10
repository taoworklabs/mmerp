// Package sales owns customers, the item catalogue, quotations and sales orders.
package sales

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

const (
	customerType = "sales.customer"
	quoteType    = "sales.quote"
	orderType    = "sales.order"
)

// kind is what differs between quotations and orders; everything else is shared.
type kind struct {
	name, docType, prefix, view, edit string
}

var (
	quoteKind = kind{"quote", quoteType, "BG", PermQuoteView, PermQuoteEdit}
	orderKind = kind{"order", orderType, "DH", PermOrderView, PermOrderEdit}
)

type Service struct{ d Deps }

//go:embed i18n/*.json
var i18nFiles embed.FS

// loadTranslations adds the print labels and default text blocks to the catalog, once.
var loadTranslations = sync.OnceValue(func() error {
	sub, err := fs.Sub(i18nFiles, "i18n")
	if err != nil {
		return err
	}
	return platform.LoadTranslations(sub)
})

// NewService registers Sales roles, record types and print templates with core.
func NewService(d Deps) *Service {
	if err := loadTranslations(); err != nil {
		panic(err) // embedded files: broken only by a bad build
	}
	s := &Service{d: d}
	work := []string{PermCustomerView, PermCustomerEdit, PermQuoteView, PermQuoteEdit, PermOrderView, PermOrderEdit, PermItemView}
	d.IAM.RegisterRoles("sales", map[string][]string{
		// The same permissions under two names, so approval rules can route steps to managers.
		"staff":   work,
		"manager": work,
		"viewer":  {PermCustomerView, PermQuoteView, PermOrderView, PermItemView},
		// The catalogue and the approval rules apply to every org unit.
		"catalog_admin":  {PermItemView, PermItemManage},
		"approval_admin": {PermApprovalManage},
	}, "catalog_admin", "approval_admin")
	d.Record.Register(record.Type{Code: customerType, Product: "sales", Kind: record.Catalog, Can: s.canCustomer,
		HasData: func(ctx context.Context) (bool, error) { return store.New(platform.DBFrom(ctx)).AnyCatalog(ctx) }})
	for _, k := range []kind{quoteKind, orderKind} {
		d.Record.Register(record.Type{
			Code: k.docType, Product: "sales", Kind: record.Document, NumberPrefix: k.prefix,
			Fields:       []record.Field{{Key: "max_discount", Kind: record.Number, Label: "sales.doc.max_discount"}},
			Can:          s.canDoc(k),
			OnTransition: s.transition,
			BeforeSubmit: s.beforeSubmit,
		})
	}
	s.registerPrints()
	return s
}

func (s *Service) allowed(ctx context.Context, perm string, units ...int64) (bool, error) {
	sc, err := s.d.IAM.Scope(ctx, "sales", perm)
	if err != nil {
		return false, err
	}
	for _, u := range units {
		if !sc.Has(u) {
			return false, nil
		}
	}
	return true, nil
}

func (s *Service) require(ctx context.Context, perm string, units ...int64) error {
	ok, err := s.allowed(ctx, perm, units...)
	if err == nil && !ok {
		return platform.ErrForbidden
	}
	return err
}

// canDoc answers for quotations or orders by the document's org unit: reading, printing
// and seeing files need view; every write, attaching included, needs edit.
func (s *Service) canDoc(k kind) func(ctx context.Context, id int64, action record.Action) (bool, error) {
	return func(ctx context.Context, id int64, action record.Action) (bool, error) {
		h, err := store.New(platform.DBFrom(ctx)).GetHeader(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) || err == nil && h.Kind != k.name {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		switch action {
		case record.View, record.Export, record.Print, record.ViewFiles:
			return s.allowed(ctx, k.view, h.OrgUnitID)
		case record.Edit, record.Post, record.Cancel, record.Attach:
			return s.allowed(ctx, k.edit, h.OrgUnitID)
		}
		return false, nil
	}
}

// transition keeps posted documents to active customers, and a quotation with a live
// order from being cancelled; the quotation's row lock orders this against making the order.
func (s *Service) transition(ctx context.Context, d record.Doc, _ record.Status) error {
	q := store.New(platform.DBFrom(ctx))
	switch d.Status {
	case record.Posted:
		h, err := q.GetHeader(ctx, d.ID)
		if err != nil {
			return err
		}
		active, err := q.CustomerActive(ctx, h.CustomerID)
		if err != nil {
			return err
		}
		if !active {
			return ErrCustomerInactive
		}
	case record.Cancelled:
		if d.Type != quoteType {
			return nil
		}
		_, err := q.LiveOrder(ctx, pgtype.Int8{Int64: d.ID, Valid: true})
		if err == nil {
			return ErrQuoteHasOrder
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	return nil
}

// beforeSubmit copies the customer's current details onto a draft being sent, so what goes to
// approval, and what posts, is the customer as it is at sending; an inactive one keeps it a draft.
func (s *Service) beforeSubmit(ctx context.Context, d record.Doc) error {
	active, err := store.New(platform.DBFrom(ctx)).CopyCustomer(ctx, d.ID)
	if err != nil {
		return err
	}
	if !active {
		return ErrCustomerInactive
	}
	return nil
}

func (s *Service) today(ctx context.Context) (pgtype.Date, error) {
	tz, err := s.d.Setting.Get(ctx, setting.Timezone)
	if err != nil {
		return pgtype.Date{}, err
	}
	return store.New(platform.DBFrom(ctx)).Today(ctx, tz)
}
