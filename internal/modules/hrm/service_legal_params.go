package hrm

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// legalParamKeys are every legal parameter a payroll reads; a payroll needs a version of each.
var legalParamKeys = []string{
	"base_salary", "min_wage_region_1", "min_wage_region_2", "min_wage_region_3", "min_wage_region_4",
	"si_employee", "hi_employee", "ui_employee", "si_employer", "hi_employer", "ui_employer", "union_employer",
	"insurance_cap_multiplier", "personal_deduction", "dependent_deduction", "pit_brackets",
	"ot_exempt_hours_month", "ot_exempt_hours_year", "ot_hours_per_day", "insurance_skip_days",
}

// bracket is a step of the progressive income tax: Rate up to Upper (nil: no limit).
type bracket struct {
	Upper *decimal.Decimal
	Rate  decimal.Decimal
}

func parseBrackets(v string) ([]bracket, bool) {
	var raw [][2]*json.Number
	if err := json.Unmarshal([]byte(v), &raw); err != nil || len(raw) == 0 {
		return nil, false
	}
	out := make([]bracket, len(raw))
	var prev decimal.Decimal
	for i, r := range raw {
		rate, ok := parseRaw(r[1])
		if !ok || (r[0] == nil) != (i == len(raw)-1) {
			return nil, false
		}
		out[i].Rate = rate
		if r[0] != nil {
			upper, ok := parseRaw(r[0])
			if !ok || !upper.GreaterThan(prev) {
				return nil, false
			}
			out[i].Upper, prev = &upper, upper
		}
	}
	return out, true
}

// parseRaw reads a number kept as text, so no float is involved.
func parseRaw[T ~string](s *T) (decimal.Decimal, bool) {
	if s == nil {
		return decimal.Decimal{}, false
	}
	d, ok := record.ParseNumber(string(*s))
	return d, ok && !d.IsNegative()
}

// checkLegalParam refuses an unknown key or a value of the wrong shape.
func checkLegalParam(key, value string) error {
	if !slices.Contains(legalParamKeys, key) {
		return platform.ErrNotFound
	}
	ok := false
	if key == "pit_brackets" {
		_, ok = parseBrackets(value)
	} else {
		_, ok = parseRaw(&value)
	}
	if !ok {
		return ErrLegalParam
	}
	return nil
}

// LegalParams lists every version of every legal parameter.
func (s *Service) LegalParams(ctx context.Context) (LegalParams, error) {
	out := LegalParams{Items: []LegalParam{}, AllowedActions: []string{}}
	rows, err := store.New(platform.DBFrom(ctx)).ListLegalParams(ctx)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Items = append(out.Items, LegalParam{Key: r.Key, EffectiveFrom: *platform.DatePtr(r.EffectiveFrom), Value: r.Value})
	}
	sc, err := s.d.IAM.Scope(ctx, "hrm", PermLegalParamManage)
	if sc.All && platform.ProductGate(ctx, "hrm", platform.ClassWrite) == nil {
		out.AllowedActions = append(out.AllowedActions, "manage")
	}
	return out, err
}

// SaveLegalParam adds a version of a legal parameter or changes one. It is a payroll source
// from its date to the next version, for every legal entity, so it is refused over any
// closed payroll period.
func (s *Service) SaveLegalParam(ctx context.Context, p LegalParam) error {
	if err := checkLegalParam(p.Key, p.Value); err != nil {
		return err
	}
	from := platform.NullDate(&p.EffectiveFrom)
	if !from.Valid {
		return ErrLegalParam
	}
	if sc, err := s.d.IAM.Scope(ctx, "hrm", PermLegalParamManage); err != nil || !sc.All {
		return platform.OrErr(err, platform.ErrForbidden)
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		rows, err := q.ListLegalParams(ctx)
		if err != nil {
			return err
		}
		var to *string
		for _, r := range rows {
			if r.Key == p.Key && r.EffectiveFrom.Time.After(from.Time) {
				to = new(r.EffectiveFrom.Time.AddDate(0, 0, -1).Format(time.DateOnly))
				break
			}
		}
		entities, err := q.LegalEntities(ctx)
		if err != nil {
			return err
		}
		for _, le := range entities {
			if err := checkPayrollPeriods(ctx, le, p.EffectiveFrom, to); err != nil {
				return err
			}
		}
		old, err := q.GetLegalParam(ctx, store.GetLegalParamParams{Key: p.Key, EffectiveFrom: from})
		if errors.Is(err, pgx.ErrNoRows) {
			old, err = "", nil
		}
		if err != nil {
			return err
		}
		if err := q.SaveLegalParam(ctx, store.SaveLegalParamParams{Key: p.Key, EffectiveFrom: from, Value: p.Value}); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "hrm.legal_param_changed", map[string]any{"key": p.Key, "effective_from": p.EffectiveFrom, "old": old, "new": p.Value})
	})
}
