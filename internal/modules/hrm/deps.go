package hrm

import (
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

type Deps struct {
	IAM     *iam.Service
	Record  *record.Service
	Audit   *audit.Service
	Setting *setting.Service
	DataIO  *dataio.Service
	Posting *posting.Service
}
