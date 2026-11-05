package accounting

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/modules/accounting/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// Pending lists the documents of a legal entity not yet in the ledger.
func (s *Service) Pending(ctx context.Context, legalEntity int64) ([]PendingDoc, error) {
	if err := s.d.IAM.Require(ctx, "accounting", PermView, legalEntity); err != nil {
		return nil, err
	}
	return pendingIn(ctx, legalEntity, nil)
}

// PostPending queues a reconcile for every pending document and returns how many.
func (s *Service) PostPending(ctx context.Context, legalEntity int64) (int, error) {
	if err := s.d.IAM.Require(ctx, "accounting", PermPost, legalEntity); err != nil {
		return 0, err
	}
	n := 0
	err := platform.InTx(ctx, func(ctx context.Context) error {
		docs, err := pendingIn(ctx, legalEntity, nil)
		if err != nil {
			return err
		}
		n = len(docs)
		for _, d := range docs {
			if _, err := platform.Enqueue(ctx, reconcileArgs{DocType: d.DocType, DocID: d.DocID}); err != nil {
				return err
			}
		}
		return nil
	})
	return n, err
}

// pendingIn lists pending documents dated on or before until, when given.
func pendingIn(ctx context.Context, legalEntity int64, until *pgtype.Date) ([]PendingDoc, error) {
	p := store.PendingDocsParams{LegalEntityID: legalEntity}
	if until != nil {
		p.Until = *until
	}
	rows, err := store.PendingDocs.Run(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]PendingDoc, 0, len(rows))
	for _, r := range rows {
		reason := "queued"
		if r.Problem.Valid {
			reason = r.Problem.String
		}
		out = append(out, PendingDoc{DocType: r.DocType, DocID: r.DocID, Number: r.Number, Date: r.Date.Time.Format("2006-01-02"),
			LegalEntityID: r.LegalEntityID, Reason: reason, MissingKinds: r.MissingKinds})
	}
	return out, nil
}
