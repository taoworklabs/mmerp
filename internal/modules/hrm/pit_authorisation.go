package hrm

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// PITAuthorisations lists everyone on a posted payroll of the legal entity in the year, and
// who of them authorised it to finalise their PIT.
func (s *Service) PITAuthorisations(ctx context.Context, legalEntity int64, year int) (PITAuthorisations, error) {
	out := PITAuthorisations{People: []PITAuthorisationPerson{}, AllowedActions: []string{}}
	if err := s.checkLegalEntity(ctx, legalEntity); err != nil {
		return out, err
	}
	if err := s.require(ctx, PermSalaryView, legalEntity); err != nil {
		return out, err
	}
	rows, err := store.PITAuthorisationCandidates.Run(ctx, store.PITAuthorisationCandidatesParams{LegalEntityID: legalEntity, Year: int32(year)})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.People = append(out.People, PITAuthorisationPerson{EmployeeID: r.ID, Code: r.Code, FullName: r.FullName, December: r.December,
			Authorised: r.AuthorisedTo.Valid && r.AuthorisedTo.Int64 == legalEntity, Elsewhere: r.AuthorisedTo.Valid && r.AuthorisedTo.Int64 != legalEntity})
	}
	if ok, err := s.allowed(ctx, PermPayrollEdit, legalEntity); err != nil {
		return out, err
	} else if ok && platform.ProductGate(ctx, "hrm", platform.ClassWrite) == nil {
		out.AllowedActions = append(out.AllowedActions, "edit")
	}
	return out, nil
}

// SavePITAuthorisations replaces who authorised the legal entity for the year. Each must be on
// one of its posted payrolls of the year and not have authorised another legal entity.
func (s *Service) SavePITAuthorisations(ctx context.Context, legalEntity int64, year int, employeeIDs []int64) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.checkLegalEntity(ctx, legalEntity); err != nil {
			return err
		}
		if err := s.require(ctx, PermPayrollEdit, legalEntity); err != nil {
			return err
		}
		rows, err := store.PITAuthorisationCandidates.Run(ctx, store.PITAuthorisationCandidatesParams{LegalEntityID: legalEntity, Year: int32(year)})
		if err != nil {
			return err
		}
		ids := slices.Compact(slices.Sorted(slices.Values(employeeIDs)))
		var elsewhere []string
		for _, id := range ids {
			i := slices.IndexFunc(rows, func(r store.PITAuthorisationCandidatesRow) bool { return r.ID == id })
			if i < 0 {
				return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_request", Params: map[string]any{"fields": []string{"employee_ids"}}}
			}
			if r := rows[i]; r.AuthorisedTo.Valid && r.AuthorisedTo.Int64 != legalEntity {
				elsewhere = append(elsewhere, r.Code)
			}
		}
		if elsewhere != nil {
			return &platform.Error{Status: ErrPITAuthorisedElsewhere.Status, Code: ErrPITAuthorisedElsewhere.Code,
				Params: map[string]any{"employees": strings.Join(elsewhere, ", ")}}
		}
		key := store.PITAuthorisedParams{LegalEntityID: legalEntity, Year: int32(year)}
		old, err := store.PITAuthorised.Run(ctx, key)
		if err != nil {
			return err
		}
		if err := store.DeletePITAuthorisations.Run(ctx, store.DeletePITAuthorisationsParams(key)); err != nil {
			return err
		}
		if err := store.InsertPITAuthorisations.Run(ctx, store.InsertPITAuthorisationsParams{EmployeeIds: ids, Year: int32(year), LegalEntityID: legalEntity}); err != nil {
			return err
		}
		added := slices.DeleteFunc(slices.Clone(ids), func(id int64) bool { return slices.Contains(old, id) })
		removed := slices.DeleteFunc(old, func(id int64) bool { return slices.Contains(ids, id) })
		return s.d.Audit.Record(ctx, "hrm.pit_authorisations_changed", map[string]any{
			"legal_entity_id": legalEntity, "year": year, "added": added, "removed": removed})
	})
}

// PITAuthorisations is a legal entity's year: allowed_actions has edit.
type PITAuthorisations struct {
	People         []PITAuthorisationPerson `json:"people" nullable:"false"`
	AllowedActions []string                 `json:"allowed_actions" nullable:"false"`
}

type PITAuthorisationPerson struct {
	EmployeeID int64  `json:"employee_id"`
	Code       string `json:"code"`
	FullName   string `json:"full_name"`
	December   bool   `json:"december" doc:"On the posted payroll of December"`
	Authorised bool   `json:"authorised"`
	Elsewhere  bool   `json:"elsewhere" doc:"Authorised another legal entity for the year"`
}
