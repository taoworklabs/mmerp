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
