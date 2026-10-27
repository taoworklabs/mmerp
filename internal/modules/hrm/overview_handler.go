package hrm

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type monthCloseOutput struct{ Body MonthClose }

func registerOverviewHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "get-month-close", Method: http.MethodGet, Path: "/hrm/overview/month-close",
		Description: "Org units without a posted timesheet and legal entities without a posted payroll for last month, within the actor's scope."},
		func(ctx context.Context, _ *struct{}) (*monthCloseOutput, error) {
			m, err := s.MonthClose(ctx)
			return &monthCloseOutput{Body: m}, err
		})
}
