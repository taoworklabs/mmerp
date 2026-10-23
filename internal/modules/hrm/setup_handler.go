package hrm

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type calendarInput struct {
	LegalEntity int64 `path:"legal_entity"`
	Year        int   `query:"year" minimum:"2000" maximum:"2100" required:"true"`
}

type calendarOutput struct{ Body WorkCalendar }

type workWeekInput struct {
	LegalEntity   int64  `path:"legal_entity"`
	EffectiveFrom string `path:"effective_from" format:"date"`
	Body          struct {
		OffDays []int `json:"off_days" nullable:"false" maxItems:"7"`
	}
}

type workWeekRefInput struct {
	LegalEntity   int64  `path:"legal_entity"`
	EffectiveFrom string `path:"effective_from" format:"date"`
}

type holidayInput struct {
	LegalEntity int64  `path:"legal_entity"`
	Date        string `path:"date" format:"date"`
	Body        struct {
		Name string `json:"name" minLength:"1" maxLength:"200"`
	}
}

type holidayRefInput struct {
	LegalEntity int64  `path:"legal_entity"`
	Date        string `path:"date" format:"date"`
}

type legalParamsOutput struct{ Body LegalParams }

type legalParamInput struct {
	Key           string `path:"key"`
	EffectiveFrom string `path:"effective_from" format:"date"`
	Body          struct {
		Value string `json:"value" minLength:"1" maxLength:"500"`
	}
}

func registerSetupHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "get-work-calendar", Method: http.MethodGet, Path: "/hrm/calendars/{legal_entity}",
		Description: "Weekly days off by version and the holidays of a year; allowed_actions has manage."},
		func(ctx context.Context, in *calendarInput) (*calendarOutput, error) {
			c, err := s.WorkCalendar(ctx, in.LegalEntity, in.Year)
			return &calendarOutput{Body: c}, err
		})
	huma.Register(api, huma.Operation{OperationID: "save-work-week", Method: http.MethodPut, Path: "/hrm/calendars/{legal_entity}/weeks/{effective_from}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *workWeekInput) (*struct{}, error) {
			return nil, s.SaveWorkWeek(ctx, in.LegalEntity, WorkWeek{EffectiveFrom: in.EffectiveFrom, OffDays: in.Body.OffDays})
		})
	huma.Register(api, huma.Operation{OperationID: "delete-work-week", Method: http.MethodDelete, Path: "/hrm/calendars/{legal_entity}/weeks/{effective_from}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *workWeekRefInput) (*struct{}, error) {
			return nil, s.DeleteWorkWeek(ctx, in.LegalEntity, in.EffectiveFrom)
		})
	huma.Register(api, huma.Operation{OperationID: "save-holiday", Method: http.MethodPut, Path: "/hrm/calendars/{legal_entity}/holidays/{date}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *holidayInput) (*struct{}, error) {
			return nil, s.SaveHoliday(ctx, in.LegalEntity, Holiday{Date: in.Date, Name: in.Body.Name})
		})
	huma.Register(api, huma.Operation{OperationID: "delete-holiday", Method: http.MethodDelete, Path: "/hrm/calendars/{legal_entity}/holidays/{date}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *holidayRefInput) (*struct{}, error) {
			return nil, s.DeleteHoliday(ctx, in.LegalEntity, in.Date)
		})
	huma.Register(api, huma.Operation{OperationID: "list-legal-params", Method: http.MethodGet, Path: "/hrm/legal-params",
		Description: "Every version of every legal parameter; allowed_actions has manage."},
		func(ctx context.Context, _ *struct{}) (*legalParamsOutput, error) {
			l, err := s.LegalParams(ctx)
			return &legalParamsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "save-legal-param", Method: http.MethodPut, Path: "/hrm/legal-params/{key}/{effective_from}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *legalParamInput) (*struct{}, error) {
			return nil, s.SaveLegalParam(ctx, LegalParam{Key: in.Key, EffectiveFrom: in.EffectiveFrom, Value: in.Body.Value})
		})
}
