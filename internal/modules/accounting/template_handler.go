package accounting

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type templatesOutput struct {
	Body struct {
		Items []TemplateInfo `json:"items" nullable:"false"`
	}
}

type applyTemplateInput struct {
	Code string `path:"code"`
}

func registerTemplateHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-chart-templates", Method: http.MethodGet, Path: "/accounting/templates"},
		func(ctx context.Context, _ *struct{}) (*templatesOutput, error) {
			out := &templatesOutput{}
			out.Body.Items = Templates()
			return out, s.requireView(ctx)
		})
	huma.Register(api, huma.Operation{OperationID: "apply-chart-template", Method: http.MethodPost, Path: "/accounting/templates/{code}/apply", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *applyTemplateInput) (*struct{}, error) {
			return nil, s.ApplyTemplate(ctx, in.Code)
		})
}
