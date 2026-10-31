package iam

import "context"

// TreeHook checks the org-unit tree after a write, in the same transaction;
// an error rolls the write back. record uses it to keep documents in their legal entity.
type TreeHook interface {
	TreeChanged(ctx context.Context) error
}

// DataProducts names the products that have at least one record; record provides it.
// Unset, no product counts as having data.
type DataProducts interface {
	ProductsWithData(ctx context.Context) ([]string, error)
}

// LegalEntityCreated runs when a unit becomes a company (created as one or changed into one),
// in the same transaction; an error rolls the write back. stock gives the company its default warehouse.
type LegalEntityCreated func(ctx context.Context, legalEntity int64) error
