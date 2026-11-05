package accounting

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type reportRangeInput struct {
	LegalEntityID int64  `query:"legal_entity_id" required:"true"`
	From          string `query:"from" required:"true" format:"date"`
	To            string `query:"to" required:"true" format:"date"`
}

type ledgerInput struct {
	reportRangeInput
	AccountID int64 `query:"account_id" required:"true"`
}

type ledgerOutput struct{ Body Ledger }

type trialBalanceOutput struct{ Body TrialBalance }

func registerReportHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "get-accounting-ledger", Method: http.MethodGet, Path: "/accounting/ledger"},
		func(ctx context.Context, in *ledgerInput) (*ledgerOutput, error) {
			l, err := s.Ledger(ctx, in.LegalEntityID, in.AccountID, in.From, in.To)
			return &ledgerOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-accounting-trial-balance", Method: http.MethodGet, Path: "/accounting/trial-balance"},
		func(ctx context.Context, in *reportRangeInput) (*trialBalanceOutput, error) {
			tb, err := s.TrialBalance(ctx, in.LegalEntityID, in.From, in.To)
			return &trialBalanceOutput{Body: tb}, err
		})
}
