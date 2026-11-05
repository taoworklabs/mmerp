package accounting

import (
	"context"

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
		stored[r.Kind] = Rule{Kind: r.Kind, AccountID: &r.AccountID, Side: &r.Side}
	}
	out := []Rule{}
	for _, k := range posting.Kinds() {
		r, ok := stored[string(k)]
		if !ok {
			r = Rule{Kind: string(k)}
		}
		out = append(out, r)
		delete(stored, string(k))
	}
	for _, r := range rows {
		if x, ok := stored[r.Kind]; ok {
			out = append(out, x)
		}
	}
	return out, nil
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
