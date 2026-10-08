package posting

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/core/record"
)

// Hooks are reactions to posting lines changing; a nil hook does nothing.
type Hooks struct {
	// OnChanged runs in the transaction that recorded or voided ref's lines.
	OnChanged func(ctx context.Context, ref record.Ref) error
}
