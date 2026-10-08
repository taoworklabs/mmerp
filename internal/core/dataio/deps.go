package dataio

import (
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
)

type Deps struct {
	IAM   *iam.Service
	Audit *audit.Service
}
