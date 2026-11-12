package posting_test

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

func TestRecordAndVoid(t *testing.T) {
	pool := pgtest.New(t)
	ctx := platform.WithDB(t.Context(), pool)
	db := pgtest.System(t, pool)
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
	// Record and Void run in their caller's transaction.
	tx := func(fn func(context.Context) error) error { return platform.InTx(ctx, fn) }
	var changed []record.Ref
	s := posting.NewService(posting.Hooks{OnChanged: func(_ context.Context, ref record.Ref) error {
		changed = append(changed, ref)
		return nil
	}})
	ref := document()
	lines := []posting.Line{{Kind: posting.SalaryExpense, OrgUnitID: dept, Amount: 100}, {Kind: posting.PitPayable, OrgUnitID: c, Amount: -5}}
	if err := tx(func(ctx context.Context) error { return s.Record(ctx, ref, lines) }); err != nil {
		t.Fatal(err)
	}
	// A second recording would double the lines.
	if err := tx(func(ctx context.Context) error { return s.Record(ctx, ref, lines) }); !errors.Is(err, posting.ErrAlreadyRecorded) {
		t.Fatalf("second record: %v", err)
	}
	if err := tx(func(ctx context.Context) error { return s.Void(ctx, ref) }); err != nil {
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
	if err := tx(func(ctx context.Context) error { return s.Record(ctx, ref, lines) }); err != nil {
		t.Fatal(err)
	}
	if err := tx(func(ctx context.Context) error { return s.Record(ctx, record.Ref{Type: "hrm.payroll", ID: -1}, lines) }); !errors.Is(err, platform.ErrNotFound) {
		t.Fatalf("unknown document: %v", err)
	}
	// Without a hook (no accounting) both still work.
	bare := posting.NewService(posting.Hooks{})
	if err := tx(func(ctx context.Context) error { return bare.Record(ctx, document(), nil) }); err != nil {
		t.Fatal(err)
	}
}

const testReceivable posting.Kind = "test_receivable"

func TestMain(m *testing.M) {
	posting.DeclareKind(testReceivable, true)
	os.Exit(m.Run())
}

func TestKindsListsEveryKind(t *testing.T) {
	want := []posting.KindDecl{{Kind: posting.SalaryExpense}, {Kind: posting.EmployerInsuranceExpense}, {Kind: posting.SalaryPayable},
		{Kind: posting.InsurancePayable}, {Kind: posting.PitPayable}, {Kind: testReceivable, ByPartner: true}}
	if got := posting.Kinds(); !slices.Equal(got, want) {
		t.Fatalf("got %v", got)
	}
}

type partnerFixture struct {
	ctx     context.Context
	s       *posting.Service
	doc     func() record.Ref
	unit    int64
	partner int64
	db      interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	}
}

func newPartnerFixture(t *testing.T) partnerFixture {
	pool := pgtest.New(t)
	ctx := platform.WithDB(t.Context(), pool)
	db := pgtest.System(t, pool)
	f := partnerFixture{ctx: ctx, s: posting.NewService(posting.Hooks{}), db: db}
	if err := db.QueryRow(ctx, `INSERT INTO iam.org_units (kind, name) VALUES ('company', 'C') RETURNING id`).Scan(&f.unit); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO partner.partners (code, name) VALUES ('DT1', 'P') RETURNING id`).Scan(&f.partner); err != nil {
		t.Fatal(err)
	}
	f.doc = func() record.Ref {
		var id int64
		if err := db.QueryRow(ctx, `INSERT INTO record.documents (doc_type, number, status, version, date, legal_entity_id, org_unit_id)
			VALUES ('test.invoice', 'I', 'posted', 1, '2026-04-30', $1, $1) RETURNING id`, f.unit).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return record.Ref{Type: "test.invoice", ID: id}
	}
	return f
}

func (f partnerFixture) record(ref record.Ref, lines ...posting.Line) error {
	return platform.InTx(f.ctx, func(ctx context.Context) error { return f.s.Record(ctx, ref, lines) })
}

func TestRecordRefusesPartnerMismatch(t *testing.T) {
	f := newPartnerFixture(t)
	p := &f.partner
	for name, l := range map[string]posting.Line{
		"partner kind without partner":   {Kind: testReceivable, OrgUnitID: f.unit, Amount: 1},
		"payroll kind with partner":      {Kind: posting.SalaryExpense, OrgUnitID: f.unit, PartnerID: p, Amount: 1},
		"undeclared kind with a partner": {Kind: "test_other", OrgUnitID: f.unit, PartnerID: p, Amount: 1},
	} {
		if err := f.record(f.doc(), l); !errors.Is(err, posting.ErrPartnerMismatch) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestRecordKeepsPartner(t *testing.T) {
	f := newPartnerFixture(t)
	ref := f.doc()
	if err := f.record(ref, posting.Line{Kind: testReceivable, OrgUnitID: f.unit, PartnerID: &f.partner, Amount: 7},
		posting.Line{Kind: posting.SalaryPayable, OrgUnitID: f.unit, Amount: -7}); err != nil {
		t.Fatal(err)
	}
	var withPartner, partner int64
	if err := f.db.QueryRow(f.ctx, `SELECT count(partner_id), max(partner_id) FROM posting.lines WHERE doc_type = $1 AND doc_id = $2`,
		ref.Type, ref.ID).Scan(&withPartner, &partner); err != nil {
		t.Fatal(err)
	}
	if withPartner != 1 || partner != f.partner {
		t.Fatalf("lines with partner %d, partner %d", withPartner, partner)
	}
}
