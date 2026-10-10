package record_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// TestContract runs recordtest on a bare document type, proving the suite itself.
func TestContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		pool := pgtest.New(t)
		ctx := platform.WithKeyring(platform.WithProducts(platform.WithDB(t.Context(), pool), []string{"test"}), pgtest.Keyring())
		ids := iam.NewService(iam.Deps{Setting: setting.NewService(), Audit: audit.NewService()})
		ids.RegisterRoles("test", map[string][]string{"approver": {"test.doc.approve"}})
		rec := record.NewService(record.Deps{IAM: ids, Numbering: numbering.NewService(), Audit: audit.NewService()})
		ids.SetTreeHook(rec)
		appr := approval.NewService(approval.Deps{IAM: ids, Record: rec, Audit: audit.NewService(), Notification: notification.NewService(notification.Deps{Record: rec, IAM: ids, Audit: audit.NewService(), Setting: setting.NewService()})})
		rec.SetApprovalGate(appr)

		var mu sync.Mutex
		effects, broken := map[int64]int{}, map[int64]bool{}
		rec.Register(record.Type{
			Code: "test.doc", Product: "test", Kind: record.Document, NumberPrefix: "T",
			Can: func(context.Context, int64, record.Action) (bool, error) { return true, nil },
			OnTransition: func(_ context.Context, d record.Doc, _ record.Status) error {
				mu.Lock()
				defer mu.Unlock()
				if d.Status != record.Posted {
					return nil
				}
				if broken[d.ID] {
					return errors.New("posting broken")
				}
				effects[d.ID]++
				return nil
			},
		})

		admin := platform.WithActor(ctx, must(t)(ids.CreateAdmin(ctx, "admin", "Admin", "long enough")))
		c := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
		a := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{ParentID: &c, Kind: "department", Name: "A"}))
		actor := platform.WithActor(ctx, must(t)(ids.CreateUser(admin, "actor", "Actor", "long enough")))
		approverID := must(t)(ids.CreateUser(admin, "approver", "Approver", "long enough"))
		must(t)(ids.GrantRole(admin, approverID, "test", "approver", nil))
		approver := platform.WithActor(ctx, approverID)

		header := func(date string) record.Header { return record.Header{Date: date, OrgUnitID: a} }
		return recordtest.Harness{
			Record: rec, Actor: actor, Admin: admin, LegalEntity: c,
			Create: func(ctx context.Context, date string) (record.Ref, error) {
				d, err := rec.Create(ctx, "test.doc", header(date))
				return d.Ref, err
			},
			Edit: func(ctx context.Context, ref record.Ref, version int32, date string) error {
				_, err := rec.Edit(ctx, ref, version, header(date))
				return err
			},
			RequireApproval: func(t *testing.T) {
				role := approval.Approver{Kind: "role", Product: "test", Role: "approver"}
				if err := appr.SaveRule(admin, "test.doc", approval.RuleInput{
					Steps: []approval.Step{{Approver: role}}, MaxLevels: 3, FallbackProduct: "test", FallbackRole: "approver",
				}); err != nil {
					t.Fatal(err)
				}
			},
			Approve: func(instance int64) error { return appr.Approve(approver, instance, 1) },
			BreakPosting: func(_ *testing.T, ref record.Ref) {
				mu.Lock()
				defer mu.Unlock()
				broken[ref.ID] = true
			},
			Effects: func(_ *testing.T, ref record.Ref) int {
				mu.Lock()
				defer mu.Unlock()
				return effects[ref.ID]
			},
		}
	})
}

func must(t *testing.T) func(int64, error) int64 {
	return func(v int64, err error) int64 {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func TestUnitWithDocumentsCannotChangeLegalEntity(t *testing.T) {
	pool := pgtest.New(t)
	ctx := platform.WithKeyring(platform.WithProducts(platform.WithDB(t.Context(), pool), []string{"test"}), pgtest.Keyring())
	ids := iam.NewService(iam.Deps{Setting: setting.NewService(), Audit: audit.NewService()})
	rec := record.NewService(record.Deps{IAM: ids, Numbering: numbering.NewService(), Audit: audit.NewService()})
	ids.SetTreeHook(rec)
	rec.Register(record.Type{Code: "test.doc", Product: "test", Kind: record.Document, NumberPrefix: "T",
		Can: func(context.Context, int64, record.Action) (bool, error) { return true, nil }})
	admin := platform.WithActor(ctx, must(t)(ids.CreateAdmin(ctx, "admin", "Admin", "long enough")))
	c1 := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C1"}))
	c2 := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C2"}))
	a := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{ParentID: &c1, Kind: "department", Name: "A"}))
	d, err := rec.Create(admin, "test.doc", record.Header{Date: "2026-03-10", OrgUnitID: a})
	if err != nil || d.Number != "T-2026-00001" || d.LegalEntityID != c1 {
		t.Fatalf("create: %+v %v", d, err)
	}
	err = ids.UpdateOrgUnit(admin, a, iam.OrgUnitInput{ParentID: &c2, Kind: "department", Name: "A"})
	if e, ok := errors.AsType[*platform.Error](err); !ok || e.Code != "org_unit_has_documents" {
		t.Fatalf("got %v", err)
	}
	if err := ids.UpdateOrgUnit(admin, a, iam.OrgUnitInput{ParentID: &c1, Kind: "department", Name: "A2"}); err != nil {
		t.Fatal(err)
	}
}

// A document created while its unit moves to another legal entity waits for the move,
// then belongs to the legal entity the unit is under.
func TestCreateWaitsForATreeMove(t *testing.T) {
	pool := pgtest.New(t)
	ctx := platform.WithKeyring(platform.WithProducts(platform.WithDB(t.Context(), pool), []string{"test"}), pgtest.Keyring())
	ids := iam.NewService(iam.Deps{Setting: setting.NewService(), Audit: audit.NewService()})
	rec := record.NewService(record.Deps{IAM: ids, Numbering: numbering.NewService(), Audit: audit.NewService()})
	ids.SetTreeHook(rec)
	rec.Register(record.Type{Code: "test.doc", Product: "test", Kind: record.Document, NumberPrefix: "T",
		Can: func(context.Context, int64, record.Action) (bool, error) { return true, nil }})
	admin := platform.WithActor(ctx, must(t)(ids.CreateAdmin(ctx, "admin", "Admin", "long enough")))
	c1 := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C1"}))
	c2 := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C2"}))
	a := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{ParentID: &c1, Kind: "department", Name: "A"}))
	var wg sync.WaitGroup
	var d record.Doc
	var created error
	if err := platform.InTx(admin, func(ctx context.Context) error {
		if err := ids.UpdateOrgUnit(ctx, a, iam.OrgUnitInput{ParentID: &c2, Kind: "department", Name: "A"}); err != nil {
			return err
		}
		wg.Go(func() { d, created = rec.Create(admin, "test.doc", record.Header{Date: "2026-03-10", OrgUnitID: a}) })
		time.Sleep(100 * time.Millisecond)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if created != nil || d.LegalEntityID != c2 {
		t.Fatalf("created %+v %v, want legal entity %d", d, created, c2)
	}
}

// BeforeSubmit runs when a draft is sent, approval or not, and its error keeps the draft.
func TestBeforeSubmitKeepsADraft(t *testing.T) {
	pool := pgtest.New(t)
	ctx := platform.WithKeyring(platform.WithProducts(platform.WithDB(t.Context(), pool), []string{"test"}), pgtest.Keyring())
	ids := iam.NewService(iam.Deps{Setting: setting.NewService(), Audit: audit.NewService()})
	rec := record.NewService(record.Deps{IAM: ids, Numbering: numbering.NewService(), Audit: audit.NewService()})
	refuse := errors.New("not ready")
	var calls int
	rec.Register(record.Type{Code: "test.doc", Product: "test", Kind: record.Document, NumberPrefix: "T",
		Can: func(context.Context, int64, record.Action) (bool, error) { return true, nil },
		BeforeSubmit: func(context.Context, record.Doc) error {
			calls++
			if calls == 1 {
				return refuse
			}
			return nil
		}})
	admin := platform.WithActor(ctx, must(t)(ids.CreateAdmin(ctx, "admin", "Admin", "long enough")))
	c := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
	d, err := rec.Create(admin, "test.doc", record.Header{Date: "2026-03-10", OrgUnitID: c})
	if err != nil {
		t.Fatal(err)
	}
	if err := rec.Transition(admin, d.Ref, 1, record.Posted); !errors.Is(err, refuse) {
		t.Fatalf("first send: %v", err)
	}
	if got, _ := rec.Get(admin, d.Ref); got.Status != record.Draft || got.Version != 1 {
		t.Fatalf("after refusal: %+v", got)
	}
	if err := rec.Transition(admin, d.Ref, 1, record.Posted); err != nil {
		t.Fatal(err)
	}
	// Cancelling is not a submission.
	if err := rec.Transition(admin, d.Ref, 2, record.Cancelled); err != nil || calls != 2 {
		t.Fatalf("cancel: %v, calls %d", err, calls)
	}
}
