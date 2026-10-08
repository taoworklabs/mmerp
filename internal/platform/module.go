package platform

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/riverqueue/river"
)

// Module is the manifest internal/app mounts. It holds only platform and library
// types; registration with core modules happens through their own functions.
type Module struct {
	Name    string
	Product string // empty for core and shared modules
	Routes  func(api huma.API)
	// Middleware wraps every /api request, e.g. to resolve the session.
	Middleware func(http.Handler) http.Handler
	// Workers adds the module's job workers.
	Workers func(w *river.Workers)
	// Periodic lists the module's periodic jobs; each is a system job written to
	// catch up on whatever is left, not on what happened since its last run.
	Periodic []*river.PeriodicJob
}

// Operation metadata keys read by internal/app.
const (
	// MetaPublic marks an operation that needs no session (true).
	MetaPublic = "public"
	// MetaClass sets the action class; without it GET is read and the rest is write.
	MetaClass = "class"
)
