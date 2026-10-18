// Package coretest wires the whole core on a fresh test database, the same way the
// composition root does, so a module test starts without copying NewService calls.
package coretest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/attachment"
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/discussion"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/printing"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// Core is a wired core on its own database.
type Core struct {
	Pool         *pgxpool.Pool
	Ctx          context.Context // database, products and keyring; no actor
	Admin        context.Context // Ctx acting as the tenant administrator "admin"
	AdminID      int64
	Audit        *audit.Service
	Setting      *setting.Service
	IAM          *iam.Service
	Numbering    *numbering.Service
	Record       *record.Service
	Notification *notification.Service
	Approval     *approval.Service
	DataIO       *dataio.Service
	Printing     *printing.Service
	Attachment   *attachment.Service
	Discussion   *discussion.Service
}

// New wires the core with the given products enabled and creates the administrator.
func New(t *testing.T, products ...string) *Core {
	t.Helper()
	pool := pgtest.New(t)
	ctx := platform.WithKeyring(platform.WithProducts(platform.WithDB(t.Context(), pool), products), pgtest.Keyring())
	aud := audit.NewService()
	set := setting.NewService()
	ids := iam.NewService(iam.Deps{Setting: set, Audit: aud})
	set.SetAuthz(ids)
	num := numbering.NewService(numbering.Deps{IAM: ids, Audit: aud})
	rec := record.NewService(record.Deps{IAM: ids, Numbering: num, Audit: aud})
	ids.SetTreeHook(rec)
	ids.SetDataProducts(rec)
	ntf := notification.NewService(notification.Deps{Record: rec, IAM: ids, Audit: aud, Setting: set})
	appr := approval.NewService(approval.Deps{IAM: ids, Record: rec, Audit: aud, Notification: ntf})
	rec.SetApprovalGate(appr)
	dio := dataio.NewService(dataio.Deps{IAM: ids, Audit: aud})
	c := &Core{
		Pool: pool, Ctx: ctx, Audit: aud, Setting: set, IAM: ids, Numbering: num, Record: rec,
		Notification: ntf, Approval: appr, DataIO: dio,
		Printing:   printing.NewService(printing.Deps{Record: rec, Audit: aud, DataIO: dio, IAM: ids}),
		Attachment: attachment.NewService(attachment.Deps{Record: rec, Audit: aud}),
		Discussion: discussion.NewService(discussion.Deps{Record: rec, Audit: aud, IAM: ids, Notification: ntf}),
	}
	id, err := ids.CreateAdmin(ctx, "admin", "Admin", "long enough")
	if err != nil {
		t.Fatal(err)
	}
	c.AdminID, c.Admin = id, platform.WithActor(ctx, id)
	return c
}
