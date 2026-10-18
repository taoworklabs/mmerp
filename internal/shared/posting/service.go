// Package posting owns posting lines: the economic meaning of posted documents,
// captured once and voided, never edited, when a document is cancelled.
package posting

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting/internal/store"
)

type Service struct{ h Hooks }

func NewService(h Hooks) *Service { return &Service{h: h} }

// Record writes the posting lines of a document being posted, in the caller's transaction,
// under the document's legal entity and date. It refuses while earlier lines are not voided.
func (s *Service) Record(ctx context.Context, ref record.Ref, lines []Line) error {
	q := store.New(platform.DBFrom(ctx))
	h, err := q.DocumentHeader(ctx, store.DocumentHeaderParams{DocType: ref.Type, ID: ref.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.ErrNotFound
	}
	if err != nil {
		return err
	}
	err = q.InsertRecording(ctx, store.InsertRecordingParams{DocType: ref.Type, DocID: ref.ID})
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return ErrAlreadyRecorded
	}
	if err != nil {
		return err
	}
	p := store.InsertLinesParams{DocType: ref.Type, DocID: ref.ID, LegalEntityID: h.LegalEntityID, Date: h.Date}
	for _, l := range lines {
		p.Kinds = append(p.Kinds, string(l.Kind))
		p.OrgUnitIds = append(p.OrgUnitIds, l.OrgUnitID)
		p.Amounts = append(p.Amounts, l.Amount)
	}
	if err := q.InsertLines(ctx, p); err != nil {
		return err
	}
	return s.changed(ctx, ref)
}

// Void marks the lines of a document being cancelled; they stay, with voided_at set.
func (s *Service) Void(ctx context.Context, ref record.Ref) error {
	q := store.New(platform.DBFrom(ctx))
	if err := q.VoidRecording(ctx, store.VoidRecordingParams{DocType: ref.Type, DocID: ref.ID}); err != nil {
		return err
	}
	if err := q.VoidLines(ctx, store.VoidLinesParams{DocType: ref.Type, DocID: ref.ID}); err != nil {
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
