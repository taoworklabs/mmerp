package stock

import (
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/printing"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

type Deps struct {
	IAM       *iam.Service
	Audit     *audit.Service
	Numbering *numbering.Service
	Record    *record.Service
	Printing  *printing.Service
	Posting   *posting.Service
}
