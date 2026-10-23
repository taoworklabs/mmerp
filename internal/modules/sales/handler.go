package sales

import (
	"github.com/danielgtaylor/huma/v2"
)

type actionsOutput struct{ Body Actions }

func registerHandlers(api huma.API, s *Service) {
	registerCustomerHandlers(api, s)
	registerItemHandlers(api, s)
	registerDocHandlers(api, s)
}
