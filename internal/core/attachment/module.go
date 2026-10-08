package attachment

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

func Module(s *Service) platform.Module {
	return platform.Module{
		Name:     "attachment",
		Routes:   func(api huma.API) { registerHandlers(api, s) },
		Workers:  addWorkers,
		Periodic: periodic,
	}
}
