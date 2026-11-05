package accounting_test

import (
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/modules/accounting"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func TestAccountingRoles(t *testing.T) {
	core := coretest.New(t, "accounting")
	accounting.NewService(accounting.Deps{IAM: core.IAM, Audit: core.Audit, Record: core.Record})
	id, err := core.IAM.CreateUser(core.Admin, "acc", "Acc", "long enough")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := core.IAM.GrantRole(core.Admin, id, "accounting", "accountant", nil); err != nil {
		t.Fatal(err)
	}
	ctx := platform.WithActor(core.Ctx, id)
	for perm, want := range map[string]bool{accounting.PermView: true, accounting.PermEntryEdit: true, accounting.PermPost: true, accounting.PermManage: false} {
		if got := core.IAM.RequireTenantWide(ctx, "accounting", perm) == nil; got != want {
			t.Errorf("accountant holds %s = %v, want %v", perm, got, want)
		}
	}
}
