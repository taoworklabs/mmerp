package accounting

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type rulesOutput struct{ Body RuleList }

type ruleInput struct {
	Kind string `path:"kind"`
	Body RuleFields
}

type ruleKindInput struct {
	Kind string `path:"kind"`
}

type startsOutput struct{ Body StartList }

type startInput struct {
	LegalEntityID int64 `path:"legal_entity_id"`
	Body          struct {
		StartDate *string `json:"start_date" format:"date" nullable:"true"`
	}
}

func registerRuleHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-accounting-rules", Method: http.MethodGet, Path: "/accounting/rules"},
		func(ctx context.Context, _ *struct{}) (*rulesOutput, error) {
			items, err := s.Rules(ctx)
			return &rulesOutput{Body: RuleList{Items: items}}, err
		})
	huma.Register(api, huma.Operation{OperationID: "save-accounting-rule", Method: http.MethodPut, Path: "/accounting/rules/{kind}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *ruleInput) (*struct{}, error) {
			return nil, s.SaveRule(ctx, in.Kind, in.Body.AccountID, in.Body.Side)
		})
	huma.Register(api, huma.Operation{OperationID: "delete-accounting-rule", Method: http.MethodDelete, Path: "/accounting/rules/{kind}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *ruleKindInput) (*struct{}, error) {
			return nil, s.DeleteRule(ctx, in.Kind)
		})
	huma.Register(api, huma.Operation{OperationID: "list-accounting-starts", Method: http.MethodGet, Path: "/accounting/starts"},
		func(ctx context.Context, _ *struct{}) (*startsOutput, error) {
			items, err := s.Starts(ctx)
			return &startsOutput{Body: StartList{Items: items}}, err
		})
	huma.Register(api, huma.Operation{OperationID: "set-accounting-start", Method: http.MethodPut, Path: "/accounting/starts/{legal_entity_id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *startInput) (*struct{}, error) {
			return nil, s.SetStart(ctx, in.LegalEntityID, in.Body.StartDate)
		})
}
