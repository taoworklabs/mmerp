package platform_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// A declaring job tells its requester once it fails for good: a coded error at once, any
// other error only on its last attempt. Undeclared and system jobs tell nobody.
func TestJobFailureNotified(t *testing.T) {
	type told struct {
		job, user int64
		failed    bool
	}
	var got []told
	notifier := func(_ context.Context, job, user int64, failed bool) error {
		got = append(got, told{job, user, failed})
		return nil
	}
	mw := platform.JobMiddleware(func(ctx context.Context) context.Context { return platform.WithJobNotifier(ctx, notifier) })
	run := func(id int64, meta string, attempt int, err error) {
		row := &rivertype.JobRow{ID: id, Attempt: attempt, MaxAttempts: 3, Metadata: []byte(meta)}
		_ = mw.Work(t.Context(), row, func(context.Context) error { return err })
	}
	coded := &platform.Error{Status: http.StatusUnprocessableEntity, Code: "import_empty"}
	plain := errors.New("disk full")
	declared := `{"requested_by":7,"notify":true}`

	run(1, declared, 1, plain)
	run(2, declared, 3, plain)
	run(3, declared, 1, coded)
	run(4, declared, 1, nil)
	run(5, `{"requested_by":7}`, 3, plain)
	run(6, `{}`, 3, plain)
	if len(got) != 2 || got[0] != (told{2, 7, true}) || got[1] != (told{3, 7, true}) {
		t.Fatalf("told %+v", got)
	}
}

type noopArgs struct{}

func (noopArgs) Kind() string           { return "test.noop" }
func (noopArgs) Spec() platform.JobSpec { return platform.JobSpec{System: true} }

// A job enqueued with no transaction in ctx would exist even if the business write
// it belongs to rolled back, so Enqueue refuses it.
func TestEnqueueRequiresTx(t *testing.T) {
	pool := pgtest.New(t)
	jobs, err := river.NewClient(riverpgxv5.New(pgtest.System(t, pool)), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := platform.WithJobs(platform.WithDB(t.Context(), pool), jobs)
	jobCount := func() (n int) {
		t.Helper()
		if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM river_job`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	if _, err := platform.Enqueue(ctx, noopArgs{}); !errors.Is(err, platform.ErrEnqueueOutsideTx) {
		t.Fatalf("err = %v, want ErrEnqueueOutsideTx", err)
	}
	if n := jobCount(); n != 0 {
		t.Fatalf("%d jobs, want 0", n)
	}
	if err := platform.InTx(ctx, func(ctx context.Context) error {
		_, err := platform.Enqueue(ctx, noopArgs{})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if n := jobCount(); n != 1 {
		t.Fatalf("%d jobs, want 1", n)
	}
}

type userArgs struct{}

func (userArgs) Kind() string           { return "test.user" }
func (userArgs) Spec() platform.JobSpec { return platform.JobSpec{Notify: true} }

type flagWorker[T platform.JobArgs] struct {
	river.WorkerDefaults[T]
	ran *atomic.Bool
}

func (w *flagWorker[T]) Work(context.Context, *river.Job[T]) error {
	w.ran.Store(true)
	return nil
}

// workRaw inserts args with empty metadata, as River does for periodic jobs and as jobs
// queued before requesters were recorded have, works it, and returns the ended job,
// whether the worker ran and how many notifications were sent.
func workRaw[T platform.JobArgs](t *testing.T, args T) (*rivertype.JobRow, bool, int) {
	t.Helper()
	pool := pgtest.New(t)
	var ran atomic.Bool
	var told atomic.Int32
	workers := river.NewWorkers()
	platform.AddWorker(workers, &flagWorker[T]{ran: &ran})
	notifier := func(context.Context, int64, int64, bool) error { told.Add(1); return nil }
	jobs, err := river.NewClient(riverpgxv5.New(pgtest.System(t, pool)), &river.Config{
		Queues: map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}}, Workers: workers,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Middleware: []rivertype.Middleware{platform.JobMiddleware(func(ctx context.Context) context.Context { return platform.WithJobNotifier(ctx, notifier) })},
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := jobs.Insert(t.Context(), args, &river.InsertOpts{Metadata: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if err := jobs.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = jobs.Stop(context.Background()) }()
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		j, err := jobs.JobGet(t.Context(), res.Job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if j.FinalizedAt != nil {
			return j, ran.Load(), int(told.Load())
		}
	}
	t.Fatal("job did not end")
	return nil, false, 0
}

// A user job with no requester would run with no actor and skip every permission check,
// so it is cancelled on its first attempt without running.
func TestUserJobWithoutRequesterCancelled(t *testing.T) {
	j, ran, told := workRaw(t, userArgs{})
	if j.State != rivertype.JobStateCancelled || j.Attempt != 1 || ran || told != 0 {
		t.Fatalf("state %s, attempt %d, ran %v, told %d", j.State, j.Attempt, ran, told)
	}
}

// A system job runs whatever its metadata, including jobs queued before an upgrade.
func TestSystemJobWithEmptyMetadataRuns(t *testing.T) {
	j, ran, _ := workRaw(t, noopArgs{})
	if j.State != rivertype.JobStateCompleted || !ran {
		t.Fatalf("state %s, ran %v", j.State, ran)
	}
}
