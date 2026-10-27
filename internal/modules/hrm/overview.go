package hrm

import (
	"context"
	"time"

	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
)

// MonthClose counts what is still open for the previous calendar month; a count is nil
// when the actor holds its permission nowhere.
type MonthClose struct {
	PeriodStart         string `json:"period_start"`
	OpenTimesheetUnits  *int   `json:"open_timesheet_units"`
	OpenPayrollEntities *int   `json:"open_payroll_entities"`
}

func (s *Service) MonthClose(ctx context.Context) (MonthClose, error) {
	today, err := s.d.Setting.Today(ctx)
	if err != nil {
		return MonthClose{}, err
	}
	start := time.Date(today.Time.Year(), today.Time.Month()-1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, -1)
	out := MonthClose{PeriodStart: start.Format(time.DateOnly)}
	ps, pe := pgDate(start), pgDate(end)

	sc, err := s.d.IAM.Scope(ctx, "hrm", PermTimesheetView)
	if err != nil {
		return out, err
	}
	if sc.Any() {
		units, err := store.OpenTimesheetUnits.Run(ctx, store.OpenTimesheetUnitsParams{PeriodStart: ps, PeriodEnd: pe})
		if err != nil {
			return out, err
		}
		out.OpenTimesheetUnits = countIn(units, sc.Has)
	}
	if sc, err = s.d.IAM.Scope(ctx, "hrm", PermPayrollView); err != nil {
		return out, err
	}
	if sc.Any() {
		entities, err := store.OpenPayrollEntities.Run(ctx, store.OpenPayrollEntitiesParams{PeriodStart: ps, PeriodEnd: pe})
		if err != nil {
			return out, err
		}
		out.OpenPayrollEntities = countIn(entities, sc.Has)
	}
	return out, nil
}

func countIn(ids []int64, has func(int64) bool) *int {
	n := 0
	for _, id := range ids {
		if has(id) {
			n++
		}
	}
	return &n
}
