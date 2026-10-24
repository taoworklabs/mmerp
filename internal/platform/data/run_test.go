package data_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// runRow is the scratch table's columns, scanned by pgx.RowToStructByName: no
// "db" tags needed since field names normalize to the same thing as the columns.
type runRow struct {
	ID        int64
	OrgUnitID int64
	UserID    int64
	Code      pgtype.Text
	StartDate pgtype.Date
}

func runFixtureMethod(struct{}, context.Context) ([]runRow, error) { return nil, nil }

func runList() data.List[runRow] {
	base := data.NewBase("test.Run",
		`SELECT id, org_unit_id, user_id, code, start_date FROM data_run_test_t`,
		[]string{"id", "org_unit_id", "user_id", "code", "start_date"},
		runFixtureMethod)
	return data.List[runRow]{
		Name:       "test.Run",
		Base:       base,
		Scope:      data.ByColumn("org_unit_id"),
		Visibility: func(v data.Viewer) data.Cond { return data.Eq("user_id", v.Actor) },
		Default:    []data.Order{data.Asc("id")},
		Tiebreak:   data.Asc("id"),
	}
}

// seedRunFixture builds a 30-row table across 3 org units (10 rows each) and
// 2 users (15 rows each, by id parity), through a SystemPool: the AppPool pool
// may be read-only outside a transaction.
func seedRunFixture(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	system := pgtest.System(t, pool)
	ctx := t.Context()
	if _, err := system.Exec(ctx, `CREATE TABLE data_run_test_t (
		id bigint PRIMARY KEY,
		org_unit_id bigint NOT NULL,
		user_id bigint NOT NULL,
		code text NOT NULL,
		start_date date NOT NULL
	)`); err != nil {
		t.Fatalf("seedRunFixture: create table: %v", err)
	}
	for id := int64(1); id <= 30; id++ {
		orgUnit := (id-1)%3 + 1
		user := int64(100)
		if id%2 == 0 {
			user = 200
		}
		code := fmt.Sprintf("C%02d", id)
		start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(id))
		if _, err := system.Exec(ctx,
			`INSERT INTO data_run_test_t (id, org_unit_id, user_id, code, start_date) VALUES ($1, $2, $3, $4, $5)`,
			id, orgUnit, user, code, start,
		); err != nil {
			t.Fatalf("seedRunFixture: insert %d: %v", id, err)
		}
	}
}

func runPaging(page, size int) platform.Paging { return platform.Paging{Page: page, PageSize: size} }

func TestRunPage(t *testing.T) {
	pool := pgtest.New(t)
	seedRunFixture(t, pool)
	ctx := platform.WithDB(t.Context(), pool)

	got, err := runList().Run(ctx, data.Query{Scope: data.GrantUnits([]int64{1}), Paging: runPaging(1, 20)})
	if err != nil {
		t.Fatal(err)
	}

	wantIDs := []int64{1, 4, 7, 10, 13, 16, 19, 22, 25, 28}
	if len(got.Items) != len(wantIDs) {
		t.Fatalf("len(Items) = %d, want %d (%+v)", len(got.Items), len(wantIDs), got.Items)
	}
	for i, id := range wantIDs {
		if got.Items[i].ID != id {
			t.Fatalf("Items[%d].ID = %d, want %d", i, got.Items[i].ID, id)
		}
	}
	if got.Total != int64(len(wantIDs)) {
		t.Fatalf("Total = %d, want %d", got.Total, len(wantIDs))
	}
}

// TestRunPageExactlyFull covers the boundary the short-page shortcut must not
// swallow: a first page exactly as long as the limit is not "shorter than the
// limit", so the count still has to run (as its own query, not batched, since
// the page already went out alone).
func TestRunPageExactlyFull(t *testing.T) {
	c := &logCapture{}
	pool := tracedPool(t, platform.NewQueryTracer(slog.New(c), 0))
	seedRunFixture(t, pool)
	ctx := platform.WithDB(t.Context(), pool)

	l := runList()
	got, err := l.Run(ctx, data.Query{Scope: data.GrantUnits([]int64{1}), Paging: runPaging(1, 10)})
	if err != nil {
		t.Fatal(err)
	}

	wantIDs := []int64{1, 4, 7, 10, 13, 16, 19, 22, 25, 28}
	if len(got.Items) != len(wantIDs) {
		t.Fatalf("len(Items) = %d, want %d (%+v)", len(got.Items), len(wantIDs), got.Items)
	}
	if got.Total != int64(len(wantIDs)) {
		t.Fatalf("Total = %d, want %d (the true count)", got.Total, len(wantIDs))
	}

	var n int
	for _, r := range c.all() {
		if r.attrs["name"] == l.Name {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("logged %d queries named %q, want 2 (page and count, the count must still run)", n, l.Name)
	}
}

func TestRunPagePastEnd(t *testing.T) {
	pool := pgtest.New(t)
	seedRunFixture(t, pool)
	ctx := platform.WithDB(t.Context(), pool)

	got, err := runList().Run(ctx, data.Query{Scope: data.GrantAll(), Paging: runPaging(99, 20)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Items == nil {
		t.Fatal("Items must be a non-nil slice")
	}
	if len(got.Items) != 0 {
		t.Fatalf("Items = %+v, want empty", got.Items)
	}
	if got.Total != 30 {
		t.Fatalf("Total = %d, want 30", got.Total)
	}
}

func TestRunSkipsCountOnShortFirstPage(t *testing.T) {
	c := &logCapture{}
	pool := tracedPool(t, platform.NewQueryTracer(slog.New(c), 0))
	seedRunFixture(t, pool)
	ctx := platform.WithDB(t.Context(), pool)

	l := runList()
	got, err := l.Run(ctx, data.Query{Scope: data.GrantUnits([]int64{1}), Paging: runPaging(1, 20)})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 10 {
		t.Fatalf("len(Items) = %d, want 10", len(got.Items))
	}

	var n int
	for _, r := range c.all() {
		if r.attrs["name"] == l.Name {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("logged %d queries named %q, want 1 (count must be skipped)", n, l.Name)
	}
}

// TestRunBatchLogsBothQueries guards the pgx.BatchTracer added for Run: a page
// past 1 always sends page+count together, and both must still be logged, not
// silently dropped because pgx traces a batch differently from a plain query.
func TestRunBatchLogsBothQueries(t *testing.T) {
	c := &logCapture{}
	pool := tracedPool(t, platform.NewQueryTracer(slog.New(c), 0))
	seedRunFixture(t, pool)
	ctx := platform.WithDB(t.Context(), pool)

	l := runList()
	got, err := l.Run(ctx, data.Query{Scope: data.GrantAll(), Paging: runPaging(2, 20)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 30 {
		t.Fatalf("Total = %d, want 30", got.Total)
	}

	var n int
	for _, r := range c.all() {
		if r.attrs["name"] == l.Name {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("logged %d queries named %q, want 2 (page and count, batched)", n, l.Name)
	}
}

func TestRunZeroScopeNoViewer(t *testing.T) {
	c := &logCapture{}
	pool := tracedPool(t, platform.NewQueryTracer(slog.New(c), 0))
	seedRunFixture(t, pool)
	ctx := platform.WithDB(t.Context(), pool)

	l := runList()
	got, err := l.Run(ctx, data.Query{Paging: runPaging(1, 20)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Items == nil {
		t.Fatal("Items must be a non-nil slice")
	}
	if len(got.Items) != 0 {
		t.Fatalf("Items = %+v, want empty", got.Items)
	}
	if got.Total != 0 {
		t.Fatalf("Total = %d, want 0", got.Total)
	}

	for _, r := range c.all() {
		if r.attrs["name"] == l.Name {
			t.Fatalf("logged a query named %q, want none", l.Name)
		}
	}
}

func TestRunVisibilityOnly(t *testing.T) {
	pool := pgtest.New(t)
	seedRunFixture(t, pool)
	ctx := platform.WithActor(platform.WithDB(t.Context(), pool), 100)

	got, err := runList().Run(ctx, data.Query{Paging: runPaging(1, 50)})
	if err != nil {
		t.Fatal(err)
	}

	wantIDs := []int64{1, 3, 5, 7, 9, 11, 13, 15, 17, 19, 21, 23, 25, 27, 29}
	if len(got.Items) != len(wantIDs) {
		t.Fatalf("len(Items) = %d, want %d (%+v)", len(got.Items), len(wantIDs), got.Items)
	}
	for i, id := range wantIDs {
		if got.Items[i].ID != id {
			t.Fatalf("Items[%d].ID = %d, want %d", i, got.Items[i].ID, id)
		}
		if got.Items[i].UserID != 100 {
			t.Fatalf("Items[%d].UserID = %d, want 100", i, got.Items[i].UserID)
		}
	}
	if got.Total != int64(len(wantIDs)) {
		t.Fatalf("Total = %d, want %d", got.Total, len(wantIDs))
	}
}

func TestRunScansIntoBaseRow(t *testing.T) {
	pool := pgtest.New(t)
	seedRunFixture(t, pool)
	ctx := platform.WithDB(t.Context(), pool)

	got, err := runList().Run(ctx, data.Query{Scope: data.GrantAll(), Paging: runPaging(1, 50)})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) == 0 {
		t.Fatal("Items is empty")
	}
	row := got.Items[0]
	wantStart := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if row.ID != 1 || row.OrgUnitID != 1 || row.UserID != 100 {
		t.Fatalf("row = %+v, want {ID:1 OrgUnitID:1 UserID:100 ...}", row)
	}
	if !row.Code.Valid || row.Code.String != "C01" {
		t.Fatalf("row.Code = %+v, want C01", row.Code)
	}
	if !row.StartDate.Valid || !row.StartDate.Time.Equal(wantStart) {
		t.Fatalf("row.StartDate = %+v, want %v", row.StartDate, wantStart)
	}
}
