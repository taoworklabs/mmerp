package discussion

import (
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/record"
)

type Deps struct {
	Record       *record.Service
	Audit        *audit.Service
	IAM          *iam.Service
	Notification *notification.Service
}
