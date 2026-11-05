package accounting

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/modules/accounting/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
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
			AccountFields: AccountFields{Code: r.Code, Name: r.Name, Type: r.Type, ParentID: platform.Int8Ptr(r.ParentID), Active: r.Active}})
	}
	return out, nil
}

func (s *Service) CreateAccount(ctx context.Context, f AccountFields) (id int64, err error) {
	err = s.writeAccount(ctx, func(ctx context.Context) error {
		f.Name = strings.TrimSpace(f.Name)
		if err := checkParent(ctx, 0, f); err != nil {
			return err
		}
		id, err = store.CreateAccount.Run(ctx, store.CreateAccountParams{Code: f.Code, Name: f.Name, Type: f.Type, ParentID: platform.NullInt8(f.ParentID), Active: f.Active})
		if err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "accounting.account_saved", map[string]any{"id": id, "code": f.Code, "name": f.Name, "type": f.Type, "active": f.Active})
	})
	return id, err
}

func (s *Service) UpdateAccount(ctx context.Context, id int64, f AccountFields) error {
	return s.writeAccount(ctx, func(ctx context.Context) error {
		f.Name = strings.TrimSpace(f.Name)
		cur, err := store.LockAccount.Run(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		if cur.HasLines && (cur.Code != f.Code || cur.Type != f.Type) || cur.HasChildren && cur.Type != f.Type {
			return ErrAccountInUse
		}
		if err := checkParent(ctx, id, f); err != nil {
			return err
		}
		if err := store.UpdateAccount.Run(ctx, store.UpdateAccountParams{ID: id, Code: f.Code, Name: f.Name, Type: f.Type, ParentID: platform.NullInt8(f.ParentID), Active: f.Active}); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "accounting.account_saved", map[string]any{"id": id, "code": f.Code, "name": f.Name, "type": f.Type, "active": f.Active})
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
	return platform.InTx(ctx, fn)
}

// checkParent: id is 0 on create. A parent exists, is not the account or below it, has its
// type and carries no entries.
func checkParent(ctx context.Context, id int64, f AccountFields) error {
	if f.ParentID == nil {
		return nil
	}
	p, err := store.AccountParent.Run(ctx, store.AccountParentParams{ID: id, ParentID: *f.ParentID})
	if errors.Is(err, pgx.ErrNoRows) || err == nil && (p.Cyclic || p.Type != f.Type || p.HasLines) {
		return ErrAccountParentInvalid
	}
	return err
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
