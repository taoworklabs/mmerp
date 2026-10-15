package numbering

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type seriesOutput struct{ Body []CodeSeries }

type setPrefixInput struct {
	Code string `path:"code"`
	Body struct {
		Prefix string `json:"prefix" doc:"0 to 10 characters, no whitespace"`
	}
}

func registerHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-code-series", Method: http.MethodGet, Path: "/code-series"},
		func(ctx context.Context, _ *struct{}) (*seriesOutput, error) {
			l, err := s.Series(ctx)
			return &seriesOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "set-code-series-prefix", Method: http.MethodPut, Path: "/code-series/{code}/prefix", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *setPrefixInput) (*struct{}, error) {
			return nil, s.SetPrefix(ctx, in.Code, in.Body.Prefix)
		})
}
