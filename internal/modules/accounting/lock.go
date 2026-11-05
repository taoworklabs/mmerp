package accounting

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// BeforeLock reports the documents not yet in the ledger that moving a legal entity's
// lock date to until would freeze; record refuses the move while there are any.
func (s *Service) BeforeLock(ctx context.Context, legalEntity int64, until pgtype.Date) ([]record.HeldDoc, error) {
	if platform.ProductGate(ctx, "accounting", platform.ClassWrite) != nil {
		return nil, nil
	}
	docs, err := pendingIn(ctx, legalEntity, &until)
	if err != nil {
		return nil, err
	}
	out := make([]record.HeldDoc, len(docs))
	for i, d := range docs {
		t, _ := time.Parse(time.DateOnly, d.Date)
		out[i] = record.HeldDoc{DocType: d.DocType, ID: d.DocID, Number: d.Number, Date: pgtype.Date{Time: t, Valid: true}}
	}
	return out, nil
}
