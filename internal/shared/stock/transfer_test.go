package stock_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/item"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

const transferType = "stock.transfer"

// warehouse adds an active warehouse of the company.
func (l *ledger) warehouse(t *testing.T, unit int64) int64 {
	t.Helper()
	id, err := l.stock.CreateWarehouse(l.admin, stock.NewWarehouse{WarehouseFields: stock.WarehouseFields{Name: "W", LegalEntityID: unit, Active: true}})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// transferFields move 4 of out from the default warehouse to to.
func (s *stocked) transferFields(date string, to int64) stock.TransferFields {
	return stock.TransferFields{Date: date, FromWarehouseID: s.wh, ToWarehouseID: to,
		Lines: []stock.TransferLine{{ItemID: s.out, Quantity: "4"}}}
}

func (s *stocked) postTransfer(t *testing.T, f stock.TransferFields) (int64, error) {
	t.Helper()
	id, err := s.stock.CreateTransfer(s.admin, stock.NewTransfer{TransferFields: f})
	if err != nil {
		return 0, err
	}
	return id, s.transition(t, transferType, id, record.Posted)
}

// transition moves a stock document to status at its current version.
func (s *stocked) transition(t *testing.T, docType string, id int64, to record.Status) error {
	t.Helper()
	var v int32
	if err := s.core.Pool.QueryRow(s.core.Ctx, `SELECT version FROM record.documents WHERE id = $1`, id).Scan(&v); err != nil {
		t.Fatal(err)
	}
	_, err := s.core.Record.Transition(s.admin, record.Ref{Type: docType, ID: id}, v, to)
	return err
}

func TestTransferRecordContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		s := newStocked(t)
		to := s.warehouse(t, s.unit)
		approver := s.user(t, "approver", "admin")
		return recordtest.Harness{
			Core: s.core, Actor: s.admin, Admin: s.core.Admin, LegalEntity: s.unit, Table: "stock.transfers",
			Get: func(ctx context.Context, r record.Ref) (any, error) { return s.stock.Transfer(ctx, r.ID) },
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				id, err := s.stock.CreateTransfer(ctx, stock.NewTransfer{RequestID: requestID, TransferFields: s.transferFields(date, to)})
				return record.Ref{Type: transferType, ID: id}, err
			},
			Edit: func(ctx context.Context, r record.Ref, version int32, date string) error {
				return s.stock.UpdateTransfer(ctx, r.ID, stock.TransferUpdate{Version: version, TransferFields: s.transferFields(date, to)})
			},
			RequireApproval: func(t *testing.T, a record.Action) {
				if err := s.core.Approval.SaveRule(s.core.Admin, transferType, a, approval.RuleInput{
					Flow:      approval.Flow{{Step: &approval.StepDef{Approver: approval.Approver{Kind: "role", Product: "stock", Role: "admin"}}}},
					MaxLevels: 3, FallbackProduct: "stock", FallbackRole: "admin",
				}); err != nil {
					t.Fatal(err)
				}
			},
			Approve: func(instance int64) error { return s.core.Approval.Approve(approver, instance, 1) },
			Reject:  func(instance int64) error { return s.core.Approval.Reject(approver, instance, 1, "không") },
			// Taking every unit of out from the source leaves the transfer short.
			BreakPosting: func(t *testing.T, _ record.Ref) {
				have, err := decimal.NewFromString(s.onHand(t, s.wh, s.out))
				if err != nil {
					t.Fatal(err)
				}
				if err := s.move(t, stock.Move{LineNo: 1, WarehouseID: s.wh, ItemID: s.out, Quantity: have.Neg()}); err != nil {
					t.Fatal(err)
				}
			},
			// Each effective posting leaves two live moves for its one line.
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

func TestTransferMovesBothSides(t *testing.T) {
	s := newStocked(t)
	to := s.warehouse(t, s.unit)
	id, err := s.postTransfer(t, s.transferFields("2026-11-02", to))
	if err != nil {
		t.Fatal(err)
	}
	if a, b := s.onHand(t, s.wh, s.out), s.onHand(t, to, s.out); a != "996" || b != "4" {
		t.Fatalf("on hand %s / %s", a, b)
	}
	tr, err := s.stock.Transfer(s.admin, id)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Status != "posted" || tr.FromWarehouseID != s.wh || tr.ToWarehouseID != to || len(tr.Lines) != 1 || tr.Lines[0].Quantity != "4" {
		t.Fatalf("read back %+v", tr)
	}
	l, err := s.stock.Transfers(s.admin, stock.TransferFilter{WarehouseID: to, Paging: page})
	if err != nil || l.Total != 1 || l.Items[0].ToWarehouseID != to {
		t.Fatalf("list by destination %+v %v", l, err)
	}
}

// A lot received with an expiry moves without restating it.
func TestTransferLotWithExpiry(t *testing.T) {
	s := newStocked(t)
	to := s.warehouse(t, s.unit)
	lt, sr := s.item(t, item.TrackLot, false), s.item(t, item.TrackSerial, false)
	exp, serial := "2027-01-31", "SN-1"
	if err := s.move(t, stock.Move{LineNo: 1, WarehouseID: s.wh, ItemID: lt, Lot: lot("L1", &exp), Quantity: qty(5)},
		stock.Move{LineNo: 2, WarehouseID: s.wh, ItemID: sr, Serial: &serial, Quantity: qty(1)}); err != nil {
		t.Fatal(err)
	}
	code := "l1"
	if _, err := s.postTransfer(t, stock.TransferFields{Date: "2026-11-02", FromWarehouseID: s.wh, ToWarehouseID: to,
		Lines: []stock.TransferLine{{ItemID: lt, Lot: &code, Quantity: "2"}, {ItemID: sr, Serial: &serial, Quantity: "1"}}}); err != nil {
		t.Fatal(err)
	}
	if a, b, c := s.onHand(t, s.wh, lt), s.onHand(t, to, lt), s.onHand(t, to, sr); a != "3" || b != "2" || c != "1" {
		t.Fatalf("on hand %s / %s / serial %s", a, b, c)
	}
}

func TestTransferShortage(t *testing.T) {
	s := newStocked(t)
	f := s.transferFields("2026-11-02", s.warehouse(t, s.unit))
	f.Lines[0].Quantity = "1001"
	_, err := s.postTransfer(t, f)
	if e := wantCode(t, err, stock.ErrInsufficient); e.Params["line_no"] != 1 {
		t.Fatalf("params %+v", e.Params)
	}
}

func TestTransferSameWarehouse(t *testing.T) {
	s := newStocked(t)
	_, err := s.stock.CreateTransfer(s.admin, stock.NewTransfer{TransferFields: s.transferFields("2026-11-02", s.wh)})
	_ = wantCode(t, err, stock.ErrTransferSameWarehouse)
}

func TestTransferOtherEntity(t *testing.T) {
	s := newStocked(t)
	other, err := s.core.IAM.CreateOrgUnit(s.core.Admin, iam.OrgUnitInput{Kind: "company", Name: "C2"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.stock.CreateTransfer(s.admin, stock.NewTransfer{TransferFields: s.transferFields("2026-11-02", s.warehouse(t, other))})
	_ = wantCode(t, err, stock.ErrWarehouseEntity)
}

func TestTransferQuantityPositive(t *testing.T) {
	s := newStocked(t)
	f := s.transferFields("2026-11-02", s.warehouse(t, s.unit))
	f.Lines[0].Quantity = "-4"
	_, err := s.stock.CreateTransfer(s.admin, stock.NewTransfer{TransferFields: f})
	_ = wantCode(t, err, stock.ErrQuantityInvalid)
}

func TestTransferCancelRefusedWhenGoodsLeft(t *testing.T) {
	s := newStocked(t)
	to := s.warehouse(t, s.unit)
	id, err := s.postTransfer(t, s.transferFields("2026-11-02", to))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.move(t, stock.Move{LineNo: 1, WarehouseID: to, ItemID: s.out, Quantity: qty(-4)}); err != nil {
		t.Fatal(err)
	}
	_ = wantCode(t, s.transition(t, transferType, id, record.Cancelled), stock.ErrInsufficient)
}

func TestTransferCancelReturnsBothSides(t *testing.T) {
	s := newStocked(t)
	to := s.warehouse(t, s.unit)
	id, err := s.postTransfer(t, s.transferFields("2026-11-02", to))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.transition(t, transferType, id, record.Cancelled); err != nil {
		t.Fatal(err)
	}
	if a, b := s.onHand(t, s.wh, s.out), s.onHand(t, to, s.out); a != "1000" || b != "0" {
		t.Fatalf("on hand %s / %s", a, b)
	}
}

func TestTransferNeedsManage(t *testing.T) {
	s := newStocked(t)
	viewer := s.user(t, "viewer", "viewer")
	_, err := s.stock.CreateTransfer(viewer, stock.NewTransfer{TransferFields: s.transferFields("2026-11-02", s.warehouse(t, s.unit))})
	_ = wantCode(t, err, platform.ErrForbidden)
}
