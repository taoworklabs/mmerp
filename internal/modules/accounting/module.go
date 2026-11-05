package accounting

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

func Module(s *Service) platform.Module {
	return platform.Module{
		Name:    "accounting",
		Product: "accounting",
		Routes:  func(api huma.API) { registerAccountHandlers(api, s); registerRuleHandlers(api, s) },
	}
}
