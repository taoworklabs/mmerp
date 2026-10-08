package platform

import (
	"context"
	"log/slog"
	"os"
)

type loggerKey struct{}

// NewLogger writes JSON to stdout. Never log personal data.
func NewLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

// LogFrom returns the request-scoped logger, or the default one outside a request.
func LogFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
