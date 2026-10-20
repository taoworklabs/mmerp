package approval

import (
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
)

type Deps struct {
	IAM          *iam.Service
	Record       *record.Service
	Audit        *audit.Service
	Notification *notification.Service
	Setting      *setting.Service
}
