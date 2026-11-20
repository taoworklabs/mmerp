package stock

import (
	"context"
	"sync"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

// Business line kinds of stock value. Amounts are positive and accounting's rules give the
// side; KindInventory is signed: goods in positive, goods out negative. Every document moving
// goods writes KindInventory with its own counterpart kind.
const (
	KindInventory         posting.Kind = "stock_inventory"
	KindAdjustmentGain    posting.Kind = "stock_adjustment_gain"
	KindAdjustmentLoss    posting.Kind = "stock_adjustment_loss"
	KindStocktakeSurplus  posting.Kind = "stocktake_surplus"
	KindStocktakeShortage posting.Kind = "stocktake_shortage"
)

// declareKinds runs once per process: tests build many services.
var declareKinds = sync.OnceFunc(func() {
	for _, k := range []posting.Kind{KindInventory, KindAdjustmentGain, KindAdjustmentLoss, KindStocktakeSurplus, KindStocktakeShortage} {
		posting.DeclareKind(k, false)
	}
})

// postDifferences writes the business lines of moves worth vals: the inventory change, and
// what came in to gain, what went out to loss.
func (s *Service) postDifferences(ctx context.Context, d record.Doc, vals []int64, gain, loss posting.Kind) error {
	var in, out int64
	for _, v := range vals {
		if v > 0 {
			in += v
		} else {
			out -= v
		}
	}
	var lines []posting.Line
	for _, l := range []posting.Line{
		{Kind: KindInventory, OrgUnitID: d.OrgUnitID, Amount: in - out},
		{Kind: gain, OrgUnitID: d.OrgUnitID, Amount: in},
		{Kind: loss, OrgUnitID: d.OrgUnitID, Amount: out},
	} {
		if l.Amount != 0 {
			lines = append(lines, l)
		}
	}
	return s.d.Posting.Record(ctx, d.Ref, lines)
}

// voidPosted takes the goods back, then voids the business lines.
func (s *Service) voidPosted(ctx context.Context, ref record.Ref) error {
	if err := s.Void(ctx, ref); err != nil {
		return err
	}
	return s.d.Posting.Void(ctx, ref)
}
