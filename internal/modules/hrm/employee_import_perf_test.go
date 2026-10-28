//go:build perf

package hrm_test

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Times a 3,000-row employee import through the job, creating then editing every row.
// Run: go test -tags perf ./internal/modules/hrm -run TestEmployeeImportPerf -v
func TestEmployeeImportPerf(t *testing.T) {
	f := newFixture(t)
	hr := f.ctxFor("hr", "hr@*")
	run := func(name string, phone string) {
		rows := make([][]any, 3000)
		for i := range rows {
			rows[i] = eRow(fmt.Sprintf("P%05d", i), map[int]any{1: fmt.Sprintf("Người %d", i), 4: phone, 7: "A", 9: "01/02/2026"})
		}
		job := f.startImport(hr, rows...)
		f.check(f.jobs.Start(t.Context()))
		defer func() { f.check(f.jobs.Stop(context.Background())) }()
		start := time.Now()
		for {
			j, err := f.jobs.JobGet(f.admin, job)
			f.check(err)
			if j.FinalizedAt != nil {
				if j.State != "completed" {
					t.Fatalf("%s: %s %v", name, j.State, j.Errors)
				}
				t.Logf("%s: %s (job ran %s)", name, time.Since(start), j.FinalizedAt.Sub(*j.AttemptedAt))
				return
			}
			if time.Since(start) > 10*time.Minute {
				t.Fatalf("%s did not end", name)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
	run("3000 creates", "0901")
	run("3000 edits", "0902")
}
