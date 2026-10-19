package record_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/approval"
	"github.com/taoworklabs/mmerp/internal/core/coretest"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/record/recordtest"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// TestContract runs recordtest on a bare document type, proving the suite itself.
func TestContract(t *testing.T) {
	recordtest.Run(t, func(t *testing.T) recordtest.Harness {
		core := coretest.New(t, "test")
		ids, rec, appr, ctx, admin := core.IAM, core.Record, core.Approval, core.Ctx, core.Admin
		ids.RegisterRoles("test", map[string][]string{"member": {"test.doc.edit"}, "approver": {"test.doc.approve", "test.doc.edit"}})

		var mu sync.Mutex
		effects, broken, unready := map[int64]int{}, map[int64]bool{}, map[int64]bool{}
		rec.Register(record.Type{
			Code: "test.doc", Product: "test", Kind: record.Document, NumberPrefix: "T",
			// Members of the document's unit may do anything with it.
			Perms:    record.Perms{View: "test.doc.edit", Edit: "test.doc.edit", ViewFiles: "test.doc.edit", Attach: "test.doc.edit"},
			OnDelete: noRows,
			BeforeSubmit: func(_ context.Context, d record.Doc) error {
				mu.Lock()
				defer mu.Unlock()
				if unready[d.ID] {
					return &platform.Error{Status: 422, Code: "not_ready"}
				}
				return nil
			},
			OnTransition: func(_ context.Context, d record.Doc, _ record.Status) error {
				mu.Lock()
				defer mu.Unlock()
				switch {
				case d.Status == record.Cancelled:
					effects[d.ID]--
				case d.Status != record.Posted:
				case broken[d.ID]:
					return errors.New("posting broken")
				default:
					effects[d.ID]++
				}
				return nil
			},
		})

		c := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"}))
		a := must(t)(ids.CreateOrgUnit(admin, iam.OrgUnitInput{ParentID: &c, Kind: "department", Name: "A"}))
		actorID := must(t)(ids.CreateUser(admin, "actor", "Actor", "long enough"))
		must(t)(ids.GrantRole(admin, actorID, "test", "member", nil))
		actor := platform.WithActor(ctx, actorID)
		approverID := must(t)(ids.CreateUser(admin, "approver", "Approver", "long enough"))
		must(t)(ids.GrantRole(admin, approverID, "test", "approver", nil))
		approver := platform.WithActor(ctx, approverID)

		header := func(date string) record.Header { return record.Header{Date: date, OrgUnitID: a} }
		type detail struct {
			Doc            record.Doc
			AllowedActions []string
		}
		return recordtest.Harness{
			Core: core, Actor: actor, Admin: admin, LegalEntity: c,
			Create: func(ctx context.Context, date, requestID string) (record.Ref, error) {
				h := header(date)
				h.RequestID = requestID
				d, err := rec.Create(ctx, "test.doc", h)
				return d.Ref, err
			},
			Edit: func(ctx context.Context, ref record.Ref, version int32, date string) error {
				_, err := rec.Edit(ctx, ref, version, header(date))
				return err
			},
			Get: func(ctx context.Context, ref record.Ref) (any, error) {
				d, err := rec.Read(ctx, ref)
				if err != nil {
					return nil, err
				}
				actions, err := rec.Actions(ctx, d)
				return detail{d, actions}, err
			},
			Reject: func(instance int64) error { return appr.Reject(approver, instance, 1, "no") },
			BreakSubmit: func(_ *testing.T, ref record.Ref) {
				mu.Lock()
				defer mu.Unlock()
				unready[ref.ID] = true
			},
			RequireApproval: func(t *testing.T) {
				role := approval.Approver{Kind: "role", Product: "test", Role: "approver"}
				if err := appr.SaveRule(admin, "test.doc", record.Post, approval.RuleInput{
					Flow: approval.Flow{{Step: &approval.StepDef{Approver: role}}}, MaxLevels: 3, FallbackProduct: "test", FallbackRole: "approver",
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

// adminDoc registers t as the document type test.doc, which the tenant administrator may do anything with.
func adminDoc(core *coretest.Core, t record.Type) {
	core.IAM.RegisterRoles("test", map[string][]string{"member": {"test.doc.edit"}})
	t.Code, t.Product, t.Kind, t.NumberPrefix = "test.doc", "test", record.Document, "T"
	t.Perms = record.Perms{View: "test.doc.edit", Edit: "test.doc.edit"}
	t.OnDelete = noRows
	core.Record.Register(t)
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
	core := coretest.New(t, "test")
	ids, rec, admin := core.IAM, core.Record, core.Admin
	adminDoc(core, record.Type{})
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
	core := coretest.New(t, "test")
	ids, rec, admin := core.IAM, core.Record, core.Admin
	adminDoc(core, record.Type{})
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
		pgtest.AwaitLockWait(t, core.Pool)
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
	core := coretest.New(t, "test")
	ids, rec, admin := core.IAM, core.Record, core.Admin
	refuse := errors.New("not ready")
	var calls int
	adminDoc(core, record.Type{
		BeforeSubmit: func(context.Context, record.Doc) error {
			calls++
			if calls == 1 {
				return refuse
			}
			return nil
		}})
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
