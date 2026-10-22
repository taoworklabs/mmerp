package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/modules/hrm"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

// A document's row, its audit row, its number and the job it queues commit or roll
// back together with the transaction that wrote them.
func TestWriteAuditJobCommitTogether(t *testing.T) {
	core := coretest.New(t, "hrm")
	h := hrm.NewService(hrm.Deps{IAM: core.IAM, Record: core.Record, Audit: core.Audit, Setting: core.Setting, DataIO: core.DataIO,
		Printing: core.Printing, Posting: posting.NewService(posting.Hooks{}), Numbering: core.Numbering})
	jobs := env(t, core.Pool, []string{"hrm"}).Jobs
	admin := platform.WithJobs(core.Admin, jobs)
	company, err := core.IAM.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"})
	if err != nil {
		t.Fatal(err)
	}
	pay, err := core.IAM.CreateUser(admin, "pay", "pay", "long enough")
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"payroll", "hr"} {
		if _, err := core.IAM.GrantRole(admin, pay, "hrm", role, nil); err != nil {
			t.Fatal(err)
		}
	}
	ctx := platform.WithActor(admin, pay)

	counts := func() (n [4]int) {
		t.Helper()
		err := core.Pool.QueryRow(t.Context(), `SELECT
			(SELECT count(*) FROM record.documents WHERE doc_type = 'hrm.payroll'),
			(SELECT count(*) FROM audit.log WHERE action = 'record.created'),
			(SELECT count(*) FROM numbering.counters WHERE doc_type = 'hrm.payroll'),
			(SELECT count(*) FROM river_job WHERE kind = 'hrm.payroll_compute')`).Scan(&n[0], &n[1], &n[2], &n[3])
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	write := func(fail error) error {
		return platform.InTx(ctx, func(ctx context.Context) error {
			if _, err := h.CreatePayroll(ctx, hrm.NewPayroll{LegalEntityID: company, Month: "2026-03"}); err != nil {
				return err
			}
			return fail
		})
	}

	boom := errors.New("boom")
	if err := write(boom); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if n := counts(); n != [4]int{} {
		t.Fatalf("after rollback: document, audit, number, job = %v, want none", n)
	}
	if err := write(nil); err != nil {
		t.Fatal(err)
	}
	if n := counts(); n != [4]int{1, 1, 1, 1} {
		t.Fatalf("after commit: document, audit, number, job = %v, want one each", n)
	}
}
