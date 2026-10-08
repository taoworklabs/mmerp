package attachment

import (
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/record"
)

type Deps struct {
	Record *record.Service
	Audit  *audit.Service
}
