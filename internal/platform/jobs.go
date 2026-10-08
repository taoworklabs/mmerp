package platform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
)

// JobSpec says how a job kind is gated and who it runs as.
type JobSpec struct {
	Product string // empty for core jobs
	Class   Class
	// System jobs run with no actor, skip permission checks and always finish.
	// Every other job runs on behalf of the user who enqueued it.
	System bool
	// Notify tells the requester when the job ends; such a job completes with CompleteJob.
	// Never on a system job.
	Notify bool
}

// JobArgs is implemented by the args of every job.
type JobArgs interface {
	river.JobArgs
	Spec() JobSpec
}

// JobMeta is what Enqueue stores in a user job's River metadata; RequestedBy
// is zero for a system job.
type JobMeta struct {
	RequestedBy int64  `json:"requested_by,omitempty"`
	Product     string `json:"product,omitempty"`
	Class       Class  `json:"class,omitempty"`
	Notify      bool   `json:"notify,omitempty"`
}

// JobOutput is the River output of a job that failed with a coded error.
type JobOutput struct {
	Error *Error `json:"error,omitempty"`
}

type jobsKey struct{}

// JobNotifier tells a requester that their job ended; internal/app wires it to the
// notification module, which platform may not know.
type JobNotifier func(ctx context.Context, jobID, requester int64, failed bool) error

type notifierKey struct{}

// WithJobNotifier puts the notifier into ctx; only internal/app calls it.
func WithJobNotifier(ctx context.Context, n JobNotifier) context.Context {
	return context.WithValue(ctx, notifierKey{}, n)
}

// notifyEnd tells the requester of a declaring job that it ended.
func notifyEnd(ctx context.Context, row *rivertype.JobRow, failed bool) error {
	var meta JobMeta
	if err := json.Unmarshal(row.Metadata, &meta); err != nil || !meta.Notify || meta.RequestedBy == 0 {
		return err
	}
	n, ok := ctx.Value(notifierKey{}).(JobNotifier)
	if !ok {
		// Wiring bug, not a runtime condition.
		panic("platform: no job notifier in context")
	}
	return n(ctx, row.ID, meta.RequestedBy, failed)
}

// WithJobs puts the job client into ctx; only internal/app calls it.
func WithJobs(ctx context.Context, c *river.Client[pgx.Tx]) context.Context {
	return context.WithValue(ctx, jobsKey{}, c)
}

// JobsFrom returns the job client, to read jobs back.
func JobsFrom(ctx context.Context) *river.Client[pgx.Tx] {
	c, ok := ctx.Value(jobsKey{}).(*river.Client[pgx.Tx])
	if !ok {
		// Wiring bug, not a runtime condition.
		panic("platform: no job client in context")
	}
	return c
}

// Enqueue is the only way to create a job: it applies the product gate for the
// job's class and records the requester of a user job. It joins the transaction
// in ctx, so the job exists only if that transaction commits.
func Enqueue(ctx context.Context, args JobArgs) (int64, error) {
	spec := args.Spec()
	if err := ProductGate(ctx, spec.Product, spec.Class); err != nil {
		return 0, err
	}
	var meta JobMeta
	if !spec.System {
		actor, ok := ActorFrom(ctx)
		if !ok {
			return 0, errors.New("platform: user job enqueued without an actor")
		}
		meta = JobMeta{RequestedBy: actor, Product: spec.Product, Class: spec.Class, Notify: spec.Notify}
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return 0, err
	}
	opts := &river.InsertOpts{Metadata: b}
	c := JobsFrom(ctx)
	var res *rivertype.JobInsertResult
	if tx := txFrom(ctx); tx != nil {
		res, err = c.InsertTx(ctx, tx, args, opts)
	} else {
		res, err = c.Insert(ctx, args, opts)
	}
	if err != nil {
		return 0, err
	}
	return res.Job.ID, nil
}

// CompleteJob marks a running job done, with its output, in the transaction in ctx: the
// job's writes, its completion and the requester's notification commit together, so a
// crash between them never runs it a second time.
func CompleteJob[T river.JobArgs](ctx context.Context, job *river.Job[T], output any) error {
	tx := txFrom(ctx)
	if tx == nil {
		return errors.New("platform: CompleteJob outside a transaction")
	}
	if _, err := JobsFrom(ctx).JobUpdateTx(ctx, tx, job.ID, &river.JobUpdateParams{Output: output}); err != nil {
		return err
	}
	if err := notifyEnd(ctx, job.JobRow, false); err != nil {
		return err
	}
	_, err := river.JobCompleteTx[*riverpgxv5.Driver](ctx, tx, job)
	return err
}

// JobMiddleware runs every job in the ctx base builds (database, products…). A user
// job runs as its requester and passes the product gate again, since the product may
// have been turned off while it waited; permissions are checked again by the services
// it calls. A coded client error cancels the job instead of retrying it, and is kept
// in the job's output for the API to read back.
func JobMiddleware(base func(context.Context) context.Context) rivertype.WorkerMiddleware {
	return river.WorkerMiddlewareFunc(func(ctx context.Context, job *rivertype.JobRow, doInner func(context.Context) error) error {
		ctx = base(ctx)
		ctx = WithLogger(ctx, LogFrom(ctx).With("job_id", job.ID, "job_kind", job.Kind))
		var meta JobMeta
		if err := json.Unmarshal(job.Metadata, &meta); err != nil {
			return river.JobCancel(err)
		}
		err := func() error {
			if meta.RequestedBy == 0 {
				return doInner(ctx)
			}
			ctx = WithActor(ctx, meta.RequestedBy)
			if err := ProductGate(ctx, meta.Product, meta.Class); err != nil {
				return err
			}
			return doInner(ctx)
		}()
		if err == nil {
			return nil
		}
		e, ok := errors.AsType[*Error](err)
		if !ok || e.Status >= http.StatusInternalServerError {
			// River discards the job after its last attempt.
			if job.Attempt >= job.MaxAttempts {
				if nerr := notifyEnd(ctx, job, true); nerr != nil {
					LogFrom(ctx).Error("job failure not notified", "error", nerr)
				}
			}
			return err
		}
		// Told before River writes the cancellation: a crash in between leaves a spare
		// failure notice for a job that will run again, never a failure untold.
		if err := notifyEnd(ctx, job, true); err != nil {
			return err
		}
		if err := river.RecordOutput(ctx, JobOutput{Error: e}); err != nil {
			return err
		}
		return river.JobCancel(err)
	})
}
