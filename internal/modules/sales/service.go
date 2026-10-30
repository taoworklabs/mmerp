// Package sales owns customers, quotations, sales orders and deliveries.
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
	"github.com/taoworklabs/mmerp/internal/modules/sales/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

const (
	customerType = "sales.customer"
	quoteType    = "sales.quote"
	orderType    = "sales.order"

	actionCreateOrder record.Action = "create_order"
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
	data.RegisterConstraints(data.Constraints{
		"sales.customers_code_key":             ErrCustomerCodeTaken,
		"sales.customers_partner_org_unit_key": ErrCustomerExists,
	})
	work := []string{PermCustomerView, PermCustomerEdit, PermQuoteView, PermQuoteEdit, PermOrderView, PermOrderEdit}
	d.IAM.RegisterRoles("sales", map[string][]string{
		// The same permissions under two names, so approval rules can route steps to managers.
		"staff":   work,
		"manager": work,
		"viewer":  {PermCustomerView, PermQuoteView, PermOrderView},
		// The approval rules apply to every org unit.
		"approval_admin": {PermApprovalManage},
	}, "approval_admin")
	d.Numbering.RegisterSeries(customerType, "sales", "KH")
	// A customer's files are as open as the customer.
	d.Record.Register(record.Type{Code: customerType, Product: "sales", Kind: record.Catalog,
		Perms:   record.Perms{View: PermCustomerView, Edit: PermCustomerEdit, ViewFiles: PermCustomerView, Attach: PermCustomerEdit},
		Locate:  locateCustomer,
		HasData: func(ctx context.Context) (bool, error) { return store.AnyCatalog.Run(ctx) }})
	for _, k := range []kind{quoteKind, orderKind} {
		var actions map[record.Action]record.ModuleAction
		if k == quoteKind {
			actions = map[record.Action]record.ModuleAction{actionCreateOrder: s.runCreateOrder}
		}
		d.Record.Register(record.Type{
			Code: k.docType, Product: "sales", Kind: record.Document, NumberPrefix: k.prefix,
			Fields: []record.Field{
				{Key: "max_discount", Kind: record.Number, Label: "sales.doc.max_discount"},
				{Key: "items", Kind: record.Set, Label: "sales.doc.items", Search: s.itemOptions, Labels: s.itemLabels},
			},
			// Reading, printing and seeing files need view; every write, attaching included, needs edit.
			Perms:         record.Perms{View: k.view, Edit: k.edit, ViewFiles: k.view, Attach: k.edit},
			OnTransition:  s.transition,
			BeforeSubmit:  s.beforeSubmit,
			OnDelete:      deleteDoc,
			ModuleActions: actions,
		})
	}
	s.registerDelivery()
	s.registerPrints()
	return s
}

func (s *Service) require(ctx context.Context, perm string, unit int64, more ...int64) error {
	return s.d.IAM.Require(ctx, "sales", perm, unit, more...)
}

// transition keeps posted documents to active customers, and a quotation with a live
// order from being cancelled; the quotation's row lock orders this against making the order.
func (s *Service) transition(ctx context.Context, d record.Doc, _ record.Status) error {
	switch d.Status {
	case record.Posted:
		h, err := store.GetHeader.Run(ctx, d.ID)
		if err != nil {
			return err
		}
		active, err := store.CustomerActive.Run(ctx, h.CustomerID)
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
		_, err := store.LiveOrder.Run(ctx, pgtype.Int8{Int64: d.ID, Valid: true})
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
	active, err := store.CopyCustomer.Run(ctx, d.ID)
	if err != nil {
		return err
	}
	if !active {
		return ErrCustomerInactive
	}
	return nil
}
