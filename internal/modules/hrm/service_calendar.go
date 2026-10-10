package hrm

import (
	"context"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// defaultOffDays apply before a legal entity's first work week version: Saturday and Sunday.
var defaultOffDays = []int{int(time.Sunday), int(time.Saturday)}

// calendar is a legal entity's work calendar over a range of days.
type calendar struct {
	Weeks    []WorkWeek        `json:"weeks"`
	Holidays map[string]string `json:"holidays"`
}

// workCalendar reads the work weeks and the holidays of [from, to] (YYYY-MM-DD).
func workCalendar(ctx context.Context, legalEntity int64, from, to string) (calendar, error) {
	q := store.New(platform.DBFrom(ctx))
	c := calendar{Holidays: map[string]string{}}
	weeks, err := q.WorkWeeks(ctx, legalEntity)
	if err != nil {
		return c, err
	}
	for _, w := range weeks {
		c.Weeks = append(c.Weeks, workWeek(w))
	}
	days, err := q.Holidays(ctx, store.HolidaysParams{LegalEntityID: legalEntity, FromDate: platform.NullDate(&from), ToDate: platform.NullDate(&to)})
	for _, h := range days {
		c.Holidays[*platform.DatePtr(h.Date)] = h.Name
	}
	return c, err
}

func workWeek(w store.WorkWeeksRow) WorkWeek {
	out := WorkWeek{EffectiveFrom: *platform.DatePtr(w.EffectiveFrom), OffDays: []int{}}
	for _, d := range w.OffDays {
		out.OffDays = append(out.OffDays, int(d))
	}
	return out
}

// weeklyOff reports whether d is a weekly day off under the version in force then.
func (c calendar) weeklyOff(d time.Time) bool {
	off := defaultOffDays
	for _, w := range c.Weeks {
		if w.EffectiveFrom <= d.Format(time.DateOnly) {
			off = w.OffDays
		}
	}
	return slices.Contains(off, int(d.Weekday()))
}

func (c calendar) holiday(d time.Time) bool {
	_, ok := c.Holidays[d.Format(time.DateOnly)]
	return ok
}

// standardDays is the month's normal working days, holidays on working days included, at most 26.
func (c calendar) standardDays(start, end time.Time) int {
	n := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if !c.weeklyOff(d) {
			n++
		}
	}
	return min(n, 26)
}

// offDays lists the weekly days off and the holidays of [start, end].
func (c calendar) offDays(start, end time.Time) []string {
	out := []string{}
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if c.weeklyOff(d) || c.holiday(d) {
			out = append(out, d.Format(time.DateOnly))
		}
	}
	return out
}

// WorkCalendar returns a legal entity's work weeks and its holidays of year.
func (s *Service) WorkCalendar(ctx context.Context, legalEntity int64, year int) (WorkCalendar, error) {
	out := WorkCalendar{WorkWeeks: []WorkWeek{}, Holidays: []Holiday{}, AllowedActions: []string{}}
	if err := s.checkLegalEntity(ctx, legalEntity); err != nil {
		return out, err
	}
	from, to := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.DateOnly), time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
	c, err := workCalendar(ctx, legalEntity, from, to)
	if err != nil {
		return out, err
	}
	if c.Weeks != nil {
		out.WorkWeeks = c.Weeks
	}
	for _, d := range slices.Sorted(maps.Keys(c.Holidays)) {
		out.Holidays = append(out.Holidays, Holiday{Date: d, Name: c.Holidays[d]})
	}
	if ok, err := s.canManageCalendar(ctx, legalEntity); err != nil {
		return out, err
	} else if ok {
		out.AllowedActions = append(out.AllowedActions, "manage")
	}
	return out, nil
}

func (s *Service) canManageCalendar(ctx context.Context, legalEntity int64) (bool, error) {
	ok, err := s.allowed(ctx, PermCalendarManage, legalEntity)
	return ok && platform.ProductGate(ctx, "hrm", platform.ClassWrite) == nil, err
}

// SaveWorkWeek sets the weekly days off from a date on. It changes payroll sources from
// that date to the next version, so it is refused over a closed payroll period.
func (s *Service) SaveWorkWeek(ctx context.Context, legalEntity int64, w WorkWeek) error {
	days := make([]int16, 0, len(w.OffDays))
	for _, d := range w.OffDays {
		if d < 0 || d > 6 || slices.Contains(days, int16(d)) {
			return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_request", Params: map[string]any{"fields": []string{"off_days"}}}
		}
		days = append(days, int16(d))
	}
	slices.Sort(days)
	return s.writeWorkWeek(ctx, legalEntity, w.EffectiveFrom, func(ctx context.Context, from pgtype.Date) error {
		return store.New(platform.DBFrom(ctx)).SaveWorkWeek(ctx, store.SaveWorkWeekParams{LegalEntityID: legalEntity, EffectiveFrom: from, OffDays: days})
	}, map[string]any{"off_days": days})
}

// DeleteWorkWeek removes a version; the one before it applies again.
func (s *Service) DeleteWorkWeek(ctx context.Context, legalEntity int64, effectiveFrom string) error {
	return s.writeWorkWeek(ctx, legalEntity, effectiveFrom, func(ctx context.Context, from pgtype.Date) error {
		n, err := store.New(platform.DBFrom(ctx)).DeleteWorkWeek(ctx, store.DeleteWorkWeekParams{LegalEntityID: legalEntity, EffectiveFrom: from})
		if err == nil && n == 0 {
			return platform.ErrNotFound
		}
		return err
	}, map[string]any{"deleted": true})
}

func (s *Service) writeWorkWeek(ctx context.Context, legalEntity int64, effectiveFrom string, write func(context.Context, pgtype.Date) error, data map[string]any) error {
	from := platform.NullDate(&effectiveFrom)
	if !from.Valid {
		return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_request", Params: map[string]any{"fields": []string{"effective_from"}}}
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.requireCalendar(ctx, legalEntity); err != nil {
			return err
		}
		q := store.New(platform.DBFrom(ctx))
		weeks, err := q.WorkWeeks(ctx, legalEntity)
		if err != nil {
			return err
		}
		var to *string
		for _, w := range weeks {
			if w.EffectiveFrom.Time.After(from.Time) {
				to = new(w.EffectiveFrom.Time.AddDate(0, 0, -1).Format(time.DateOnly))
				break
			}
		}
		if err := checkPayrollPeriods(ctx, legalEntity, effectiveFrom, to); err != nil {
			return err
		}
		if err := write(ctx, from); err != nil {
			return err
		}
		data["legal_entity_id"], data["effective_from"] = legalEntity, effectiveFrom
		return s.d.Audit.Record(ctx, "hrm.work_week_changed", data)
	})
}

// SaveHoliday adds or renames a holiday; refused in a closed payroll period.
func (s *Service) SaveHoliday(ctx context.Context, legalEntity int64, h Holiday) error {
	return s.writeHoliday(ctx, legalEntity, h.Date, func(ctx context.Context, date pgtype.Date) error {
		return store.New(platform.DBFrom(ctx)).SaveHoliday(ctx, store.SaveHolidayParams{LegalEntityID: legalEntity, Date: date, Name: h.Name})
	}, map[string]any{"name": h.Name})
}

func (s *Service) DeleteHoliday(ctx context.Context, legalEntity int64, date string) error {
	return s.writeHoliday(ctx, legalEntity, date, func(ctx context.Context, d pgtype.Date) error {
		n, err := store.New(platform.DBFrom(ctx)).DeleteHoliday(ctx, store.DeleteHolidayParams{LegalEntityID: legalEntity, Date: d})
		if err == nil && n == 0 {
			return platform.ErrNotFound
		}
		return err
	}, map[string]any{"deleted": true})
}

func (s *Service) writeHoliday(ctx context.Context, legalEntity int64, date string, write func(context.Context, pgtype.Date) error, data map[string]any) error {
	d := platform.NullDate(&date)
	if !d.Valid {
		return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_request", Params: map[string]any{"fields": []string{"date"}}}
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.requireCalendar(ctx, legalEntity); err != nil {
			return err
		}
		if err := checkPayrollPeriods(ctx, legalEntity, date, &date); err != nil {
			return err
		}
		if err := write(ctx, d); err != nil {
			return err
		}
		data["legal_entity_id"], data["date"] = legalEntity, date
		return s.d.Audit.Record(ctx, "hrm.holiday_changed", data)
	})
}

func (s *Service) requireCalendar(ctx context.Context, legalEntity int64) error {
	if err := s.checkLegalEntity(ctx, legalEntity); err != nil {
		return err
	}
	return s.require(ctx, PermCalendarManage, legalEntity)
}

func (s *Service) checkLegalEntity(ctx context.Context, id int64) error {
	le, err := s.d.IAM.LegalEntityOf(ctx, id)
	if err != nil || le != id {
		return platform.OrErr(err, ErrNotLegalEntity)
	}
	return nil
}
