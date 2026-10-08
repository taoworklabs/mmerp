// Package numbering hands out document numbers.
package numbering

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/core/numbering/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type Service struct{}

func NewService() *Service { return &Service{} }

// Next returns the next number of a document type for a legal entity and year.
// Call it inside the transaction that creates the document.
func (s *Service) Next(ctx context.Context, docType string, legalEntity int64, year int32) (int64, error) {
	return store.New(platform.DBFrom(ctx)).Next(ctx, store.NextParams{DocType: docType, LegalEntityID: legalEntity, Year: year})
}
