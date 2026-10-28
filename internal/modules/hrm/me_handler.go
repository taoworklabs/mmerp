package hrm

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type myProfileOutput struct{ Body MyProfile }

type myDaysInput struct {
	Month string `query:"month" required:"true" pattern:"^[0-9]{4}-(0[1-9]|1[0-2])$" doc:"YYYY-MM"`
}

type myDaysOutput struct{ Body MyDays }

type myPayslipsOutput struct{ Body []MyPayslipSummary }

type myPayslipInput struct {
	PayrollID int64 `path:"payroll_id"`
}

type myPayslipOutput struct{ Body MyPayslip }

func registerMeHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "get-my-profile", Method: http.MethodGet, Path: "/hrm/me",
		Description: "The actor's own employee record, without sensitive fields."},
		func(ctx context.Context, _ *struct{}) (*myProfileOutput, error) {
			p, err := s.MyProfile(ctx)
			return &myProfileOutput{Body: p}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-my-days", Method: http.MethodGet, Path: "/hrm/me/days",
		Description: "The actor's month day by day: worked days, posted leave and overtime, the calendar."},
		func(ctx context.Context, in *myDaysInput) (*myDaysOutput, error) {
			d, err := s.MyDays(ctx, in.Month)
			return &myDaysOutput{Body: d}, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-my-payslips", Method: http.MethodGet, Path: "/hrm/me/payslips",
		Description: "The actor's payslips of posted payrolls, newest first; every read is audited."},
		func(ctx context.Context, _ *struct{}) (*myPayslipsOutput, error) {
			l, err := s.MyPayslips(ctx)
			return &myPayslipsOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-my-payslip", Method: http.MethodGet, Path: "/hrm/me/payslips/{payroll_id}",
		Description: "The actor's part of a posted payroll; every read is audited."},
		func(ctx context.Context, in *myPayslipInput) (*myPayslipOutput, error) {
			p, err := s.MyPayslip(ctx, in.PayrollID)
			return &myPayslipOutput{Body: p}, err
		})
}
