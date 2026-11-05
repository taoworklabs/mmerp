package accounting

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/modules/accounting/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// Starts lists every legal entity with its accounting start date, if set.
func (s *Service) Starts(ctx context.Context) ([]Start, error) {
	sc, err := s.d.IAM.Scope(ctx, "accounting", PermView)
	if err != nil {
		return nil, err
	}
	if !sc.Any() {
		return nil, platform.ErrForbidden
	}
	rows, err := store.ListStarts.Run(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Start, 0, len(rows))
	for _, r := range rows {
		out = append(out, Start{LegalEntityID: r.LegalEntityID, StartDate: platform.DatePtr(r.StartDate), Fixed: r.Fixed})
	}
	return out, nil
}

// SetStart sets or (nil) clears a legal entity's start date.
func (s *Service) SetStart(ctx context.Context, legalEntity int64, date *string) error {
	return s.writeAccount(ctx, func(ctx context.Context) error {
		ok, err := store.IsLegalEntity.Run(ctx, legalEntity)
		if err != nil {
			return err
		}
		if !ok {
			return platform.ErrNotFound
		}
		// The period row before the starts row, as every document write takes them.
		locked, err := s.d.Record.SharePeriod(ctx, legalEntity)
		if err != nil {
			return err
		}
		// Locked before the check, so the first entry under the old date waits or is seen.
		if err := store.LockStart.Run(ctx, legalEntity); err != nil {
			return err
		}
		fixed, err := store.HasEntries.Run(ctx, legalEntity)
		if err != nil {
			return err
		}
		if fixed {
			return ErrStartDateFixed
		}
		// Documents recorded in the locked part would become pending where nothing can post.
		if date != nil && locked.Valid {
			until := locked.Time.Format(time.DateOnly)
			if *date <= until {
				in, err := store.RecordedBetween.Run(ctx, store.RecordedBetweenParams{LegalEntityID: legalEntity, FromDate: platform.NullDate(date), Until: locked})
				if err != nil {
					return err
				}
				if in {
					return &platform.Error{Status: ErrStartDateInLockedPeriod.Status, Code: ErrStartDateInLockedPeriod.Code, Params: map[string]any{"locked_until": until}}
				}
			}
		}
		if date == nil {
			err = store.ClearStart.Run(ctx, legalEntity)
		} else {
			err = store.SetStart.Run(ctx, store.SetStartParams{LegalEntityID: legalEntity, StartDate: platform.NullDate(date)})
		}
		if err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "accounting.start_set", map[string]any{"legal_entity_id": legalEntity, "start_date": date})
	})
}

// startOf is the start date of a legal entity; invalid when unset.
func startOf(ctx context.Context, legalEntity int64) (pgtype.Date, error) {
	d, err := store.StartOf.Run(ctx, legalEntity)
	if errors.Is(err, pgx.ErrNoRows) {
		return pgtype.Date{}, nil
	}
	return d, err
}
