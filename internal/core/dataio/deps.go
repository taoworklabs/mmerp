package dataio

import (
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
)

type Deps struct {
	IAM    *iam.Service
	Audit  *audit.Service
	Record *record.Service
}
