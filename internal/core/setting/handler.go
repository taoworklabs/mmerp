package setting

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type entityInput struct {
	ID int64 `path:"id"`
}

type settingsOutput struct{ Body []LegalEntitySetting }

type setInput struct {
	ID   int64  `path:"id"`
	Key  string `path:"key"`
	Body struct {
		Value string `json:"value"`
	}
}

func registerHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-legal-entity-settings", Method: http.MethodGet, Path: "/legal-entities/{id}/settings"},
		func(ctx context.Context, in *entityInput) (*settingsOutput, error) {
			l, err := s.LegalEntitySettings(ctx, in.ID)
			return &settingsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "set-legal-entity-setting", Method: http.MethodPut, Path: "/legal-entities/{id}/settings/{key}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *setInput) (*struct{}, error) {
			return nil, s.SetFor(ctx, in.ID, in.Key, in.Body.Value)
		})
}
