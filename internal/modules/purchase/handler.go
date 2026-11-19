package purchase

import "github.com/danielgtaylor/huma/v2"

type actionsOutput struct{ Body PurchaseActions }

func registerHandlers(api huma.API, s *Service) {
	registerSupplierHandlers(api, s)
}
