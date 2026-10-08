package iam

import (
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/setting"
)

type Deps struct {
	Setting *setting.Service
	Audit   *audit.Service
}
