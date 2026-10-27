package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taoworklabs/mmerp/internal/app"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/migrations"
	"github.com/taoworklabs/mmerp/web"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "mmerp:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := platform.LoadConfig()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := app.CheckProducts(cfg.Products); err != nil {
		return err
	}
	log := platform.NewLogger(cfg.LogLevel)

	tracer := platform.NewQueryTracer(log, cfg.SlowQuery)
	pool, err := platform.NewPool(ctx, cfg.DatabaseURL, platform.AppPool, tracer)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()
	system, err := platform.NewPool(ctx, cfg.DatabaseURL, platform.SystemPool, tracer)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer system.Close()

	// HTTP opens only after the schema is current.
	if err := platform.Migrate(ctx, system, migrations.FS); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	log.Info("migrations applied")
	if err := app.EnsureRoot(ctx, pool, cfg.RootPassword); err != nil {
		return fmt.Errorf("root account: %w", err)
	}

	if len(os.Args) > 1 {
		return command(ctx, pool, os.Args[1:])
	}

	if err := os.MkdirAll(cfg.FilesDir, 0o750); err != nil {
		return fmt.Errorf("files: %w", err)
	}
	env := app.Env{Pool: pool, SystemPool: system, Log: log, Products: cfg.Products, Keys: cfg.Keys, Files: platform.Files{Dir: cfg.FilesDir}}
	modules := app.Modules()
	if env.Jobs, err = app.NewJobs(env, modules); err != nil {
		return fmt.Errorf("jobs: %w", err)
	}
	if cfg.RunJobs {
		if err := env.Jobs.Start(ctx); err != nil {
			return fmt.Errorf("jobs: %w", err)
		}
	}

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: app.New(env, modules, web.FS()), ReadHeaderTimeout: 10 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("listening", "addr", cfg.HTTPAddr, "version", app.Version)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	if !cfg.RunJobs {
		return nil
	}
	// Jobs still running past the timeout are rescued and run again later.
	return env.Jobs.Stop(shutdownCtx)
}

const usage = "usage: mmerp [reset-password <login>]  (password on stdin)"

func command(ctx context.Context, pool *pgxpool.Pool, args []string) error {
	if len(args) != 2 || args[0] != "reset-password" {
		return errors.New(usage)
	}
	// Read from stdin so the password stays out of shell history and the process list.
	password, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && password == "" {
		return fmt.Errorf("reset-password: password on stdin: %w", err)
	}
	if err := app.RecoverPassword(ctx, pool, args[1], strings.TrimRight(password, "\r\n")); err != nil {
		return fmt.Errorf("reset-password: %w", err)
	}
	fmt.Fprintln(os.Stderr, "reset-password:", args[1], "done")
	return nil
}
