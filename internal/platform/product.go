package platform

import (
	"context"
	"net/http"
	"slices"
)

// Class is the action class every operation belongs to.
type Class int

const (
	ClassRead Class = iota
	ClassExport
	ClassWrite
)

type productsKey struct{}

// WithProducts puts the tenant's enabled products into ctx.
func WithProducts(ctx context.Context, products []string) context.Context {
	return context.WithValue(ctx, productsKey{}, products)
}

// ProductsFrom returns the tenant's enabled products.
func ProductsFrom(ctx context.Context) []string {
	p, _ := ctx.Value(productsKey{}).([]string)
	return p
}

// ProductGate decides every operation on a product: a disabled product stays
// readable and exportable, never writable. Core and shared modules have no product.
func ProductGate(ctx context.Context, product string, class Class) error {
	if product == "" || class != ClassWrite || slices.Contains(ProductsFrom(ctx), product) {
		return nil
	}
	return &Error{Status: http.StatusForbidden, Code: "product_not_enabled", Params: map[string]any{"product": product}}
}
