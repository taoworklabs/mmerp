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

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()

	// HTTP opens only after the schema is current.
	if err := platform.Migrate(ctx, pool, migrations.FS); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	log.Info("migrations applied")

	if len(os.Args) > 1 {
		return command(ctx, pool, os.Args[1:])
	}

	if err := os.MkdirAll(cfg.FilesDir, 0o750); err != nil {
		return fmt.Errorf("files: %w", err)
	}
	env := app.Env{Pool: pool, Log: log, Products: cfg.Products, Keys: cfg.Keys, Files: platform.Files{Dir: cfg.FilesDir}}
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

const usage = "usage: mmerp [create-admin <login> <name>]  (password on stdin)"

func command(ctx context.Context, pool *pgxpool.Pool, args []string) error {
	if len(args) != 3 || args[0] != "create-admin" {
		return errors.New(usage)
	}
	// Read from stdin so the password stays out of shell history and the process list.
	password, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && password == "" {
		return fmt.Errorf("create-admin: password on stdin: %w", err)
	}
	if err := app.CreateAdmin(ctx, pool, args[1], args[2], strings.TrimRight(password, "\r\n")); err != nil {
		return fmt.Errorf("create-admin: %w", err)
	}
	fmt.Fprintln(os.Stderr, "create-admin: user", args[1], "created")
	return nil
}
