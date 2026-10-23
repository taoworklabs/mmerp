package sales

import (
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/printing"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/shared/partner"
)

type Deps struct {
	IAM       *iam.Service
	Record    *record.Service
	Audit     *audit.Service
	Setting   *setting.Service
	Printing  *printing.Service
	Numbering *numbering.Service
	Partner   *partner.Service
}
