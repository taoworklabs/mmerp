package hrm

import (
	"context"
	"net/http"

	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// checkPayrollPeriods refuses a change to a source document whose affected range
// [from, to] (to nil: open-ended) overlaps a closed payroll period of the legal entity.
// It holds the legal entity's payroll lock FOR SHARE until commit, so the set of
// closed periods cannot change under the caller.
func checkPayrollPeriods(ctx context.Context, legalEntity int64, from string, to *string) error {
	q := store.New(platform.DBFrom(ctx))
	if err := q.EnsurePayrollLock(ctx, legalEntity); err != nil {
		return err
	}
	if err := q.SharePayrollLock(ctx, legalEntity); err != nil {
		return err
	}
	rows, err := q.ClosedPeriods(ctx, store.ClosedPeriodsParams{LegalEntityID: legalEntity, FromDate: platform.NullDate(&from), ToDate: platform.NullDate(to)})
	if err != nil || len(rows) == 0 {
		return err
	}
	periods := make([]Period, len(rows))
	for i, r := range rows {
		periods[i] = Period{Start: *platform.DatePtr(r.PeriodStart), End: *platform.DatePtr(r.PeriodEnd)}
	}
	return &platform.Error{Status: http.StatusConflict, Code: "payroll_period_closed", Params: map[string]any{"periods": periods}}
}
