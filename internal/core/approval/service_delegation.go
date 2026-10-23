package approval

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/core/approval/internal/store"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// DelegateUsers lists the accounts a delegate, or whom a delegation is for, can be picked from; any
// user may read it. Logins are sign-in identifiers, so only delegation managers see them.
func (s *Service) DelegateUsers(ctx context.Context) ([]ApproverUser, error) {
	all, err := s.CanManageAllDelegations(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := store.ListUsers.Run(ctx)
	out := make([]ApproverUser, len(rows))
	for i, r := range rows {
		out[i] = ApproverUser(r)
		if !all {
			out[i].Login = ""
		}
	}
	return out, err
}

// CanManageAllDelegations says whether the actor manages everyone's delegations, not only their own.
func (s *Service) CanManageAllDelegations(ctx context.Context) (bool, error) {
	err := s.d.IAM.RequireCore(ctx, iam.PermManageApproval)
	if errors.Is(err, platform.ErrForbidden) {
		return false, nil
	}
	return err == nil, err
}

// Delegations lists the delegations not yet ended that the actor manages, by start date.
func (s *Service) Delegations(ctx context.Context) ([]Delegation, error) {
	all, err := s.CanManageAllDelegations(ctx)
	if err != nil {
		return nil, err
	}
	today, err := s.d.Setting.Today(ctx)
	if err != nil {
		return nil, err
	}
	actor, _ := platform.ActorFrom(ctx)
	rows, err := store.ListDelegations.Run(ctx, store.ListDelegationsParams{Today: today, Everyone: all, Actor: actor})
	out := make([]Delegation, len(rows))
	for i, r := range rows {
		out[i] = Delegation{ID: r.ID, UserID: r.UserID, UserName: r.UserName, DelegateID: r.DelegateID, DelegateName: r.DelegateName,
			StartsOn: *platform.DatePtr(r.StartsOn), EndsOn: *platform.DatePtr(r.EndsOn),
			Active:         !r.StartsOn.Time.After(today.Time),
			AllowedActions: []string{"delete"}, // every row listed is one the actor manages
		}
	}
	return out, err
}

// CreateDelegation records that in's delegate votes for its user over its dates.
func (s *Service) CreateDelegation(ctx context.Context, in DelegationInput) (int64, error) {
	actor, _ := platform.ActorFrom(ctx)
	user := actor
	if in.UserID != nil && *in.UserID != actor {
		if err := s.d.IAM.RequireCore(ctx, iam.PermManageApproval); err != nil {
			return 0, err
		}
		// A manager naming themself could vote anyone's open seats at once.
		if in.DelegateID == actor {
			return 0, ErrInvalidDelegation("creator")
		}
		user = *in.UserID
	}
	if in.DelegateID == user {
		return 0, ErrInvalidDelegation("self")
	}
	var id int64
	err := platform.InTx(ctx, func(ctx context.Context) error {
		ok, err := store.UsersExist.Run(ctx, store.UsersExistParams{UserID: user, DelegateID: in.DelegateID})
		if err != nil {
			return err
		}
		if !ok {
			return platform.ErrNotFound
		}
		starts, ends := platform.NullDate(&in.StartsOn), platform.NullDate(&in.EndsOn)
		if !starts.Valid || !ends.Valid || ends.Time.Before(starts.Time) {
			return ErrInvalidDelegation("dates")
		}
		today, err := s.d.Setting.Today(ctx)
		if err != nil {
			return err
		}
		if ends.Time.Before(today.Time) {
			return ErrInvalidDelegation("past")
		}
		if err := store.LockDelegations.Run(ctx, user); err != nil {
			return err
		}
		other, err := store.OverlappingDelegation.Run(ctx, store.OverlappingDelegationParams{UserID: user, StartsOn: starts, EndsOn: ends})
		if err == nil {
			e := ErrInvalidDelegation("overlap")
			e.Params["id"] = other
			return e
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if id, err = store.InsertDelegation.Run(ctx, store.InsertDelegationParams{UserID: user, DelegateID: in.DelegateID,
			StartsOn: starts, EndsOn: ends, CreatedBy: actor}); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "approval.delegation_created", map[string]any{"id": id, "user": user,
			"delegate": in.DelegateID, "starts_on": in.StartsOn, "ends_on": in.EndsOn})
	})
	return id, err
}

// DeleteDelegation removes a delegation; votes already cast through it stay as cast.
// Anyone who doesn't manage it gets not_found, so ids of others' rows don't leak.
func (s *Service) DeleteDelegation(ctx context.Context, id int64) error {
	all, err := s.CanManageAllDelegations(ctx)
	if err != nil {
		return err
	}
	actor, _ := platform.ActorFrom(ctx)
	return platform.InTx(ctx, func(ctx context.Context) error {
		d, err := store.GetDelegation.Run(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) || err == nil && !all && d.UserID != actor {
			return platform.ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := store.DeleteDelegation.Run(ctx, id); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "approval.delegation_deleted", map[string]any{"id": id, "user": d.UserID,
			"delegate": d.DelegateID, "starts_on": d.StartsOn.Time.Format(time.DateOnly), "ends_on": d.EndsOn.Time.Format(time.DateOnly)})
	})
}
