package discussion

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/core/record"
)

type refInput struct {
	Type string `path:"type" doc:"Record type, e.g. hrm.leave_request"`
	ID   int64  `path:"id"`
}

type listOutput struct{ Body Discussion }

type mentionableOutput struct {
	Body []Mentionable `nullable:"false"`
}

type addInput struct {
	Type string `path:"type"`
	ID   int64  `path:"id"`
	Body struct {
		Body string `json:"body"`
	}
}

func registerHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-comments", Method: http.MethodGet, Path: "/records/{type}/{id}/comments",
		Description: "Not found unless the caller may view the record."},
		func(ctx context.Context, in *refInput) (*listOutput, error) {
			d, err := s.List(ctx, record.Ref{Type: in.Type, ID: in.ID})
			return &listOutput{Body: d}, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-mentionable", Method: http.MethodGet, Path: "/records/{type}/{id}/mentionable",
		Description: "The users the caller may mention in a comment: the others who may view the record. Not found unless the caller may view it."},
		func(ctx context.Context, in *refInput) (*mentionableOutput, error) {
			users, err := s.Mentionable(ctx, record.Ref{Type: in.Type, ID: in.ID})
			return &mentionableOutput{Body: users}, err
		})
	huma.Register(api, huma.Operation{OperationID: "add-comment", Method: http.MethodPost, Path: "/records/{type}/{id}/comments", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *addInput) (*struct{}, error) {
			return nil, s.Add(ctx, record.Ref{Type: in.Type, ID: in.ID}, in.Body.Body)
		})
}
