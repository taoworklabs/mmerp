package printing_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/printing"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

// Posting a printable document freezes its print data in the same transaction: if
// freezing fails, the document stays a draft.
func TestPostingFreezesThePrint(t *testing.T) {
	pool := pgtest.New(t)
	ctx := platform.WithKeyring(platform.WithProducts(platform.WithDB(t.Context(), pool), []string{"test"}), pgtest.Keyring())
	ids := iam.NewService(iam.Deps{Setting: setting.NewService(), Audit: audit.NewService()})
	rec := record.NewService(record.Deps{IAM: ids, Numbering: numbering.NewService(), Audit: audit.NewService()})
	ids.SetTreeHook(rec)
	rec.Register(record.Type{Code: "test.doc", Product: "test", Kind: record.Document, NumberPrefix: "T",
		Can: func(context.Context, int64, record.Action) (bool, error) { return true, nil }})
	broken := errors.New("data broken")
	var fail bool
	prn := printing.NewService(printing.Deps{Record: rec, Audit: audit.NewService(), IAM: ids,
		DataIO: dataio.NewService(dataio.Deps{IAM: ids, Audit: audit.NewService()})})
	prn.Register(printing.Template{Code: "test.doc", DocType: "test.doc", Product: "test",
		Data: func(context.Context, int64) ([]printing.Part, error) {
			if fail {
				return nil, broken
			}
			return []printing.Part{{Key: 7, Data: json.RawMessage(`{"amount":12345678}`)}}, nil
		},
		Layouts: []printing.Layout{func(*printing.Page, json.RawMessage) error { return nil }}})

	admin := platform.WithActor(ctx, must[int64](t)(ids.CreateAdmin(ctx, "admin", "Admin", "long enough")))
	c := must[int64](t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	snapshots := func(id int64) (n int) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM printing.snapshots WHERE doc_id = $1`, id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	fail = true
	d := must[record.Doc](t)(rec.Create(admin, "test.doc", record.Header{Date: "2026-03-10", OrgUnitID: c}))
	if err := rec.Transition(admin, d.Ref, d.Version, record.Posted); !errors.Is(err, broken) {
		t.Fatalf("post with broken data: %v", err)
	}
	if got := must[record.Doc](t)(rec.Get(admin, d.Ref)); got.Status != record.Draft || snapshots(d.ID) != 0 {
		t.Fatalf("after a failed freeze: %s, %d snapshots", got.Status, snapshots(d.ID))
	}

	fail = false
	if err := rec.Transition(admin, d.Ref, d.Version, record.Posted); err != nil {
		t.Fatal(err)
	}
	if snapshots(d.ID) != 1 {
		t.Fatal("no snapshot")
	}
	// Encrypted: the amount is not stored in clear.
	var leaked bool
	if err := pool.QueryRow(ctx, `SELECT position('12345678' IN encode(data, 'escape')) > 0 FROM printing.snapshots WHERE doc_id = $1`, d.ID).Scan(&leaked); err != nil || leaked {
		t.Fatalf("snapshot in clear: %v %v", leaked, err)
	}
}
