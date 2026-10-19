package approval_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
	"github.com/taoworklabs/mmerp/migrations"
)

func TestMigrationConvertsRulesAndOpenInstances(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.Empty(t)
	before := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() < "0041" {
			data, err := fs.ReadFile(migrations.FS, e.Name())
			if err != nil {
				t.Fatal(err)
			}
			before[e.Name()] = &fstest.MapFile{Data: data}
		}
	}
	if err := platform.Migrate(ctx, pool, before); err != nil {
		t.Fatal(err)
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	var u, inst int64
	if err := pool.QueryRow(ctx, `INSERT INTO iam.users (login, name, password_hash) VALUES ('u', 'u', 'x') RETURNING id`).Scan(&u); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO approval.rules (doc_type, steps, max_levels, fallback_product, fallback_role) VALUES ('test.doc', $1, 3, 'test', 'hr')`,
		fmt.Sprintf(`[{"approver":{"kind":"user","user_id":%d},"condition":{"field":"days","op":"gt","value":"3"}},{"approver":{"kind":"module"}}]`, u))
	if err := pool.QueryRow(ctx, `INSERT INTO approval.instances (doc_type, doc_id, version, submitted_by, status, current_step, max_levels, fallback_product, fallback_role)
		VALUES ('test.doc', 1, 1, $1, 'open', 2, 3, 'test', 'hr') RETURNING id`, u).Scan(&inst); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO approval.instance_steps (instance_id, position, approver, approvers, decided_by, decision, decided_at)
		VALUES ($1, 1, $2, ARRAY[$3::bigint], $3, 'approved', now()), ($1, 2, '{"kind":"module"}', ARRAY[$3::bigint], NULL, NULL, NULL)`,
		inst, fmt.Sprintf(`{"kind":"user","user_id":%d}`, u), u)

	if err := platform.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatal(err)
	}

	same := func(what string, got []byte, want string) {
		t.Helper()
		var g, w any
		if err := json.Unmarshal(got, &g); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(want), &w); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(g, w) {
			t.Fatalf("%s: got %s, want %s", what, got, want)
		}
	}
	var flow, plan []byte
	if err := pool.QueryRow(ctx, `SELECT flow FROM approval.rules WHERE doc_type = 'test.doc'`).Scan(&flow); err != nil {
		t.Fatal(err)
	}
	same("flow", flow, fmt.Sprintf(`[{"split":{"mode":"one","otherwise":[],"branches":[{"condition":{"field":"days","op":"gt","value":"3"},`+
		`"body":[{"step":{"approver":{"kind":"user","user_ids":[%d]},"approvals":1,"rejections":1}}]}]}},`+
		`{"step":{"approver":{"kind":"module"},"approvals":1,"rejections":1}}]`, u))
	if err := pool.QueryRow(ctx, `SELECT plan FROM approval.instances WHERE id = $1`, inst).Scan(&plan); err != nil {
		t.Fatal(err)
	}
	same("plan", plan, `{"seq":[{"step":1},{"step":2}]}`)

	rows, _ := pool.Query(ctx, `SELECT status FROM approval.instance_steps WHERE instance_id = $1 ORDER BY position`, inst)
	var statuses []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		statuses = append(statuses, s)
	}
	if rows.Err() != nil || !reflect.DeepEqual(statuses, []string{"approved", "open"}) {
		t.Fatalf("statuses %v %v", statuses, rows.Err())
	}
	var votes int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM approval.step_votes WHERE instance_id = $1 AND position = 1 AND user_id = $2 AND decision = 'approved'`, inst, u).Scan(&votes); err != nil {
		t.Fatal(err)
	}
	var all int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM approval.step_votes`).Scan(&all); err != nil || votes != 1 || all != 1 {
		t.Fatalf("votes %d of %d (%v)", votes, all, err)
	}
}
