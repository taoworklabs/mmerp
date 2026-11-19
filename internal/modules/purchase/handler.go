package purchase

import "github.com/danielgtaylor/huma/v2"

type actionsOutput struct{ Body Actions }

func registerHandlers(api huma.API, s *Service) {
	registerSupplierHandlers(api, s)
	registerOrderHandlers(api, s)
}
