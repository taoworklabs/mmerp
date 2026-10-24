//go:build lintfixture

// Package store stands in for a module's generated internal/store package: scripts
// cannot import internal/**/internal/store (Go's internal rule), so the store.New
// fixture gets its own throwaway package with the same shape instead.
package store

import "github.com/taoworklabs/mmerp/internal/platform"

// Queries is a stand-in for a module's generated Queries type.
type Queries struct{}

// New mirrors a module's generated store.New(conn).
func New(c platform.Conn) *Queries { return &Queries{} }
