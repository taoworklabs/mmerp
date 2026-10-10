package printing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	pdfread "github.com/ledongthuc/pdf"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
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

type harness struct {
	pool  platform.Conn
	rec   *record.Service
	prn   *Service
	admin context.Context
	// a draft of the printable type test.doc
	doc record.Doc
}

// setup prints the document type test.doc, which anyone may do anything with, from data.
func setup(t *testing.T, data func(context.Context, int64) ([]Part, error), layout Layout) harness {
	t.Helper()
	pool := pgtest.New(t)
	ctx := platform.WithKeyring(platform.WithProducts(platform.WithDB(t.Context(), pool), []string{"test"}), pgtest.Keyring())
	ids := iam.NewService(iam.Deps{Setting: setting.NewService(), Audit: audit.NewService()})
	rec := record.NewService(record.Deps{IAM: ids, Numbering: numbering.NewService(), Audit: audit.NewService()})
	ids.SetTreeHook(rec)
	rec.Register(record.Type{Code: "test.doc", Product: "test", Kind: record.Document, NumberPrefix: "T",
		Can: func(context.Context, int64, record.Action) (bool, error) { return true, nil }})
	prn := NewService(Deps{Record: rec, Audit: audit.NewService(), IAM: ids,
		DataIO: dataio.NewService(dataio.Deps{IAM: ids, Audit: audit.NewService()})})
	prn.Register(Template{Code: "test.doc", DocType: "test.doc", Data: data, Layouts: []Layout{layout}})
	admin := platform.WithActor(ctx, must[int64](t)(ids.CreateAdmin(ctx, "admin", "Admin", "long enough")))
	c := must[int64](t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	d := must[record.Doc](t)(rec.Create(admin, "test.doc", record.Header{Date: "2026-03-10", OrgUnitID: c}))
	return harness{pool: pool, rec: rec, prn: prn, admin: admin, doc: d}
}

// Posting a printable document freezes its print data in the same transaction: if
// freezing fails, the document stays a draft.
func TestPostingFreezesThePrint(t *testing.T) {
	broken := errors.New("data broken")
	fail := true
	h := setup(t, func(context.Context, int64) ([]Part, error) {
		if fail {
			return nil, broken
		}
		return []Part{{Key: 7, Data: json.RawMessage(`{"amount":12345678}`)}}, nil
	}, func(*Page, json.RawMessage) error { return nil })
	snapshots := func() (n int) {
		t.Helper()
		if err := h.pool.QueryRow(h.admin, `SELECT count(*) FROM printing.snapshots WHERE doc_id = $1`, h.doc.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	if err := h.rec.Transition(h.admin, h.doc.Ref, h.doc.Version, record.Posted); !errors.Is(err, broken) {
		t.Fatalf("post with broken data: %v", err)
	}
	if got := must[record.Doc](t)(h.rec.Get(h.admin, h.doc.Ref)); got.Status != record.Draft || snapshots() != 0 {
		t.Fatalf("after a failed freeze: %s, %d snapshots", got.Status, snapshots())
	}

	fail = false
	if err := h.rec.Transition(h.admin, h.doc.Ref, h.doc.Version, record.Posted); err != nil {
		t.Fatal(err)
	}
	if snapshots() != 1 {
		t.Fatal("no snapshot")
	}
	// Encrypted: the amount is not stored in clear.
	var leaked bool
	if err := h.pool.QueryRow(h.admin, `SELECT position('12345678' IN encode(data, 'escape')) > 0 FROM printing.snapshots WHERE doc_id = $1`, h.doc.ID).Scan(&leaked); err != nil || leaked {
		t.Fatalf("snapshot in clear: %v %v", leaked, err)
	}
}

// A payroll's worth of payslips prints into one PDF, a page each.
func TestPrintManyParts(t *testing.T) {
	const n = 3000
	h := setup(t, func(context.Context, int64) ([]Part, error) {
		parts := make([]Part, n)
		for i := range parts {
			parts[i] = Part{Key: int64(i + 1), Data: json.RawMessage(fmt.Sprintf(`{"name":"Nhân viên %d","net":%d}`, i+1, 15_000_000+i))}
		}
		return parts, nil
	}, func(p *Page, raw json.RawMessage) error {
		var d struct {
			Name string
			Net  int64
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return err
		}
		p.Title("PHIẾU LƯƠNG")
		p.Field("Họ và tên", d.Name)
		p.Table([]Column{{Title: "Khoản", Share: 0.7}, {Title: "Số tiền", Share: 0.3, Right: true}}, [][]string{{"Thực lĩnh", p.Money(d.Net)}}, true)
		return nil
	})
	f, err := h.prn.render(h.admin, h.prn.templates["test.doc"], json.RawMessage(fmt.Sprintf(`{"id":%d}`, h.doc.ID)))
	if err != nil {
		t.Fatal(err)
	}
	r, err := pdfread.NewReader(bytes.NewReader(f.Body), int64(len(f.Body)))
	if err != nil || r.NumPage() != n {
		t.Fatalf("pages: %v %v", r.NumPage(), err)
	}
}
