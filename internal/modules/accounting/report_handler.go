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
	LegalEntityID int64  `query:"legal_entity_id" required:"true"`
	AccountID     int64  `query:"account_id" required:"true"`
	From          string `query:"from" required:"true" format:"date"`
	To            string `query:"to" required:"true" format:"date"`
}

type partnerBalancesInput struct {
	LegalEntityID int64  `query:"legal_entity_id" required:"true"`
	AccountID     int64  `query:"account_id" doc:"A tracked account; all of them when left out"`
	From          string `query:"from" required:"true" format:"date"`
	To            string `query:"to" required:"true" format:"date"`
}

type partnerLedgerInput struct {
	LegalEntityID int64  `query:"legal_entity_id" required:"true"`
	PartnerID     int64  `query:"partner_id" required:"true"`
	AccountID     int64  `query:"account_id" doc:"A tracked account; all of them when left out"`
	From          string `query:"from" required:"true" format:"date"`
	To            string `query:"to" required:"true" format:"date"`
}

type partnerBalancesOutput struct{ Body PartnerBalances }

// optionalID: a query id left out arrives as 0.
func optionalID(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
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
	huma.Register(api, huma.Operation{OperationID: "get-accounting-partner-balances", Method: http.MethodGet, Path: "/accounting/partner-balances"},
		func(ctx context.Context, in *partnerBalancesInput) (*partnerBalancesOutput, error) {
			b, err := s.PartnerBalances(ctx, in.LegalEntityID, optionalID(in.AccountID), in.From, in.To)
			return &partnerBalancesOutput{Body: b}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-accounting-partner-ledger", Method: http.MethodGet, Path: "/accounting/partner-ledger"},
		func(ctx context.Context, in *partnerLedgerInput) (*ledgerOutput, error) {
			l, err := s.PartnerLedger(ctx, in.LegalEntityID, in.PartnerID, optionalID(in.AccountID), in.From, in.To)
			return &ledgerOutput{Body: l}, err
		})
}
