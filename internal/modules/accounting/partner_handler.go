package accounting

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type partnersInput struct {
	Q string `query:"q" maxLength:"100" doc:"Code, name or tax code"`
}

type partnersOutput struct{ Body AccountingPartnerList }

func registerPartnerHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-accounting-partners", Method: http.MethodGet, Path: "/accounting/partners"},
		func(ctx context.Context, in *partnersInput) (*partnersOutput, error) {
			l, err := s.Partners(ctx, in.Q)
			return &partnersOutput{Body: l}, err
		})
}
