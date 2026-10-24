package partner

import (
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/record"
)

type Deps struct {
	IAM       *iam.Service
	Record    *record.Service
	Audit     *audit.Service
	Numbering *numbering.Service
}
