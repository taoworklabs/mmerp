package accounting

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type balanceSheetInput struct {
	LegalEntityID int64  `query:"legal_entity_id" required:"true"`
	Date          string `query:"date" required:"true" format:"date"`
}

type statementOutput struct{ Body Statement }

type statementFormInput struct {
	Form string `path:"form" enum:"balance_sheet,income_statement"`
}

type statementFormOutput struct{ Body StatementForm }

type saveStatementFormInput struct {
	Form string `path:"form" enum:"balance_sheet,income_statement"`
	Body StatementForm
}

func registerStatementHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "get-balance-sheet", Method: http.MethodGet, Path: "/accounting/statements/balance-sheet"},
		func(ctx context.Context, in *balanceSheetInput) (*statementOutput, error) {
			st, err := s.BalanceSheet(ctx, in.LegalEntityID, in.Date)
			return &statementOutput{Body: st}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-income-statement", Method: http.MethodGet, Path: "/accounting/statements/income-statement"},
		func(ctx context.Context, in *reportRangeInput) (*statementOutput, error) {
			st, err := s.IncomeStatement(ctx, in.LegalEntityID, in.From, in.To)
			return &statementOutput{Body: st}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-statement-form", Method: http.MethodGet, Path: "/accounting/statement-forms/{form}"},
		func(ctx context.Context, in *statementFormInput) (*statementFormOutput, error) {
			f, err := s.StatementForm(ctx, in.Form)
			return &statementFormOutput{Body: f}, err
		})
	huma.Register(api, huma.Operation{OperationID: "save-statement-form", Method: http.MethodPut, Path: "/accounting/statement-forms/{form}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *saveStatementFormInput) (*struct{}, error) {
			return nil, s.SaveStatementForm(ctx, in.Form, in.Body)
		})
}
