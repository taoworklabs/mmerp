package stock_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/printing"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/item"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

const adjustmentType = "stock.adjustment"

// stocked is a ledger with an item already on hand, so adjustments can take some out.
type stocked struct {
	*ledger
	in, out int64 // quantity-tracked items: out starts with 1000 on hand
}

func newStocked(t *testing.T) *stocked {
	t.Helper()
	s := &stocked{ledger: newLedger(t)}
	s.in, s.out = s.item(t, item.TrackQuantity, false), s.item(t, item.TrackQuantity, false)
	if err := s.move(t, stock.Move{LineNo: 1, WarehouseID: s.wh, ItemID: s.out, Quantity: qty(1000)}); err != nil {
		t.Fatal(err)
	}
	return s
}

// fields bring 5 of in and take 1 of out.
func (s *stocked) fields(date string) stock.AdjustmentFields {
	return stock.AdjustmentFields{Date: date, WarehouseID: s.wh, Reason: "Kiểm kê",
		Lines: []stock.AdjustmentLine{{ItemID: s.in, Quantity: "5"}, {ItemID: s.out, Quantity: "-1"}}}
}

func TestAdjustmentRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		s := newStocked(t)
		approver := s.user(t, "approver", "admin")
		return recordtest.Harness{
			Core: s.core, Actor: s.admin, Admin: s.core.Admin, LegalEntity: s.unit, Table: "stock.adjustments",
			Get: func(ctx context.Context, r record.Ref) (any, error) { return s.stock.Adjustment(ctx, r.ID) },
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				id, err := s.stock.CreateAdjustment(ctx, stock.NewAdjustment{RequestID: requestID, AdjustmentFields: s.fields(date)})
				return record.Ref{Type: adjustmentType, ID: id}, err
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return s.stock.UpdateAdjustment(ctx, r.ID, stock.AdjustmentUpdate{Version: version, AdjustmentFields: s.fields(date)})
			},
			RequireApproval: func(t *testing.T, a record.Action) {
				if err := s.core.Approval.SaveRule(s.core.Admin, adjustmentType, a, approval.RuleInput{
					Flow:      approval.Flow{{Step: &approval.StepDef{Approver: approval.Approver{Kind: "role", Product: "stock", Role: "admin"}}}},
					MaxLevels: 3, FallbackProduct: "stock", FallbackRole: "admin",
				}); err != nil {
					t.Fatal(err)
				}
			},
			Approve: func(instance int64) error { return s.core.Approval.Approve(approver, instance, 1) },
			Reject:  func(instance int64) error { return s.core.Approval.Reject(approver, instance, 1, "không") },
			// Taking every unit of out elsewhere leaves the adjustment short.
			BreakPosting: func(t *testing.T, _ record.Ref) {
				have, err := decimal.NewFromString(s.onHand(t, s.wh, s.out))
				if err != nil {
					t.Fatal(err)
				}
				if err := s.move(t, stock.Move{LineNo: 1, WarehouseID: s.wh, ItemID: s.out, Quantity: have.Neg()}); err != nil {
					t.Fatal(err)
				}
			},
			// Each effective posting leaves one live move per line.
			Effects: func(t *testing.T, r record.Ref) int {
				var n int
				if err := s.core.Pool.QueryRow(s.core.Ctx, `SELECT count(*) FROM stock.moves m WHERE m.doc_type = $1 AND m.doc_id = $2
					AND m.reverses IS NULL AND NOT EXISTS (SELECT 1 FROM stock.moves r WHERE r.reverses = m.id)`, r.Type, r.ID).Scan(&n); err != nil {
					t.Fatal(err)
				}
				return n / 2
			},
		}
	})
}

func TestAdjustmentCreatesLotsInline(t *testing.T) {
	s := newStocked(t)
	lt := s.item(t, item.TrackLot, false)
	exp := "2027-01-31"
	id, err := s.stock.CreateAdjustment(s.admin, stock.NewAdjustment{AdjustmentFields: stock.AdjustmentFields{Date: "2026-11-02", WarehouseID: s.wh, Reason: "Nhập tồn đầu",
		Lines: []stock.AdjustmentLine{{ItemID: lt, Lot: lot("L9", &exp), Quantity: "12.5"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := s.onHand(t, s.wh, lt); got != "0" {
		t.Fatalf("a draft moved stock: %s", got)
	}
	a, err := s.stock.Adjustment(s.admin, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.core.Record.Transition(s.admin, record.Ref{Type: adjustmentType, ID: id}, a.Version, record.Posted); err != nil {
		t.Fatal(err)
	}
	if got := s.onHand(t, s.wh, lt); got != "12.5" {
		t.Fatalf("on hand %s", got)
	}
	var code, expiry string
	if err := s.core.Pool.QueryRow(s.core.Ctx, `SELECT code, expiry::text FROM stock.lots WHERE item_id = $1`, lt).Scan(&code, &expiry); err != nil || code != "L9" || expiry != exp {
		t.Fatalf("lot %s %s %v", code, expiry, err)
	}
	a, err = s.stock.Adjustment(s.admin, id)
	if err != nil {
		t.Fatal(err)
	}
	if l := a.Lines[0]; a.Status != "posted" || l.Lot == nil || l.Lot.Code != "L9" || *l.Lot.Expiry != exp || l.Quantity != "12.5" {
		t.Fatalf("read back %+v", a)
	}
}

func TestAdjustmentNeedsManage(t *testing.T) {
	s := newStocked(t)
	viewer := s.user(t, "viewer", "viewer")
	_, err := s.stock.CreateAdjustment(viewer, stock.NewAdjustment{AdjustmentFields: s.fields("2026-11-02")})
	_ = wantCode(t, err, platform.ErrForbidden)
	l, err := s.stock.Adjustments(viewer, stock.AdjustmentFilter{Paging: page})
	if err != nil || l.Total != 0 {
		t.Fatalf("list %+v %v", l, err)
	}
	if _, err := s.stock.CreateAdjustment(s.admin, stock.NewAdjustment{AdjustmentFields: s.fields("2026-11-02")}); err != nil {
		t.Fatal(err)
	}
	if l, err = s.stock.Adjustments(viewer, stock.AdjustmentFilter{Paging: page}); err != nil || l.Total != 1 || l.Items[0].WarehouseID != s.wh {
		t.Fatalf("list %+v %v", l, err)
	}
}

// The print's text blocks are saved with no product enabled: stock is not a product.
func TestAdjustmentPrintBlocksSaved(t *testing.T) {
	s := newStocked(t)
	if err := s.core.Printing.SaveBlocks(s.core.Admin, adjustmentType, []printing.BlockText{{Key: "footer", Vi: "Thủ kho", En: "Storekeeper"}}); err != nil {
		t.Fatal(err)
	}
}

// Saving checks the quantity but leaves tracking to posting.
func TestAdjustmentQuantityChecked(t *testing.T) {
	s := newStocked(t)
	f := s.fields("2026-11-02")
	f.Lines[1].Quantity = "0"
	_, err := s.stock.CreateAdjustment(s.admin, stock.NewAdjustment{AdjustmentFields: f})
	if e := wantCode(t, err, stock.ErrQuantityInvalid); e.Params["line_no"] != 2 {
		t.Fatalf("params %+v", e.Params)
	}
}
