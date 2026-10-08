// Package posting owns posting lines: the economic meaning of posted documents,
// captured once and voided, never edited, when a document is cancelled.
package posting

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting/internal/store"
)

type Service struct{ h Hooks }

func NewService(h Hooks) *Service { return &Service{h: h} }

// Record writes the posting lines of a document being posted, in the caller's transaction.
func (s *Service) Record(ctx context.Context, ref record.Ref, legalEntity int64, date string, lines []Line) error {
	p := store.InsertLinesParams{DocType: ref.Type, DocID: ref.ID, LegalEntityID: legalEntity, Date: platform.NullDate(&date)}
	for _, l := range lines {
		p.Kinds = append(p.Kinds, string(l.Kind))
		p.OrgUnitIds = append(p.OrgUnitIds, l.OrgUnitID)
		p.Amounts = append(p.Amounts, l.Amount)
	}
	if err := store.New(platform.DBFrom(ctx)).InsertLines(ctx, p); err != nil {
		return err
	}
	return s.changed(ctx, ref)
}

// Void marks the lines of a document being cancelled; they stay, with voided_at set.
func (s *Service) Void(ctx context.Context, ref record.Ref) error {
	if err := store.New(platform.DBFrom(ctx)).VoidLines(ctx, store.VoidLinesParams{DocType: ref.Type, DocID: ref.ID}); err != nil {
		return err
	}
	return s.changed(ctx, ref)
}

func (s *Service) changed(ctx context.Context, ref record.Ref) error {
	if s.h.OnChanged == nil {
		return nil
	}
	return s.h.OnChanged(ctx, ref)
}
