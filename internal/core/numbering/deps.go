package numbering

import (
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
)

// Deps are needed only to manage code series; issuing numbers and codes needs none.
type Deps struct {
	IAM   *iam.Service
	Audit *audit.Service
}
