package printing

import (
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
)

type Deps struct {
	Record *record.Service
	Audit  *audit.Service
	DataIO *dataio.Service
	IAM    *iam.Service
}
