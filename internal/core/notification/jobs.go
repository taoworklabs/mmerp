package notification

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// emailArgs names what to send, never an address or a text: the job's arguments are kept
// with the job and the address is read when it runs.
type emailArgs struct {
	Notification int64 `json:"notification,omitempty"`
	TestTo       int64 `json:"test_to,omitempty"` // a user, for the administrator's test message
}

func (emailArgs) Kind() string           { return "notification.email" }
func (emailArgs) Spec() platform.JobSpec { return platform.JobSpec{System: true} }

type emailWorker struct {
	river.WorkerDefaults[emailArgs]
	s *Service
}

func (w *emailWorker) Work(ctx context.Context, j *river.Job[emailArgs]) error {
	return w.s.sendEmail(ctx, j.Args)
}

func addWorkers(s *Service) func(*river.Workers) {
	return func(w *river.Workers) { river.AddWorker(w, &emailWorker{s: s}) }
}
