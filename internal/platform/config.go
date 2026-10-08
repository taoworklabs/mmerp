package platform

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Config is read from the environment once at startup.
type Config struct {
	DatabaseURL string
	HTTPAddr    string
	LogLevel    slog.Level
	Products    []string // enabled products; internal/app checks the names
	Keys        *Keyring
	FilesDir    string // must be backed up with the database
	// RunJobs false: this instance enqueues jobs but never works them, e.g. a second
	// instance in tests that runs with other products on the same database.
	RunJobs bool
}

// LoadConfig reads and validates every variable, reporting all problems at once.
func LoadConfig() (Config, error) {
	var errs []error
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		HTTPAddr:    envOr("HTTP_ADDR", ":8080"),
		FilesDir:    envOr("FILES_DIR", "data/files"),
		RunJobs:     envOr("RUN_JOBS", "true") != "false",
	}
	for p := range strings.SplitSeq(os.Getenv("PRODUCTS"), ",") {
		if p = strings.TrimSpace(p); p != "" {
			cfg.Products = append(cfg.Products, p)
		}
	}
	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	var err error
	if cfg.Keys, err = ParseKeys(os.Getenv("ENCRYPTION_KEYS")); err != nil {
		errs = append(errs, fmt.Errorf("ENCRYPTION_KEYS: %w", err))
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(envOr("LOG_LEVEL", "info"))); err != nil {
		errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
	}
	return cfg, errors.Join(errs...)
}

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
