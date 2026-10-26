package approval

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/taoworklabs/mmerp/internal/core/approval/internal/store"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type overdueArgs struct{}

func (overdueArgs) Kind() string           { return "approval.overdue" }
func (overdueArgs) Spec() platform.JobSpec { return platform.JobSpec{System: true} }

type overdueWorker struct {
	river.WorkerDefaults[overdueArgs]
	s *Service
}

func (w *overdueWorker) Work(ctx context.Context, _ *river.Job[overdueArgs]) error {
	_, err := w.s.NotifyOverdue(ctx)
	return err
}

// periodic looks for overdue steps every quarter hour and at start, so a step that
// fell due while the app was down is announced at the next start.
var periodic = []*river.PeriodicJob{river.NewPeriodicJob(river.PeriodicInterval(15*time.Minute),
	func() (river.JobArgs, *river.InsertOpts) { return overdueArgs{}, nil }, &river.PeriodicJobOpts{RunOnStart: true})}

// NotifyOverdue tells the pending approvers (with their delegates) and the submitter of
// every open step past its due time, once per step, each step in its own transaction.
// It returns how many steps it announced.
// ponytail: a step whose notice keeps failing stops each run at it; mark it told on error if one ever does.
func (s *Service) NotifyOverdue(ctx context.Context) (int, error) {
	n := 0
	for {
		found, told := false, false
		err := platform.InTx(ctx, func(ctx context.Context) error {
			st, err := store.ClaimOverdue.Run(ctx)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			found = true
			told, err = s.announceOverdue(ctx, st)
			return err
		})
		if err != nil || !found {
			return n, err
		}
		if told {
			n++
		}
	}
}

// announceOverdue tells who is waited on that st is overdue; a submission whose document
// moved on is stale and tells nobody, though its claim stays so it is not looked at again.
func (s *Service) announceOverdue(ctx context.Context, st store.ClaimOverdueRow) (bool, error) {
	inst, err := store.GetInstance.Run(ctx, st.InstanceID)
	if err != nil {
		return false, err
	}
	d, err := s.d.Record.Get(ctx, record.Ref{Type: inst.DocType, ID: inst.DocID})
	if errors.Is(err, platform.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !current(inst, d) {
		return false, nil
	}
	votes, err := store.CastVotes.Run(ctx, store.CastVotesParams{InstanceID: st.InstanceID, Position: st.Position})
	if err != nil {
		return false, err
	}
	pending := slices.DeleteFunc(slices.Clone(st.Approvers), func(id int64) bool {
		return slices.ContainsFunc(votes, func(v store.CastVotesRow) bool { return v.UserID == id })
	})
	excluded, err := s.excluded(ctx, inst, d.Ref)
	if err != nil {
		return false, err
	}
	if len(pending) > 0 {
		if err := s.requestVotes(ctx, notification.ApprovalOverdue, d, excluded, pending); err != nil {
			return false, err
		}
	}
	return true, s.d.Notification.Send(ctx, notification.ApprovalOverdue, d.Ref, []int64{inst.SubmittedBy})
}
