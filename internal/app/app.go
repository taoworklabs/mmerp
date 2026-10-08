// Package app is the composition root: it wires every module into one HTTP handler.
package app

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/attachment"
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/discussion"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

// Version is set at build time.
var Version = "dev"

// products maps every product to the products it depends on.
var products = map[string][]string{
	"hrm": nil,
}

// CheckProducts refuses unknown products and enabled products missing a dependency.
func CheckProducts(enabled []string) error {
	for _, p := range enabled {
		deps, ok := products[p]
		if !ok {
			return fmt.Errorf("PRODUCTS: unknown product %q", p)
		}
		for _, d := range deps {
			if !slices.Contains(enabled, d) {
				return fmt.Errorf("PRODUCTS: %q needs %q", p, d)
			}
		}
	}
	return nil
}

func newIAM(set *setting.Service) *iam.Service {
	ids := iam.NewService(iam.Deps{Setting: set, Audit: audit.NewService()})
	set.SetAuthz(ids)
	return ids
}

// Modules lists every module; all are mounted regardless of enabled products.
func Modules() []platform.Module {
	set := setting.NewService()
	ids := newIAM(set)
	rec := record.NewService(record.Deps{IAM: ids, Numbering: numbering.NewService(), Audit: audit.NewService()})
	ids.SetTreeHook(rec)
	ids.SetDataProducts(rec)
	ntf := notification.NewService(notification.Deps{Record: rec, IAM: ids, Audit: audit.NewService(), Setting: set})
	appr := approval.NewService(approval.Deps{IAM: ids, Record: rec, Audit: audit.NewService(), Notification: ntf})
	rec.SetApprovalGate(appr)
	dio := dataio.NewService(dataio.Deps{IAM: ids, Audit: audit.NewService()})
	att := attachment.NewService(attachment.Deps{Record: rec, Audit: audit.NewService()})
	dis := discussion.NewService(discussion.Deps{Record: rec, Audit: audit.NewService(), IAM: ids, Notification: ntf})
	h := hrm.NewService(hrm.Deps{IAM: ids, Record: rec, Audit: audit.NewService(), Setting: set, DataIO: dio,
		// No accounting yet: nothing reacts to posting lines.
		Posting: posting.NewService(posting.Hooks{})})
	return []platform.Module{iam.Module(ids), setting.Module(set), record.Module(rec), approval.Module(appr), dataio.Module(dio), attachment.Module(att), discussion.Module(dis), notification.Module(ntf), hrm.Module(h)}
}

// CreateAdmin adds the first administrator at install time.
func CreateAdmin(ctx context.Context, pool *pgxpool.Pool, login, name, password string) error {
	_, err := newIAM(setting.NewService()).CreateAdmin(platform.WithDB(ctx, pool), login, name, password)
	return err
}

// Env is what every request and job runs with. Single-tenant mode: one database, one product list, one keyring.
type Env struct {
	Pool     *pgxpool.Pool
	Log      *slog.Logger
	Products []string
	Keys     *platform.Keyring
	Files    platform.Files
	Jobs     *river.Client[pgx.Tx]
}

func (env Env) context(ctx context.Context) context.Context {
	ctx = platform.WithLogger(platform.WithDB(ctx, env.Pool), env.Log)
	ctx = platform.WithKeyring(platform.WithProducts(ctx, env.Products), env.Keys)
	ctx = platform.WithJobNotifier(ctx, notification.JobEnded)
	return platform.WithJobs(platform.WithFiles(ctx, env.Files), env.Jobs)
}

// maxRequestBytes is above every operation's own limit (an attachment is 20 MB).
const maxRequestBytes = 32 << 20

// jobRetention keeps finished jobs long enough for users to read their results back.
const jobRetention = 7 * 24 * time.Hour

// NewJobs builds the job client with every module's workers; env.Jobs is set to it
// for the jobs it runs. Start it to work jobs; unstarted, it only enqueues.
func NewJobs(env Env, modules []platform.Module) (*river.Client[pgx.Tx], error) {
	workers := river.NewWorkers()
	var periodic []*river.PeriodicJob
	for _, m := range modules {
		if m.Workers != nil {
			m.Workers(workers)
		}
		periodic = append(periodic, m.Periodic...)
	}
	client, err := river.NewClient(riverpgxv5.New(env.Pool), &river.Config{
		Queues:                      map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 10}},
		Workers:                     workers,
		PeriodicJobs:                periodic,
		Logger:                      env.Log,
		CompletedJobRetentionPeriod: jobRetention,
		CancelledJobRetentionPeriod: jobRetention,
		DiscardedJobRetentionPeriod: jobRetention,
		// env is captured by reference: Jobs is set below, before any job runs.
		Middleware: []rivertype.Middleware{platform.JobMiddleware(func(ctx context.Context) context.Context { return env.context(ctx) })},
	})
	env.Jobs = client
	return client, err
}

// New builds the HTTP handler. webFS is the built frontend, nil when it is not embedded.
func New(env Env, modules []platform.Module, webFS fs.FS) http.Handler {
	pool, log := env.Pool, env.Log
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer)
	// Huma bounds JSON bodies but not multipart ones; nothing takes more than this.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			req.Body = http.MaxBytesReader(w, req.Body, maxRequestBytes)
			next.ServeHTTP(w, req)
		})
	})
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			l := log.With("request_id", middleware.GetReqID(req.Context()))
			ctx := platform.WithLogger(env.context(req.Context()), l)
			ww := middleware.NewWrapResponseWriter(w, req.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, req.WithContext(ctx))
			l.Info("request", "method", req.Method, "path", req.URL.Path, "status", ww.Status(), "duration_ms", time.Since(start).Milliseconds())
		})
	})

	r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
		if err := pool.Ping(req.Context()); err != nil {
			platform.LogFrom(req.Context()).Error("healthz: database unreachable", "err", err)
			http.Error(w, "database_unreachable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api", func(r chi.Router) {
		for _, m := range modules {
			if m.Middleware != nil {
				r.Use(m.Middleware)
			}
		}
		newAPI(r, modules)
	})

	if webFS != nil {
		r.Handle("/*", spa(webFS))
	}
	return r
}

func newAPI(r chi.Router, modules []platform.Module) huma.API {
	cfg := huma.DefaultConfig("mmerp", Version)
	cfg.Servers = []*huma.Server{{URL: "/api"}}
	api := humachi.New(r, cfg)
	api.UseMiddleware(requireSession(api))
	for _, m := range modules {
		if m.Routes == nil {
			continue
		}
		g := huma.NewGroup(api)
		// The tag splits the spec into one frontend client per product.
		tag := cmp.Or(m.Product, "core")
		g.UseSimpleModifier(func(op *huma.Operation) { op.Tags = append(op.Tags, tag) })
		if m.Product != "" {
			g.UseMiddleware(productGate(api, m.Product))
		}
		m.Routes(g)
	}
	return api
}

func requireSession(api huma.API) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if _, ok := platform.ActorFrom(ctx.Context()); ok || ctx.Operation().Metadata[platform.MetaPublic] == true {
			next(ctx)
			return
		}
		_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "")
	}
}

func productGate(api huma.API, product string) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		op := ctx.Operation()
		class, ok := op.Metadata[platform.MetaClass].(platform.Class)
		if !ok && op.Method != http.MethodGet {
			class = platform.ClassWrite
		}
		if err := platform.ProductGate(ctx.Context(), product, class); err != nil {
			_ = huma.WriteErr(api, ctx, http.StatusForbidden, "", err)
			return
		}
		next(ctx)
	}
}

// OpenAPI returns the spec of every module's routes without needing a database.
func OpenAPI() ([]byte, error) {
	return json.MarshalIndent(newAPI(chi.NewRouter(), Modules()).OpenAPI(), "", "  ")
}
