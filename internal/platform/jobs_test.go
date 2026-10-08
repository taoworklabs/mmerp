package platform_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/riverqueue/river/rivertype"

	"github.com/taoworklabs/mmerp/internal/platform"
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
