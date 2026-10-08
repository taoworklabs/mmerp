package audit_test

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

func TestRecordChangesEncryptsSensitiveValues(t *testing.T) {
	pool := pgtest.New(t)
	ctx := platform.WithKeyring(platform.WithDB(t.Context(), pool), pgtest.Keyring())
	s := audit.NewService()
	ref := audit.Ref{Type: "hrm.employee", ID: 7}
	nid := "079123456789"
	err := s.RecordChanges(ctx, "hrm.employee_updated", ref, []audit.Change{
		{Field: "full_name", Old: "An", New: "Ân"},
		{Field: "phone", Old: "1", New: "1"},
		{Field: "national_id", Old: nil, New: &nid, Sensitive: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RecordChanges(ctx, "hrm.employee_updated", ref, []audit.Change{{Field: "phone", Old: "1", New: "1"}}); err != nil {
		t.Fatal(err)
	}

	var n int
	var raw []byte
	if err := pool.QueryRow(t.Context(), `SELECT count(*) OVER (), data FROM audit.log WHERE doc_type = 'hrm.employee' AND doc_id = 7`).Scan(&n, &raw); err != nil {
		t.Fatal(err)
	}
	if n != 1 || strings.Contains(string(raw), nid) || strings.Contains(string(raw), "phone") {
		t.Fatalf("%d rows, data = %s", n, raw)
	}
	var data struct {
		Changes map[string]struct {
			Old, New  *string
			Sensitive bool
		}
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	c := data.Changes["national_id"]
	sealed, _ := base64.StdEncoding.DecodeString(*c.New)
	pt, err := platform.Decrypt(ctx, "audit.hrm.employee.national_id", sealed)
	if err != nil || string(pt) != `"079123456789"` || !c.Sensitive || c.Old != nil || *data.Changes["full_name"].New != "Ân" {
		t.Fatalf("data = %s, decrypted %s %v", raw, pt, err)
	}
}
