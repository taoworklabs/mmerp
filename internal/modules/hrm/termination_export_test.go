package hrm

import (
	"context"

	"github.com/taoworklabs/mmerp/internal/core/printing"
)

func (s *Service) TerminationPrint(ctx context.Context, id int64) ([]printing.Part, error) {
	return s.terminationPrint(ctx, id)
}
