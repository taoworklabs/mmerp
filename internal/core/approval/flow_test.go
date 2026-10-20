package approval

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
)

// step builds a kind-user step whose single UserID stands in for a label (A=1, B=2, ...).
func step(id int64) Element {
	return Element{Step: &StepDef{Approver: Approver{Kind: ByUser, UserIDs: []int64{id}}}}
}

// ids extracts the label of each resolved step, in order.
func ids(steps []StepDef) []int64 {
	out := make([]int64, len(steps))
	for i, st := range steps {
		out[i] = st.Approver.UserIDs[0]
	}
	return out
}

// holdsStub evaluates "amount" (gt/gte against amount) and "kind" (eq against kind).
func holdsStub(amount int, kind string) func(Expr, bool) (bool, error) {
	return func(e Expr, _ bool) (bool, error) {
		switch e.Field {
		case "amount":
			threshold, _ := strconv.Atoi(e.Value)
			if e.Op == OpGTE {
				return amount >= threshold, nil
			}
			return amount > threshold, nil
		case "kind":
			return kind == e.Value, nil
		}
		return false, nil
	}
}

func assertPlan(t *testing.T, got, want Plan) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("plan = %+v, want %+v", got, want)
	}
}

func assertIDs(t *testing.T, got, want []int64) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
}

func assertStart(t *testing.T, got, want []int32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("start = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("start = %v, want %v", got, want)
		}
	}
}

func amountCond(op Op, value string) *Expr {
	return &Expr{Field: "amount", Op: op, Value: value}
}

func kindCond(value string) *Expr {
	return &Expr{Field: "kind", Op: OpEq, Value: value}
}

func TestResolveChooseOneTakesTheFirstHoldingBranch(t *testing.T) {
	const a, b, c = 1, 2, 3
	flow := Flow{{Split: &Split{
		Mode: ChooseOne,
		Branches: []Branch{
			{Condition: amountCond(OpGT, "2000"), Body: Flow{step(a), step(b)}},
			{Condition: amountCond(OpGT, "500"), Body: Flow{step(a)}},
		},
		Otherwise: Flow{step(c)},
	}}}

	plan, steps, err := resolve(flow, holdsStub(600, ""))
	if err != nil {
		t.Fatal(err)
	}
	assertPlan(t, plan, Plan{Seq: []Plan{{Step: 1}}})
	assertIDs(t, ids(steps), []int64{a})

	_, steps, err = resolve(flow, holdsStub(100, ""))
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, ids(steps), []int64{c})

	plan, steps, err = resolve(flow, holdsStub(3000, ""))
	if err != nil {
		t.Fatal(err)
	}
	assertPlan(t, plan, Plan{Seq: []Plan{{Step: 1}, {Step: 2}}})
	assertIDs(t, ids(steps), []int64{a, b})
}

func TestResolveParallelRunsEveryHoldingBranch(t *testing.T) {
	const h, tier, p = 1, 2, 3
	flow := Flow{
		step(h),
		{Split: &Split{
			Mode: Parallel,
			Branches: []Branch{
				{Condition: nil, Body: Flow{step(tier)}},
				{Condition: kindCond("a"), Body: Flow{step(p)}},
			},
		}},
	}

	plan, steps, err := resolve(flow, holdsStub(0, "a"))
	if err != nil {
		t.Fatal(err)
	}
	assertPlan(t, plan, Plan{Seq: []Plan{{Step: 1}, {Par: []Plan{{Step: 2}, {Step: 3}}}}})
	assertIDs(t, ids(steps), []int64{h, tier, p})

	plan, steps, err = resolve(flow, holdsStub(0, "b"))
	if err != nil {
		t.Fatal(err)
	}
	assertPlan(t, plan, Plan{Seq: []Plan{{Step: 1}, {Step: 2}}}) // single-child group collapsed
	assertIDs(t, ids(steps), []int64{h, tier})
}

func TestResolveDropsEmptyParts(t *testing.T) {
	flow := Flow{
		{Split: &Split{Mode: Parallel, Branches: []Branch{
			{Condition: kindCond("x"), Body: Flow{step(1)}},
		}}},
		{Split: &Split{Mode: ChooseOne, Branches: []Branch{
			{Condition: kindCond("y"), Body: Flow{step(2)}},
		}, Otherwise: Flow{}}},
	}

	plan, steps, err := resolve(flow, holdsStub(0, ""))
	if err != nil {
		t.Fatal(err)
	}
	assertPlan(t, plan, Plan{})
	if len(steps) != 0 {
		t.Fatalf("steps = %v, want none", steps)
	}
}

func TestResolveDoubtTakesTheBranch(t *testing.T) {
	leafFor := func(vals map[string]bool) func(Expr) (bool, error) {
		return func(e Expr) (bool, error) { return vals[e.Field], nil }
	}

	// kind is unreadable; amount is 600.
	doubting := func(e Expr, doubt bool) (bool, error) {
		if e.Field == "kind" {
			return doubt, nil
		}
		return holdsStub(600, "")(e, doubt)
	}
	// A doubtful choose-one branch runs, and so does the next branch that surely holds.
	flow1 := Flow{{Split: &Split{Mode: ChooseOne, Branches: []Branch{
		{Condition: kindCond("a"), Body: Flow{step(1)}},
		{Condition: amountCond(OpGT, "500"), Body: Flow{step(2)}},
		{Condition: amountCond(OpGT, "100"), Body: Flow{step(3)}},
	}, Otherwise: Flow{step(4)}}}}
	plan, steps, err := resolve(flow1, doubting)
	if err != nil {
		t.Fatal(err)
	}
	assertPlan(t, plan, Plan{Seq: []Plan{{Par: []Plan{{Step: 1}, {Step: 2}}}}})
	assertIDs(t, ids(steps), []int64{1, 2})
	// Doubt that cannot change the outcome is no doubt: all[kind=a, amount>5000] surely fails.
	flow2 := Flow{{Split: &Split{Mode: ChooseOne, Branches: []Branch{
		{Condition: &Expr{All: []Expr{*kindCond("a"), *amountCond(OpGT, "5000")}}, Body: Flow{step(1)}},
	}, Otherwise: Flow{step(2)}}}}
	if _, steps, err = resolve(flow2, doubting); err != nil {
		t.Fatal(err)
	}
	assertIDs(t, ids(steps), []int64{2})
	// A parallel split runs a doubtful branch.
	flow3 := Flow{{Split: &Split{Mode: Parallel, Branches: []Branch{
		{Condition: kindCond("a"), Body: Flow{step(1)}},
		{Condition: amountCond(OpGT, "5000"), Body: Flow{step(2)}},
	}}}}
	if _, steps, err = resolve(flow3, doubting); err != nil {
		t.Fatal(err)
	}
	assertIDs(t, ids(steps), []int64{1})

	all := Expr{All: []Expr{{Field: "a"}, {Field: "b"}}}
	if ok, err := all.eval(leafFor(map[string]bool{"a": true, "b": false})); err != nil || ok {
		t.Fatalf("all[true,false] = %v, %v; want false, nil", ok, err)
	}

	any := Expr{Any: []Expr{{Field: "a"}, {Field: "b"}}}
	if ok, err := any.eval(leafFor(map[string]bool{"a": false, "b": true})); err != nil || !ok {
		t.Fatalf("any[false,true] = %v, %v; want true, nil", ok, err)
	}

	// all[] has neither children nor a leaf: an empty element, not vacuously true.
	flow := Flow{{Split: &Split{Mode: ChooseOne, Branches: []Branch{
		{Condition: &Expr{All: []Expr{}}, Body: Flow{step(1)}},
	}}}}
	if path, reason := checkFlow(flow, numberField, noRoles, false); reason != "empty_element" {
		t.Fatalf("checkFlow(all[]) = %q, %q; want reason empty_element", path, reason)
	}
}

func TestNextWalksSequenceAndParallel(t *testing.T) {
	plan := Plan{Seq: []Plan{{Step: 1}, {Par: []Plan{{Step: 2}, {Step: 3}}}, {Step: 4}}}

	start, finished := plan.next(nil, nil)
	assertStart(t, start, []int32{1})
	if finished {
		t.Fatal("finished too early with nothing done")
	}

	start, finished = plan.next(map[int32]bool{1: true}, nil)
	assertStart(t, start, []int32{2, 3})
	if finished {
		t.Fatal("finished too early after step 1")
	}

	start, finished = plan.next(map[int32]bool{1: true, 2: true}, map[int32]bool{3: true})
	assertStart(t, start, []int32{})
	if finished {
		t.Fatal("finished too early while step 3 is started but not done")
	}

	start, finished = plan.next(map[int32]bool{1: true, 2: true, 3: true}, nil)
	assertStart(t, start, []int32{4})
	if finished {
		t.Fatal("finished too early before step 4")
	}

	_, finished = plan.next(map[int32]bool{1: true, 2: true, 3: true, 4: true}, nil)
	if !finished {
		t.Fatal("want finished once every leaf is done")
	}
}

func numberField(f string) (record.FieldKind, bool) {
	if f == "amount" {
		return record.Number, true
	}
	return "", false
}

func setField(f string) (record.FieldKind, bool) {
	if f == "parts" {
		return record.Set, true
	}
	return "", false
}

func orgUnitField(f string) (record.FieldKind, bool) {
	if f == "org_unit" {
		return record.OrgUnit, true
	}
	return "", false
}

func noRoles(product, role string) bool  { return false }
func allRoles(product, role string) bool { return true }

func TestCheckFlowReasons(t *testing.T) {
	userStep := func(approvals int32, userIDs ...int64) Element {
		return Element{Step: &StepDef{Approver: Approver{Kind: ByUser, UserIDs: userIDs}, Approvals: approvals}}
	}

	cases := []struct {
		name       string
		flow       Flow
		fieldKind  func(string) (record.FieldKind, bool)
		roleExists func(string, string) bool
		module     bool
		path       string
		reason     string
	}{
		{
			name:   "empty element",
			flow:   Flow{{}},
			path:   "1",
			reason: "empty_element",
		},
		{
			name: "two kinds",
			flow: Flow{{
				Step:  &StepDef{Approver: Approver{Kind: ByUser, UserIDs: []int64{1}}},
				Split: &Split{Mode: ChooseOne, Branches: []Branch{{Body: Flow{step(1)}}}},
			}},
			path:   "1",
			reason: "two_kinds",
		},
		{
			name:       "unknown role",
			flow:       Flow{{Step: &StepDef{Approver: Approver{Kind: ByRole, Product: "hrm", Role: "manager"}}}},
			roleExists: noRoles,
			path:       "1",
			reason:     "unknown_role",
		},
		{
			name:   "missing user",
			flow:   Flow{{Step: &StepDef{Approver: Approver{Kind: ByUser}}}},
			path:   "1",
			reason: "missing_user",
		},
		{
			name:   "no module approvers",
			flow:   Flow{{Step: &StepDef{Approver: Approver{Kind: ByModule}}}},
			module: false,
			path:   "1",
			reason: "no_module_approvers",
		},
		{
			name:      "invalid condition unknown field",
			flow:      Flow{{Split: &Split{Mode: ChooseOne, Branches: []Branch{{Condition: &Expr{Field: "unknown", Op: OpEq, Value: "x"}, Body: Flow{step(1)}}}}}},
			fieldKind: numberField,
			path:      "1.branches.1",
			reason:    "invalid_condition",
		},
		{
			name:      "invalid condition non-number value",
			flow:      Flow{{Split: &Split{Mode: ChooseOne, Branches: []Branch{{Condition: amountCond(OpGT, "not-a-number"), Body: Flow{step(1)}}}}}},
			fieldKind: numberField,
			path:      "1.branches.1",
			reason:    "invalid_condition",
		},
		{
			name:      "invalid condition empty value",
			flow:      Flow{{Split: &Split{Mode: ChooseOne, Branches: []Branch{{Condition: amountCond(OpGT, ""), Body: Flow{step(1)}}}}}},
			fieldKind: numberField,
			path:      "1.branches.1",
			reason:    "invalid_condition",
		},
		{
			name:      "has on a number field",
			flow:      Flow{{Split: &Split{Mode: ChooseOne, Branches: []Branch{{Condition: amountCond(OpHas, "3"), Body: Flow{step(1)}}}}}},
			fieldKind: numberField,
			path:      "1.branches.1",
			reason:    "invalid_condition",
		},
		{
			name:      "has a non-id value",
			flow:      Flow{{Split: &Split{Mode: ChooseOne, Branches: []Branch{{Condition: &Expr{Field: "parts", Op: OpHas, Value: "abc"}, Body: Flow{step(1)}}}}}},
			fieldKind: setField,
			path:      "1.branches.1",
			reason:    "invalid_condition",
		},
		{
			name:      "has zero",
			flow:      Flow{{Split: &Split{Mode: ChooseOne, Branches: []Branch{{Condition: &Expr{Field: "parts", Op: OpHas, Value: "0"}, Body: Flow{step(1)}}}}}},
			fieldKind: setField,
			path:      "1.branches.1",
			reason:    "invalid_condition",
		},
		{
			name:      "has an id is valid",
			flow:      Flow{{Split: &Split{Mode: ChooseOne, Branches: []Branch{{Condition: &Expr{Field: "parts", Op: OpHas, Value: "17"}, Body: Flow{step(1)}}}}}},
			fieldKind: setField,
		},
		{
			name:   "approvals exceed users",
			flow:   Flow{userStep(2, 1)},
			path:   "1",
			reason: "approvals_exceed_users",
		},
		{
			name:   "approvals exceed distinct users: a duplicate id does not count twice",
			flow:   Flow{userStep(2, 5, 5)},
			path:   "1",
			reason: "approvals_exceed_users",
		},
		{
			name: "otherwise on parallel",
			flow: Flow{
				step(1),
				{Split: &Split{Mode: Parallel, Branches: []Branch{{Body: Flow{step(2)}}}, Otherwise: Flow{step(3)}}},
			},
			path:   "2",
			reason: "otherwise_on_parallel",
		},
		{
			name: "nested empty element in a branch body",
			flow: Flow{
				step(1),
				{Split: &Split{Mode: ChooseOne, Branches: []Branch{
					{Condition: amountCond(OpGT, "1"), Body: Flow{{}}},
				}}},
			},
			fieldKind: numberField,
			path:      "2.branches.1.body.1",
			reason:    "empty_element",
		},
		{
			name: "empty element in the otherwise list",
			flow: Flow{
				step(1),
				{Split: &Split{Mode: ChooseOne, Branches: []Branch{
					{Condition: amountCond(OpGT, "1"), Body: Flow{step(2)}},
				}, Otherwise: Flow{{}}}},
			},
			fieldKind: numberField,
			path:      "2.otherwise.1",
			reason:    "empty_element",
		},
		{
			name: "too large: more than 50 steps",
			flow: func() Flow {
				f := make(Flow, 51)
				for i := range f {
					f[i] = step(int64(i))
				}
				return f
			}(),
			path:   "51",
			reason: "too_large",
		},
		{
			// 6 levels of split nesting around a single step: the 6th (innermost)
			// split is where the depth of 5 is exceeded.
			name: "too large: nesting deeper than 5 splits",
			flow: func() Flow {
				f := Flow{step(1)}
				for i := 0; i < 6; i++ {
					f = Flow{{Split: &Split{Mode: ChooseOne, Branches: []Branch{{Body: f}}}}}
				}
				return f
			}(),
			path:   "1.branches.1.body.1.branches.1.body.1.branches.1.body.1.branches.1.body.1.branches.1.body.1",
			reason: "too_large",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fieldKind := tc.fieldKind
			if fieldKind == nil {
				fieldKind = numberField
			}
			roleExists := tc.roleExists
			if roleExists == nil {
				roleExists = allRoles
			}
			path, reason := checkFlow(tc.flow, fieldKind, roleExists, tc.module)
			if path != tc.path || reason != tc.reason {
				t.Fatalf("checkFlow() = %q, %q; want %q, %q", path, reason, tc.path, tc.reason)
			}
		})
	}
}

func TestCheckFlowValidFlowHasNoReason(t *testing.T) {
	flow := Flow{
		step(1),
		{Split: &Split{Mode: ChooseOne, Branches: []Branch{
			{Condition: amountCond(OpGT, "500"), Body: Flow{step(2)}},
		}, Otherwise: Flow{step(3)}}},
	}
	if path, reason := checkFlow(flow, numberField, allRoles, false); reason != "" {
		t.Fatalf("checkFlow() = %q, %q; want no reason", path, reason)
	}
}

func TestCheckFlowOrgUnitCondition(t *testing.T) {
	flow := Flow{{Split: &Split{Mode: ChooseOne, Branches: []Branch{
		{Condition: &Expr{Field: "org_unit", Op: OpWithin, Value: "1"}, Body: Flow{step(1)}},
	}}}}
	if path, reason := checkFlow(flow, orgUnitField, allRoles, false); reason != "" {
		t.Fatalf("checkFlow() = %q, %q; want no reason", path, reason)
	}
}

func TestNormalizeDefaultsCounts(t *testing.T) {
	flow := Flow{{Split: &Split{Mode: ChooseOne, Branches: []Branch{
		{Body: Flow{{Step: &StepDef{Approver: Approver{Kind: ByUser, UserIDs: []int64{1}}}}}},
	}, Otherwise: Flow{{Step: &StepDef{Approver: Approver{Kind: ByUser, UserIDs: []int64{2}}}}}}}}

	normalized := flow.normalize()

	branchStep := normalized[0].Split.Branches[0].Body[0].Step
	if branchStep.Approvals != 1 || branchStep.Rejections != 1 {
		t.Fatalf("branch step = %+v, want Approvals=1, Rejections=1", branchStep)
	}
	otherwiseStep := normalized[0].Split.Otherwise[0].Step
	if otherwiseStep.Approvals != 1 || otherwiseStep.Rejections != 1 {
		t.Fatalf("otherwise step = %+v, want Approvals=1, Rejections=1", otherwiseStep)
	}
}

func TestResolveChooseOneDoubtKeepsLaterBranches(t *testing.T) {
	const a, c = 1, 3
	flow := Flow{{Split: &Split{Mode: ChooseOne, Branches: []Branch{
		{Condition: kindCond("a"), Body: Flow{step(a)}},
	}, Otherwise: Flow{step(c)}}}}

	unreadable := func(_ Expr, doubt bool) (bool, error) { return doubt, nil }
	plan, steps, err := resolve(flow, unreadable)
	if err != nil {
		t.Fatal(err)
	}
	assertPlan(t, plan, Plan{Seq: []Plan{{Par: []Plan{{Step: 1}, {Step: 2}}}}})
	assertIDs(t, ids(steps), []int64{a, c})

	plan, steps, err = resolve(flow, holdsStub(0, "a"))
	if err != nil {
		t.Fatal(err)
	}
	assertPlan(t, plan, Plan{Seq: []Plan{{Step: 1}}})
	assertIDs(t, ids(steps), []int64{a})
}
