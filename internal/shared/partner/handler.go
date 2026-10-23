package partner

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type partnersOutput struct{ Body List }

type partnerOutput struct{ Body Partner }

type partnerInput struct{ Body NewPartner }

type updatePartnerInput struct {
	ID   int64 `path:"id"`
	Body Fields
}

func registerHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-partners", Method: http.MethodGet, Path: "/partners"},
		func(ctx context.Context, in *Filter) (*partnersOutput, error) {
			l, err := s.List(ctx, *in)
			return &partnersOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-partner", Method: http.MethodGet, Path: "/partners/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*partnerOutput, error) {
			p, err := s.Get(ctx, in.ID)
			return &partnerOutput{Body: p}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-partner", Method: http.MethodPost, Path: "/partners", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *partnerInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.Create(ctx, in.Body.Code, in.Body.Fields))
		})
	huma.Register(api, huma.Operation{OperationID: "update-partner", Method: http.MethodPut, Path: "/partners/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updatePartnerInput) (*struct{}, error) {
			return nil, s.Update(ctx, in.ID, in.Body)
		})
}
