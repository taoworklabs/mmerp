package posting_test

import (
	"context"
	"errors"
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
	document := func() record.Ref {
		var id int64
		if err := db.QueryRow(ctx, `INSERT INTO record.documents (doc_type, number, status, version, date, legal_entity_id, org_unit_id)
			VALUES ('hrm.payroll', 'P', 'posted', 1, '2026-04-30', $1, $1) RETURNING id`, c).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return record.Ref{Type: "hrm.payroll", ID: id}
	}
	var changed []record.Ref
	s := posting.NewService(posting.Hooks{OnChanged: func(_ context.Context, ref record.Ref) error {
		changed = append(changed, ref)
		return nil
	}})
	ref := document()
	lines := []posting.Line{{Kind: posting.SalaryExpense, OrgUnitID: dept, Amount: 100}, {Kind: posting.PitPayable, OrgUnitID: c, Amount: -5}}
	if err := s.Record(ctx, ref, lines); err != nil {
		t.Fatal(err)
	}
	// A second recording would double the lines.
	if err := s.Record(ctx, ref, lines); !errors.Is(err, posting.ErrAlreadyRecorded) {
		t.Fatalf("second record: %v", err)
	}
	if err := s.Void(ctx, ref); err != nil {
		t.Fatal(err)
	}
	var count, voided, sum, le int64
	var date string
	if err := db.QueryRow(ctx, `SELECT count(*), count(voided_at), sum(amount), min(legal_entity_id), min(date)::text FROM posting.lines WHERE doc_type = $1 AND doc_id = $2`,
		ref.Type, ref.ID).Scan(&count, &voided, &sum, &le, &date); err != nil {
		t.Fatal(err)
	}
	if count != 2 || voided != 2 || sum != 95 || le != c || date != "2026-04-30" || len(changed) != 2 {
		t.Fatalf("lines %d voided %d sum %d le %d date %s, hook ran %d times", count, voided, sum, le, date, len(changed))
	}
	// Once voided, the document may be recorded again.
	if err := s.Record(ctx, ref, lines); err != nil {
		t.Fatal(err)
	}
	if err := s.Record(ctx, record.Ref{Type: "hrm.payroll", ID: -1}, lines); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("unknown document: %v", err)
	}
	// Without a hook (no accounting) both still work.
	if err := posting.NewService(posting.Hooks{}).Record(ctx, document(), nil); err != nil {
		t.Fatal(err)
	}
}
