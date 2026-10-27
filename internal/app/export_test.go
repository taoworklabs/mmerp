package app

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// CreateAdmin adds a tenant administrator, as tests need one besides the root.
func CreateAdmin(ctx context.Context, pool *pgxpool.Pool, login, name, password string) error {
	_, err := newIAM(setting.NewService()).CreateAdmin(platform.WithDB(ctx, pool), login, name, password)
	return err
}
