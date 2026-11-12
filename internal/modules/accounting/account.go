package accounting

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/modules/accounting/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

// Accounts lists the whole chart by code to anyone who may view accounting at some org unit.
func (s *Service) Accounts(ctx context.Context) ([]Account, error) {
	sc, err := s.d.IAM.Scope(ctx, "accounting", PermView)
	if err != nil {
		return nil, err
	}
	if !sc.Any() {
		return nil, platform.ErrForbidden
	}
	rows, err := store.ListAccounts.Run(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(rows))
	for _, r := range rows {
		out = append(out, Account{ID: r.ID, Postable: r.Postable, HasLines: r.HasLines,
			AccountFields: AccountFields{Code: r.Code, Name: r.Name, Type: r.Type, ParentID: platform.Int8Ptr(r.ParentID), Active: r.Active, ByPartner: r.ByPartner}})
	}
	return out, nil
}

func (s *Service) CreateAccount(ctx context.Context, f AccountFields) (id int64, err error) {
	err = s.writeAccount(ctx, func(ctx context.Context) error {
		if f.Name = strings.TrimSpace(f.Name); f.Name == "" {
			return ErrAccountNameRequired
		}
		if err := lockAccounts(ctx, 0, f.ParentID); err != nil {
			return err
		}
		if err := checkParent(ctx, 0, f); err != nil {
			return err
		}
		id, err = store.CreateAccount.Run(ctx, store.CreateAccountParams{Code: f.Code, Name: f.Name, Type: f.Type, ParentID: platform.NullInt8(f.ParentID), Active: f.Active, ByPartner: f.ByPartner})
		if err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "accounting.account_saved", map[string]any{"id": id, "code": f.Code, "name": f.Name, "type": f.Type, "active": f.Active, "by_partner": f.ByPartner})
	})
	return id, err
}

func (s *Service) UpdateAccount(ctx context.Context, id int64, f AccountFields) error {
	return s.writeAccount(ctx, func(ctx context.Context) error {
		if f.Name = strings.TrimSpace(f.Name); f.Name == "" {
			return ErrAccountNameRequired
		}
		if err := lockAccounts(ctx, id, f.ParentID); err != nil {
			return err
		}
		cur, err := store.LockAccount.Run(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		// Moving an account with lines on it or below it would change the rollups of balances already reported.
		moved := cur.ParentID.Valid != (f.ParentID != nil) || f.ParentID != nil && cur.ParentID.Int64 != *f.ParentID
		if cur.HasLines && (cur.Code != f.Code || cur.Type != f.Type) || cur.TreeHasLines && moved || cur.HasChildren && cur.Type != f.Type {
			return ErrAccountInUse
		}
		if cur.ByPartner != f.ByPartner {
			if err := checkFlagChange(ctx, id, cur.HasLines, cur.HasChildren, f.ByPartner); err != nil {
				return err
			}
		}
		if err := checkParent(ctx, id, f); err != nil {
			return err
		}
		if err := store.UpdateAccount.Run(ctx, store.UpdateAccountParams{ID: id, Code: f.Code, Name: f.Name, Type: f.Type, ParentID: platform.NullInt8(f.ParentID), Active: f.Active, ByPartner: f.ByPartner}); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "accounting.account_saved", map[string]any{"id": id, "code": f.Code, "name": f.Name, "type": f.Type, "active": f.Active, "by_partner": f.ByPartner})
	})
}

func (s *Service) DeleteAccount(ctx context.Context, id int64) error {
	return s.writeAccount(ctx, func(ctx context.Context) error {
		cur, err := store.LockAccount.Run(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		if cur.HasLines || cur.HasChildren || cur.HasRule {
			return ErrAccountInUse
		}
		if err := store.DeleteAccount.Run(ctx, id); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "accounting.account_deleted", map[string]any{"id": id, "code": cur.Code})
	})
}

func (s *Service) writeAccount(ctx context.Context, fn func(context.Context) error) error {
	if err := s.d.IAM.RequireTenantWide(ctx, "accounting", PermManage); err != nil {
		return err
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if err := store.LockChart.Run(ctx); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// lockAccounts locks the edited account (0 on create) and its new parent, so no entry is
// written on either while their lines or children are checked.
func lockAccounts(ctx context.Context, id int64, parent *int64) error {
	ids := []int64{id}
	if parent != nil {
		ids = append(ids, *parent)
	}
	return store.LockAccounts.Run(ctx, ids)
}

// checkParent: id is 0 on create. A parent exists, is not the account or below it, has its
// type and carries no entries.
func checkParent(ctx context.Context, id int64, f AccountFields) error {
	if f.ParentID == nil {
		return nil
	}
	p, err := store.AccountParent.Run(ctx, store.AccountParentParams{ID: id, ParentID: *f.ParentID})
	// A tracked account must stay postable, so it takes no child.
	if errors.Is(err, pgx.ErrNoRows) || err == nil && (p.Cyclic || p.Type != f.Type || p.HasLines || p.ByPartner) {
		return ErrAccountParentInvalid
	}
	return err
}

// checkFlagChange: the flag moves only on a leaf without lines, and never so that a rule
// pointing at the account stops matching its kind.
func checkFlagChange(ctx context.Context, id int64, hasLines, hasChildren, byPartner bool) error {
	if hasLines {
		return ErrAccountInUse
	}
	if byPartner && hasChildren {
		return ErrAccountNotPostable
	}
	kinds, err := store.RuleKindsOfAccount.Run(ctx, id)
	if err != nil {
		return err
	}
	for _, k := range kinds {
		if d, _ := posting.Declared(posting.Kind(k)); d.ByPartner != byPartner {
			return ErrAccountRuleMismatch
		}
	}
	return nil
}

// accountFlags maps each existing account among ids to its by_partner flag.
func accountFlags(ctx context.Context, ids []int64) (map[int64]bool, error) {
	rows, err := store.AccountFlags.Run(ctx, ids)
	out := make(map[int64]bool, len(rows))
	for _, r := range rows {
		out[r.ID] = r.ByPartner
	}
	return out, err
}

// postableActive: true for accounts that exist, are active and have no child.
func postableActive(ctx context.Context, ids []int64) (map[int64]bool, error) {
	got, err := store.PostableActive.Run(ctx, ids)
	out := make(map[int64]bool, len(got))
	for _, id := range got {
		out[id] = true
	}
	return out, err
}
