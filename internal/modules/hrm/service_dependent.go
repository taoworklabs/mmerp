package hrm

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// Dependents lists an employee's dependents; every read is audited.
func (s *Service) Dependents(ctx context.Context, employeeID int64) ([]Dependent, error) {
	out := []Dependent{}
	err := platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.dependentAccess(ctx, employeeID, PermView); err != nil {
			return err
		}
		if err := s.d.Audit.RecordFor(ctx, "hrm.dependents_viewed", audit.Ref{Type: employeeType, ID: employeeID}, nil); err != nil {
			return err
		}
		rows, err := store.New(platform.DBFrom(ctx)).ListDependents(ctx, employeeID)
		if err != nil {
			return err
		}
		for _, r := range rows {
			d, err := openDependent(ctx, r.Data)
			if err != nil {
				return err
			}
			out = append(out, Dependent{ID: r.ID, DependentInput: d})
		}
		return nil
	})
	return out, err
}

// SaveDependent creates a dependent (id 0) or replaces one.
func (s *Service) SaveDependent(ctx context.Context, employeeID, id int64, in DependentInput) (int64, error) {
	err := platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.dependentAccess(ctx, employeeID, PermEdit); err != nil {
			return err
		}
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		data := platform.Encrypt(ctx, "hrm.dependents.data", raw)
		q := store.New(platform.DBFrom(ctx))
		var old *DependentInput
		if id == 0 {
			if id, err = q.CreateDependent(ctx, store.CreateDependentParams{EmployeeID: employeeID, Data: data}); err != nil {
				return err
			}
		} else {
			prev, err := s.lockDependent(ctx, employeeID, id)
			if err != nil {
				return err
			}
			old = &prev
			if err := q.UpdateDependent(ctx, store.UpdateDependentParams{ID: id, EmployeeID: employeeID, Data: data}); err != nil {
				return err
			}
		}
		return s.d.Audit.RecordChanges(ctx, "hrm.dependent_saved", audit.Ref{Type: employeeType, ID: employeeID},
			[]audit.Change{{Field: "dependent", Old: old, New: Dependent{ID: id, DependentInput: in}, Sensitive: true}})
	})
	return id, err
}

// DeleteDependent removes a dependent.
func (s *Service) DeleteDependent(ctx context.Context, employeeID, id int64) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.dependentAccess(ctx, employeeID, PermEdit); err != nil {
			return err
		}
		old, err := s.lockDependent(ctx, employeeID, id)
		if err != nil {
			return err
		}
		if err := store.New(platform.DBFrom(ctx)).DeleteDependent(ctx, store.DeleteDependentParams{ID: id, EmployeeID: employeeID}); err != nil {
			return err
		}
		return s.d.Audit.RecordChanges(ctx, "hrm.dependent_deleted", audit.Ref{Type: employeeType, ID: employeeID},
			[]audit.Change{{Field: "dependent", Old: Dependent{ID: id, DependentInput: old}, New: nil, Sensitive: true}})
	})
}

func (s *Service) lockDependent(ctx context.Context, employeeID, id int64) (DependentInput, error) {
	data, err := store.New(platform.DBFrom(ctx)).GetDependent(ctx, store.GetDependentParams{ID: id, EmployeeID: employeeID})
	if errors.Is(err, pgx.ErrNoRows) {
		return DependentInput{}, platform.ErrNotFound
	}
	if err != nil {
		return DependentInput{}, err
	}
	return openDependent(ctx, data)
}

// Dependents are wholly sensitive: perm plus hrm.employee.sensitive at the employee's unit.
func (s *Service) dependentAccess(ctx context.Context, employeeID int64, perm string) error {
	e, err := s.visible(ctx, employeeID)
	if err != nil {
		return err
	}
	if err := s.require(ctx, perm, e.OrgUnitID); err != nil {
		return err
	}
	return s.require(ctx, PermSensitive, e.OrgUnitID)
}

func openDependent(ctx context.Context, data []byte) (DependentInput, error) {
	var d DependentInput
	raw, err := platform.Decrypt(ctx, "hrm.dependents.data", data)
	if err != nil {
		return d, err
	}
	return d, json.Unmarshal(raw, &d)
}
