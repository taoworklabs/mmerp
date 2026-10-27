package approval

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/riverqueue/river"

	"github.com/taoworklabs/mmerp/internal/platform"
)

func Module(s *Service) platform.Module {
	return platform.Module{
		Name:     "approval",
		Routes:   func(api huma.API) { registerHandlers(api, s) },
		Workers:  func(w *river.Workers) { platform.AddWorker(w, &overdueWorker{s: s}) },
		Periodic: periodic,
	}
}
