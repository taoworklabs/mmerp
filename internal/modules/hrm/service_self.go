package hrm

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// Self-service documents (leave and overtime requests) are filed by the employee
// themself, viewed by their managers, and handled by HR through view and edit permissions.

// selfService is who a self-service document is about.
type selfService struct {
	employee int64
	user     pgtype.Int8 // the employee's account, if any
	unit     int64       // the document's org unit
}

// canSelfService: one's own documents need no role; managers up the chain may view;
// others need view or edit at the document's org unit. Only HR cancels.
func (s *Service) canSelfService(ctx context.Context, d selfService, action record.Action, viewPerm, editPerm string) (bool, error) {
	actor, _ := platform.ActorFrom(ctx)
	self := d.user.Valid && d.user.Int64 == actor
	switch action {
	case record.View, record.Export:
		if self {
			return true, nil
		}
		if ok, err := s.allowed(ctx, viewPerm, d.unit); err != nil || ok {
			return ok, err
		}
		return store.New(platform.DBFrom(ctx)).IsManager(ctx, store.IsManagerParams{Employee: d.employee, Actor: actor})
	case record.Edit, record.Post:
		if self {
			return true, nil
		}
		return s.allowed(ctx, editPerm, d.unit)
	case record.Cancel:
		return s.allowed(ctx, editPerm, d.unit)
	}
	return false, nil
}

// selfGrant is the relation grant of a self-service document: one's own need no role,
// except to cancel; managers up the chain may see them.
func selfGrant(ctx context.Context, employee int64, user pgtype.Int8, action record.Action) (bool, error) {
	actor, _ := platform.ActorFrom(ctx)
	if user.Valid && user.Int64 == actor {
		return action != record.Cancel, nil
	}
	switch action {
	case record.View, record.Export, record.ViewFiles:
		return store.New(platform.DBFrom(ctx)).IsManager(ctx, store.IsManagerParams{Employee: employee, Actor: actor})
	}
	return false, nil
}

// managerApprover returns the account of the manager level steps up the employee's chain.
func managerApprover(ctx context.Context, employee int64, level int) ([]int64, error) {
	u, err := store.New(platform.DBFrom(ctx)).ManagerUser(ctx, store.ManagerUserParams{Employee: employee, Level: int32(level)})
	if errors.Is(err, pgx.ErrNoRows) || err == nil && !u.Valid {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return []int64{u.Int64}, nil
}

// subjectOf is the employee's account, who never approves their own document.
func subjectOf(user pgtype.Int8) []int64 {
	if !user.Valid {
		return nil
	}
	return []int64{user.Int64}
}

// selfServiceActions lists what the actor may do before picking a self-service document.
func (s *Service) selfServiceActions(ctx context.Context, editPerm string) (SelfServiceActions, error) {
	out, err := selfEmployee(ctx)
	if err != nil {
		return out, err
	}
	edit, err := s.d.IAM.Scope(ctx, "hrm", editPerm)
	if err != nil {
		return out, err
	}
	if platform.ProductGate(ctx, "hrm", platform.ClassWrite) != nil {
		return out, nil
	}
	if out.SelfEmployeeID != nil || edit.Any() {
		out.AllowedActions = append(out.AllowedActions, "create")
	}
	if edit.Any() {
		out.AllowedActions = append(out.AllowedActions, "create_for_others")
	}
	return out, nil
}

// selfEmployee names the actor's own employee record, if any, with no actions yet.
func selfEmployee(ctx context.Context) (SelfServiceActions, error) {
	out := SelfServiceActions{AllowedActions: []string{}}
	actor, _ := platform.ActorFrom(ctx)
	self, err := store.New(platform.DBFrom(ctx)).EmployeeByUser(ctx, pgtype.Int8{Int64: actor, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err == nil {
		out.SelfEmployeeID, out.SelfEmployeeName = &self.ID, &self.FullName
	}
	return out, err
}

// filer resolves who a new self-service document is for: the actor's own employee
// record when id is nil, otherwise an employee the actor may file for with editPerm.
func (s *Service) filer(ctx context.Context, id *int64, editPerm string) (employee, unit int64, err error) {
	employee, unit, user, err := subject(ctx, id)
	if err != nil {
		return 0, 0, err
	}
	if actor, _ := platform.ActorFrom(ctx); !user.Valid || user.Int64 != actor {
		if ok, err := s.allowed(ctx, editPerm, unit); err != nil || !ok {
			return 0, 0, platform.OrErr(err, platform.ErrNotFound)
		}
	}
	return employee, unit, nil
}

// subject resolves the employee a new self-service document is about, with their
// account: the actor's own record when id is nil. Whether the actor may file for
// them is not checked here.
func subject(ctx context.Context, id *int64) (employee, unit int64, user pgtype.Int8, err error) {
	q := store.New(platform.DBFrom(ctx))
	if id == nil {
		actor, _ := platform.ActorFrom(ctx)
		self, err := q.EmployeeByUser(ctx, pgtype.Int8{Int64: actor, Valid: true})
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, user, ErrNotAnEmployee
		}
		return self.ID, self.OrgUnitID, pgtype.Int8{Int64: actor, Valid: true}, err
	}
	e, err := q.GetEmployee(ctx, *id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, user, platform.ErrNotFound
	}
	return e.ID, e.OrgUnitID, e.UserID, err
}
