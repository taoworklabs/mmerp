package purchase_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// printer works print jobs for the fixture's users.
type printer struct {
	*payments
	jobs  *river.Client[pgx.Tx]
	files platform.Files
}

func newPrinter(t *testing.T) *printer {
	p := &printer{payments: newPayments(t), files: platform.Files{Dir: t.TempDir()}}
	workers := river.NewWorkers()
	dataio.Module(p.core.DataIO).Workers(workers)
	base := func(c context.Context) context.Context {
		c = platform.WithJobNotifier(platform.WithKeyring(platform.WithProducts(platform.WithDB(c, p.core.Pool), []string{"purchase"}), pgtest.Keyring()), notification.JobEnded)
		return platform.WithFiles(platform.WithJobs(c, p.jobs), p.files)
	}
	jobs, err := river.NewClient(riverpgxv5.New(p.core.System), &river.Config{
		Queues: map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 2}}, Workers: workers,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Middleware: []rivertype.Middleware{platform.JobMiddleware(base)},
	})
	p.check(err)
	p.jobs = jobs
	return p
}

// print exports docType id as ctx and returns the PDF, or nil when the job ends otherwise than want.
func (p *printer) print(ctx context.Context, docType string, id int64, want string) []byte {
	p.t.Helper()
	ctx = platform.WithFiles(platform.WithJobs(ctx, p.jobs), p.files)
	job, err := p.core.DataIO.StartExport(ctx, "printing."+docType, fmt.Appendf(nil, `{"id":%d}`, id))
	p.check(err)
	p.check(p.jobs.Start(p.t.Context()))
	defer func() { p.check(p.jobs.Stop(context.Background())) }()
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		j, err := p.jobs.JobGet(p.admin, job)
		p.check(err)
		if j.FinalizedAt == nil {
			continue
		}
		got := string(j.State)
		var out platform.JobOutput
		if json.Unmarshal(j.Output(), &out) == nil && out.Error != nil {
			got = out.Error.Code
		}
		if got != want {
			p.t.Fatalf("%s %d: %s, want %s (%v)", docType, id, got, want, j.Errors)
		}
		if want != "completed" {
			return nil
		}
		dj, err := p.core.DataIO.Job(ctx, job)
		p.check(err)
		_, file, err := p.core.DataIO.File(ctx, *dj.FileID)
		p.check(err)
		defer func() { p.check(file.Close()) }()
		b, err := io.ReadAll(file)
		p.check(err)
		return b
	}
	p.t.Fatalf("%s %d: job never ended", docType, id)
	return nil
}

func (p *printer) snapshots(id int64) (n int) {
	p.t.Helper()
	p.check(p.core.Pool.QueryRow(p.ctx, `SELECT count(*) FROM printing.snapshots WHERE doc_id = $1`, id).Scan(&n))
	return n
}

// Every Purchasing document prints, and posting freezes what it prints.
func TestPrintEveryDocument(t *testing.T) {
	p := newPrinter(t)
	receipt := must(t)(p.receive(p.order, "2026-03-11", lotLine(1, "6", "L1", str("2027-01-31")), serialLine(2, "S1")))
	p.check(p.move(p.staff, "purchase.goods_receipt", receipt, record.Posted))
	pay := must(t)(p.pay(p.staff, p.fields("2026-03-14", "bank", alloc(p.inv1, p.t1), alloc(p.inv2, 1_000))))
	p.check(p.move(p.staff, "purchase.payment", pay, record.Posted))
	for docType, id := range map[string]int64{"purchase.order": p.order, "purchase.goods_receipt": receipt, "purchase.invoice": p.inv1, "purchase.payment": pay} {
		if p.snapshots(id) != 1 {
			t.Errorf("%s: no frozen print", docType)
		}
		if b := p.print(p.staff, docType, id, "completed"); !bytes.HasPrefix(b, []byte("%PDF")) {
			t.Errorf("%s: no PDF", docType)
		}
	}
}

func TestPrintNeedsView(t *testing.T) {
	p := newPrinter(t)
	pay := must(t)(p.pay(p.staff, p.fields("2026-03-14", "cash", alloc(p.inv1, 1_000))))
	p.print(p.ctxFor("staff_b", "staff@B"), "purchase.payment", pay, "not_found")
}
