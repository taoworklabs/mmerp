package printing

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type templatesOutput struct {
	Body []PrintTemplate `nullable:"false"`
}

type codeInput struct {
	Code string `path:"code" doc:"e.g. hrm.contract"`
}

type templateOutput struct{ Body PrintTemplateBlocks }

type blocksInput struct {
	Code string `path:"code"`
	Body struct {
		Blocks []BlockText `json:"blocks" nullable:"false" doc:"Every block of the template"`
	}
}

func registerHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-print-templates", Method: http.MethodGet, Path: "/print-templates"},
		func(ctx context.Context, _ *struct{}) (*templatesOutput, error) {
			l, err := s.Templates(ctx)
			return &templatesOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-print-template", Method: http.MethodGet, Path: "/print-templates/{code}"},
		func(ctx context.Context, in *codeInput) (*templateOutput, error) {
			t, err := s.TemplateBlocks(ctx, in.Code)
			return &templateOutput{Body: t}, err
		})
	huma.Register(api, huma.Operation{OperationID: "save-print-template-blocks", Method: http.MethodPut, Path: "/print-templates/{code}/blocks",
		DefaultStatus: http.StatusNoContent, Description: "Saves every block as the template's next version; documents already printed keep their texts."},
		func(ctx context.Context, in *blocksInput) (*struct{}, error) {
			return nil, s.SaveBlocks(ctx, in.Code, in.Body.Blocks)
		})
}
