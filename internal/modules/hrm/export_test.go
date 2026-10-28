package hrm

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/core/printing"
)

func (s *Service) ContractAt(ctx context.Context, employeeID int64, date string) (ContractTerms, bool, error) {
	return s.contractAt(ctx, employeeID, date)
}

var DayKinds = dayKinds

func (s *Service) Payslips(ctx context.Context, id int64) ([]printing.Part, error) {
	return s.payslips(ctx, id)
}

var (
	UnitPaths   = unitPaths
	ResolveUnit = resolveUnit
)
