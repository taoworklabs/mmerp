package hrm

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type pitAuthorisationsInput struct {
	LegalEntity int64 `path:"legal_entity"`
	Year        int   `path:"year" minimum:"2000" maximum:"2100"`
}

type pitAuthorisationsOutput struct{ Body PITAuthorisations }

type savePITAuthorisationsInput struct {
	LegalEntity int64 `path:"legal_entity"`
	Year        int   `path:"year" minimum:"2000" maximum:"2100"`
	Body        struct {
		EmployeeIDs []int64 `json:"employee_ids" nullable:"false" maxItems:"10000"`
	}
}

func registerPITAuthorisationHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "get-pit-authorisations", Method: http.MethodGet, Path: "/hrm/pit-authorisations/{legal_entity}/{year}",
		Description: "Everyone on a posted payroll of the legal entity in the year, and who authorised it to finalise their PIT; allowed_actions has edit."},
		func(ctx context.Context, in *pitAuthorisationsInput) (*pitAuthorisationsOutput, error) {
			a, err := s.PITAuthorisations(ctx, in.LegalEntity, in.Year)
			return &pitAuthorisationsOutput{Body: a}, err
		})
	huma.Register(api, huma.Operation{OperationID: "save-pit-authorisations", Method: http.MethodPut, Path: "/hrm/pit-authorisations/{legal_entity}/{year}", DefaultStatus: http.StatusNoContent,
		Description: "Replaces who authorised the legal entity to finalise their PIT for the year."},
		func(ctx context.Context, in *savePITAuthorisationsInput) (*struct{}, error) {
			return nil, s.SavePITAuthorisations(ctx, in.LegalEntity, in.Year, in.Body.EmployeeIDs)
		})
}
