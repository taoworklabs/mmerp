package accounting

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/modules/accounting/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

// Rules lists one rule per declared posting kind, then any other kind that has a rule.
func (s *Service) Rules(ctx context.Context) ([]Rule, error) {
	sc, err := s.d.IAM.Scope(ctx, "accounting", PermView)
	if err != nil {
		return nil, err
	}
	if !sc.Any() {
		return nil, platform.ErrForbidden
	}
	rows, err := store.ListRules.Run(ctx)
	if err != nil {
		return nil, err
	}
	stored := make(map[string]Rule, len(rows))
	for _, r := range rows {
		d, _ := posting.Declared(posting.Kind(r.Kind))
		stored[r.Kind] = Rule{Kind: r.Kind, AccountID: &r.AccountID, Side: &r.Side, ByPartner: d.ByPartner}
	}
	out := []Rule{}
	for _, d := range posting.Kinds() {
		k := d.Kind
		r, ok := stored[string(k)]
		if !ok {
			r = Rule{Kind: string(k), ByPartner: d.ByPartner}
		}
		out = append(out, r)
		delete(stored, string(k))
	}
	for _, r := range rows {
		if x, ok := stored[r.Kind]; ok {
			out = append(out, x)
		}
	}
	return out, suggestRules(ctx, out)
}

// suggestRules fills the applied template's account and side into rules without one, where
// the template's account still exists and matches the kind's partner flag.
func suggestRules(ctx context.Context, rules []Rule) error {
	code, err := store.GetChartTemplate.Run(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	t, _ := templateOf(code)
	byKind := map[string]int{}
	var codes []string
	for i, r := range t.Rules {
		byKind[r.Kind] = i
		codes = append(codes, r.Account)
	}
	accounts, err := store.AccountIDsByCode.Run(ctx, codes)
	if err != nil {
		return err
	}
	for i, r := range rules {
		j, ok := byKind[r.Kind]
		if r.AccountID != nil || !ok {
			continue
		}
		for _, a := range accounts {
			if a.Code == t.Rules[j].Account && a.ByPartner == r.ByPartner {
				rules[i].Suggested = &RuleSuggestion{AccountID: a.ID, Side: t.Rules[j].Side}
			}
		}
	}
	return nil
}

func (s *Service) SaveRule(ctx context.Context, kind string, accountID int64, side string) error {
	if side != "debit" && side != "credit" {
		return ErrRuleSideInvalid
	}
	return s.writeAccount(ctx, func(ctx context.Context) error {
		ok, err := postableActive(ctx, []int64{accountID})
		if err != nil {
			return err
		}
		if !ok[accountID] {
			return ErrAccountNotPostable
		}
		flags, err := accountFlags(ctx, []int64{accountID})
		if err != nil {
			return err
		}
		if d, _ := posting.Declared(posting.Kind(kind)); d.ByPartner != flags[accountID] {
			return ErrRulePartnerMismatch
		}
		if err := store.SaveRule.Run(ctx, store.SaveRuleParams{Kind: kind, AccountID: accountID, Side: side}); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "accounting.rule_saved", map[string]any{"kind": kind, "account_id": accountID, "side": side})
	})
}

func (s *Service) DeleteRule(ctx context.Context, kind string) error {
	return s.writeAccount(ctx, func(ctx context.Context) error {
		if err := store.DeleteRule.Run(ctx, kind); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "accounting.rule_deleted", map[string]any{"kind": kind})
	})
}
