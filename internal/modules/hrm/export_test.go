package hrm

import "context"

func (s *Service) ContractAt(ctx context.Context, employeeID int64, date string) (ContractTerms, bool, error) {
	return s.contractAt(ctx, employeeID, date)
}

var DayKinds = dayKinds
