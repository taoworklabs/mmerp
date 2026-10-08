package posting_test

import (
	"context"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

func TestRecordAndVoid(t *testing.T) {
	ctx := platform.WithDB(t.Context(), pgtest.New(t))
	db := platform.DBFrom(ctx)
	var c, dept int64
	if err := db.QueryRow(ctx, `INSERT INTO iam.org_units (kind, name) VALUES ('company', 'C') RETURNING id`).Scan(&c); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO iam.org_units (parent_id, kind, name) VALUES ($1, 'department', 'A') RETURNING id`, c).Scan(&dept); err != nil {
		t.Fatal(err)
	}
	var changed []record.Ref
	s := posting.NewService(posting.Hooks{OnChanged: func(_ context.Context, ref record.Ref) error {
		changed = append(changed, ref)
		return nil
	}})
	ref := record.Ref{Type: "hrm.payroll", ID: 7}
	if err := s.Record(ctx, ref, c, "2026-04-30", []posting.Line{
		{Kind: posting.SalaryExpense, OrgUnitID: dept, Amount: 100}, {Kind: posting.PitPayable, OrgUnitID: c, Amount: -5},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Void(ctx, ref); err != nil {
		t.Fatal(err)
	}
	var lines, voided, sum int64
	if err := db.QueryRow(ctx, `SELECT count(*), count(voided_at), sum(amount) FROM posting.lines WHERE doc_type = 'hrm.payroll' AND doc_id = 7`).Scan(&lines, &voided, &sum); err != nil {
		t.Fatal(err)
	}
	if lines != 2 || voided != 2 || sum != 95 || len(changed) != 2 {
		t.Fatalf("lines %d voided %d sum %d, hook ran %d times", lines, voided, sum, len(changed))
	}
	// Without a hook (no accounting) both still work.
	if err := posting.NewService(posting.Hooks{}).Record(ctx, record.Ref{Type: "hrm.payroll", ID: 8}, c, "2026-04-30", nil); err != nil {
		t.Fatal(err)
	}
}
