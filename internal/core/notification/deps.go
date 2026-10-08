package notification

import (
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
)

type Deps struct {
	Record  *record.Service
	IAM     *iam.Service
	Audit   *audit.Service
	Setting *setting.Service
}
