package printing

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

func Module(s *Service) platform.Module {
	return platform.Module{
		Name:   "printing",
		Routes: func(api huma.API) { registerHandlers(api, s) },
	}
}
