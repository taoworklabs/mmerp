package attachment

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/taoworklabs/mmerp/internal/core/attachment/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type cleanupArgs struct{}

func (cleanupArgs) Kind() string           { return "attachment.cleanup" }
func (cleanupArgs) Spec() platform.JobSpec { return platform.JobSpec{System: true} }

type cleanupWorker struct {
	river.WorkerDefaults[cleanupArgs]
}

func (w *cleanupWorker) Work(ctx context.Context, _ *river.Job[cleanupArgs]) error {
	return cleanup(ctx)
}

func addWorkers(w *river.Workers) { river.AddWorker(w, &cleanupWorker{}) }

// periodic runs cleanup every hour and at start, so files left while the app was
// down go at the next start.
var periodic = []*river.PeriodicJob{river.NewPeriodicJob(river.PeriodicInterval(time.Hour),
	func() (river.JobArgs, *river.InsertOpts) { return cleanupArgs{}, nil }, &river.PeriodicJobOpts{RunOnStart: true})}

// cleanup removes files older than a day that no row names: removed attachments, and
// uploads whose row was rolled back. A day keeps every file the last daily backup of
// the database may name.
func cleanup(ctx context.Context) error {
	return files(ctx).Sweep(ctx, time.Now().Add(-24*time.Hour), store.New(platform.DBFrom(ctx)).KnownFiles)
}
