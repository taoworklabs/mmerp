// Package posting owns posting lines: the economic meaning of posted documents,
// captured once and voided, never edited, when a document is cancelled.
package posting

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/shared/posting/internal/store"
)

type Service struct{ h Hooks }

func NewService(h Hooks) *Service {
	data.RegisterConstraints(data.Constraints{"posting.recordings_doc_once": ErrAlreadyRecorded})
	return &Service{h: h}
}

// Record writes the posting lines of a document being posted, in the caller's transaction,
// under the document's legal entity and date. It refuses while earlier lines are not voided.
func (s *Service) Record(ctx context.Context, ref record.Ref, lines []Line) error {
	h, err := store.DocumentHeader.Run(ctx, store.DocumentHeaderParams{DocType: ref.Type, ID: ref.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := store.InsertRecording.Run(ctx, store.InsertRecordingParams{DocType: ref.Type, DocID: ref.ID}); err != nil {
		return err
	}
	p := store.InsertLinesParams{DocType: ref.Type, DocID: ref.ID, LegalEntityID: h.LegalEntityID, Date: h.Date}
	for _, l := range lines {
		p.Kinds = append(p.Kinds, string(l.Kind))
		p.OrgUnitIds = append(p.OrgUnitIds, l.OrgUnitID)
		p.Amounts = append(p.Amounts, l.Amount)
	}
	if err := store.InsertLines.Run(ctx, p); err != nil {
		return err
	}
	return s.changed(ctx, ref)
}

// Void marks the lines of a document being cancelled; they stay, with voided_at set.
func (s *Service) Void(ctx context.Context, ref record.Ref) error {
	if err := store.VoidRecording.Run(ctx, store.VoidRecordingParams{DocType: ref.Type, DocID: ref.ID}); err != nil {
		return err
	}
	if err := store.VoidLines.Run(ctx, store.VoidLinesParams{DocType: ref.Type, DocID: ref.ID}); err != nil {
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
