package stock

import (
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
)

type Deps struct {
	IAM       *iam.Service
	Audit     *audit.Service
	Numbering *numbering.Service
}
