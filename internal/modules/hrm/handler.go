package hrm

import (
	"github.com/danielgtaylor/huma/v2"
)

type selfServiceActionsOutput struct{ Body SelfServiceActions }

func registerHandlers(api huma.API, s *Service) {
	registerLeaveHandlers(api, s)
	registerPayrollHandlers(api, s)
	registerSetupHandlers(api, s)
	registerOvertimeHandlers(api, s)
	registerContractHandlers(api, s)
	registerTimesheetHandlers(api, s)
	registerEmployeeHandlers(api, s)
}
