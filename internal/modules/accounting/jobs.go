package accounting

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// reconcileArgs brings the ledger in line with one document's posting lines.
type reconcileArgs struct {
	DocType string `json:"doc_type"`
	DocID   int64  `json:"doc_id"`
}

func (reconcileArgs) Kind() string { return "accounting.reconcile" }

func (reconcileArgs) Spec() platform.JobSpec {
	return platform.JobSpec{Product: "accounting", Class: platform.ClassWrite, System: true}
}

// A burst of changes collapses into one waiting job. A running job also counts: it completes
// in the transaction that reconciles, so a change committed after that one queues a new job.
func (reconcileArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true, ByState: []rivertype.JobState{
		rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning, rivertype.JobStateScheduled, rivertype.JobStateRetryable}}}
}

type reconcileWorker struct {
	river.WorkerDefaults[reconcileArgs]
	s *Service
}

func (w *reconcileWorker) Work(ctx context.Context, j *river.Job[reconcileArgs]) error {
	// Turned off while it waited: the document counts as pending once accounting is back.
	if platform.ProductGate(ctx, "accounting", platform.ClassWrite) != nil {
		return nil
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if err := w.s.reconcile(ctx, record.Ref{Type: j.Args.DocType, ID: j.Args.DocID}); err != nil {
			return err
		}
		return platform.CompleteJob(ctx, j, nil)
	})
}
