package stock_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/item"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

const stocktakeType = "stock.stocktake"

// stocktakeFields count out at 990 (1000 on hand).
func (s *stocked) stocktakeFields(date string) stock.StocktakeFields {
	return stock.StocktakeFields{Date: date, WarehouseID: s.wh, Lines: []stock.StocktakeLine{{ItemID: s.out, Counted: "990"}}}
}

func TestStocktakeRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		s := newStocked(t)
		approver := s.user(t, "approver", "admin")
		return recordtest.Harness{
			Core: s.core, Actor: s.admin, Admin: s.core.Admin, LegalEntity: s.unit, Table: "stock.stocktakes",
			Get: func(ctx context.Context, r record.Ref) (any, error) { return s.stock.Stocktake(ctx, r.ID) },
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				id, err := s.stock.CreateStocktake(ctx, stock.NewStocktake{RequestID: requestID, StocktakeFields: s.stocktakeFields(date)})
				return record.Ref{Type: stocktakeType, ID: id}, err
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return s.stock.UpdateStocktake(ctx, r.ID, stock.StocktakeUpdate{Version: version, StocktakeFields: s.stocktakeFields(date)})
			},
			RequireApproval: func(t *testing.T, a record.Action) {
				if err := s.core.Approval.SaveRule(s.core.Admin, stocktakeType, a, approval.RuleInput{
					Flow:      approval.Flow{{Step: &approval.StepDef{Approver: approval.Approver{Kind: "role", Product: "stock", Role: "admin"}}}},
					MaxLevels: 3, FallbackProduct: "stock", FallbackRole: "admin",
				}); err != nil {
					t.Fatal(err)
				}
			},
			Approve: func(instance int64) error { return s.core.Approval.Approve(approver, instance, 1) },
			Reject:  func(instance int64) error { return s.core.Approval.Reject(approver, instance, 1, "không") },
			// A count never goes short, but deactivating its warehouse makes the ledger refuse it.
			BreakPosting: func(t *testing.T, _ record.Ref) {
				if err := s.stock.UpdateWarehouse(s.admin, s.wh, stock.WarehouseUpdate{Name: "Kho", Active: false}); err != nil {
					t.Fatal(err)
				}
			},
			// Each effective posting leaves one live move for its one line.
			Effects: func(t *testing.T, r record.Ref) int {
				var n int
				if err := s.core.Pool.QueryRow(s.core.Ctx, `SELECT count(*) FROM stock.moves m WHERE m.doc_type = $1 AND m.doc_id = $2
					AND m.reverses IS NULL AND NOT EXISTS (SELECT 1 FROM stock.moves r WHERE r.reverses = m.id)`, r.Type, r.ID).Scan(&n); err != nil {
					t.Fatal(err)
				}
				return n
			},
		}
	})
}

func (s *stocked) postStocktake(t *testing.T, id int64) error {
	t.Helper()
	return s.transition(t, stocktakeType, id, record.Posted)
}

func TestStocktakePostsDifference(t *testing.T) {
	s := newStocked(t)
	if err := s.move(t, stock.Move{LineNo: 1, WarehouseID: s.wh, ItemID: s.in, Quantity: qty(5)}); err != nil {
		t.Fatal(err)
	}
	untouched := s.item(t, item.TrackQuantity, false)
	if err := s.move(t, stock.Move{LineNo: 1, WarehouseID: s.wh, ItemID: untouched, Quantity: qty(7)}); err != nil {
		t.Fatal(err)
	}
	f := s.stocktakeFields("2026-11-02")
	f.Lines = append(f.Lines, stock.StocktakeLine{ItemID: s.in, Counted: "8"})
	id, err := s.stock.CreateStocktake(s.admin, stock.NewStocktake{StocktakeFields: f})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := s.stock.Stocktake(s.admin, id)
	if err != nil {
		t.Fatal(err)
	}
	if l := draft.Lines[0]; l.OnHand == nil || *l.OnHand != "1000" || l.Book != nil {
		t.Fatalf("draft line %+v", l)
	}
	if err := s.postStocktake(t, id); err != nil {
		t.Fatal(err)
	}
	for it, want := range map[int64]string{s.out: "990", s.in: "8", untouched: "7"} {
		if got := s.onHand(t, s.wh, it); got != want {
			t.Fatalf("item %d on hand %s, want %s", it, got, want)
		}
	}
	st, err := s.stock.Stocktake(s.admin, id)
	if err != nil {
		t.Fatal(err)
	}
	a, b := st.Lines[0], st.Lines[1]
	if a.OnHand != nil || *a.Book != "1000" || *a.Difference != "-10" || *b.Book != "5" || *b.Difference != "3" {
		t.Fatalf("posted lines %+v %+v", a, b)
	}
	// Cancelling returns on hand and keeps what was found.
	if err := s.transition(t, stocktakeType, id, record.Cancelled); err != nil {
		t.Fatal(err)
	}
	if got := s.onHand(t, s.wh, s.out); got != "1000" {
		t.Fatalf("after cancel %s", got)
	}
	if st, _ = s.stock.Stocktake(s.admin, id); st.Lines[0].Book == nil {
		t.Fatalf("cancelled lines lost their book quantity")
	}
}

// What moves between saving and posting counts: the difference is taken at posting.
func TestStocktakeDifferenceAtPosting(t *testing.T) {
	s := newStocked(t)
	id, err := s.stock.CreateStocktake(s.admin, stock.NewStocktake{StocktakeFields: s.stocktakeFields("2026-11-02")})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.move(t, stock.Move{LineNo: 1, WarehouseID: s.wh, ItemID: s.out, Quantity: qty(-4)}); err != nil {
		t.Fatal(err)
	}
	if err := s.postStocktake(t, id); err != nil {
		t.Fatal(err)
	}
	st, err := s.stock.Stocktake(s.admin, id)
	if err != nil {
		t.Fatal(err)
	}
	if l := st.Lines[0]; *l.Book != "996" || *l.Difference != "-6" || s.onHand(t, s.wh, s.out) != "990" {
		t.Fatalf("line %+v on hand %s", l, s.onHand(t, s.wh, s.out))
	}
}

func TestStocktakeDuplicateLine(t *testing.T) {
	s := newStocked(t)
	lt := s.item(t, item.TrackLot, false)
	f := stock.StocktakeFields{Date: "2026-11-02", WarehouseID: s.wh,
		Lines: []stock.StocktakeLine{{ItemID: lt, Lot: lot(" l1 ", nil), Counted: "1"}, {ItemID: lt, Lot: lot("L1", nil), Counted: "2"}}}
	_, err := s.stock.CreateStocktake(s.admin, stock.NewStocktake{StocktakeFields: f})
	if e := wantCode(t, err, stock.ErrCountDuplicate); e.Params["line_no"] != 2 {
		t.Fatalf("params %+v", e.Params)
	}
}

func TestStocktakeCountedNotNegative(t *testing.T) {
	s := newStocked(t)
	f := s.stocktakeFields("2026-11-02")
	f.Lines[0].Counted = "-1"
	_, err := s.stock.CreateStocktake(s.admin, stock.NewStocktake{StocktakeFields: f})
	_ = wantCode(t, err, stock.ErrQuantityInvalid)
}

func TestStocktakeNeedsManage(t *testing.T) {
	s := newStocked(t)
	viewer := s.user(t, "viewer", "viewer")
	_, err := s.stock.CreateStocktake(viewer, stock.NewStocktake{StocktakeFields: s.stocktakeFields("2026-11-02")})
	_ = wantCode(t, err, platform.ErrForbidden)
}

func TestOnHandLines(t *testing.T) {
	s := newStocked(t)
	lt := s.item(t, item.TrackLot, false)
	if err := s.move(t, stock.Move{LineNo: 1, WarehouseID: s.wh, ItemID: lt, Lot: lot("L1", nil), Quantity: decimal.RequireFromString("2.5")}); err != nil {
		t.Fatal(err)
	}
	viewer := s.user(t, "viewer", "viewer")
	l, err := s.stock.OnHandLines(viewer, s.wh, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Items) != 2 {
		t.Fatalf("lines %+v", l.Items)
	}
	var lotLine stock.OnHandLine
	for _, x := range l.Items {
		if x.ItemID == lt {
			lotLine = x
		}
	}
	if lotLine.Lot == nil || lotLine.Lot.Code != "L1" || lotLine.Counted != "2.5" || lotLine.ItemCode == "" || lotLine.Unit != "cái" {
		t.Fatalf("lot line %+v", lotLine)
	}
	if one, err := s.stock.OnHandLines(viewer, s.wh, lt); err != nil || len(one.Items) != 1 {
		t.Fatalf("one item %+v %v", one, err)
	}
}

func TestStocktakeShapeCheckedOnSave(t *testing.T) {
	s := newStocked(t)
	lt := s.item(t, item.TrackLot, false)
	f := s.stocktakeFields("2026-11-02")
	f.Lines = append(f.Lines, stock.StocktakeLine{ItemID: lt, Counted: "0"})
	_, err := s.stock.CreateStocktake(s.admin, stock.NewStocktake{StocktakeFields: f})
	if e := wantCode(t, err, stock.ErrLotRequired); e.Params["line_no"] != 2 {
		t.Fatalf("params %+v", e.Params)
	}
}

// Loading on hand says when the warehouse holds more rows than a stocktake takes.
func TestOnHandLinesTruncated(t *testing.T) {
	s := newStocked(t)
	l, err := s.stock.OnHandLines(s.admin, s.wh, 0)
	if err != nil || l.Truncated {
		t.Fatalf("small warehouse %+v %v", l.Truncated, err)
	}
	sr := s.item(t, item.TrackSerial, false)
	moves := make([]stock.Move, 2000)
	for i := range moves {
		code := fmt.Sprintf("SN%04d", i)
		moves[i] = stock.Move{LineNo: i + 1, WarehouseID: s.wh, ItemID: sr, Serial: &code, Quantity: qty(1)}
	}
	if err := s.move(t, moves...); err != nil {
		t.Fatal(err)
	}
	if l, err = s.stock.OnHandLines(s.admin, s.wh, 0); err != nil || !l.Truncated || len(l.Items) != 2000 {
		t.Fatalf("large warehouse %d %v %v", len(l.Items), l.Truncated, err)
	}
}

func TestStocktakePostsValues(t *testing.T) {
	s := newStocked(t)
	s.postAdjustment(t, stock.AdjustmentLine{ItemID: s.in, Quantity: "10", UnitCost: new(int64(100))})
	id, err := s.stock.CreateStocktake(s.admin, stock.NewStocktake{StocktakeFields: stock.StocktakeFields{Date: "2026-11-03", WarehouseID: s.wh,
		Lines: []stock.StocktakeLine{{ItemID: s.in, Counted: "7"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.postStocktake(t, id); err != nil {
		t.Fatal(err)
	}
	ref := record.Ref{Type: "stock.stocktake", ID: id}
	if got := s.lines(t, ref); len(got) != 2 || got["stock_inventory"] != -300 || got["stocktake_shortage"] != 300 {
		t.Fatalf("lines %v", got)
	}
	if err := s.transition(t, ref.Type, id, record.Cancelled); err != nil {
		t.Fatal(err)
	}
	if got := s.lines(t, ref); len(got) != 0 {
		t.Fatalf("cancelled %v", got)
	}
}
