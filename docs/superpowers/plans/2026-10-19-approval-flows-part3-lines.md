# Approval flows, part 3: conditions on document lines — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** An approval condition can test what a document's lines contain, e.g. "any line sells item Z → the price committee approves". The value is picked from a large catalogue by search.

**Architecture:** A line condition is an approval field computed from the lines, not a per-line expression. A record type declares a field of a new kind, `set`: the distinct ids its lines reference (e.g. the items sold). The module writes the field into the header on every save, the same way Sales already writes `max_discount`, which is the largest line discount. A condition tests a set field with the new op `has` ("the set contains this id"). A set field declares `Search`, which finds catalogue entries by text, and `Labels`, which names saved ids. The rule editor picks the value through a server-searched select.

Numeric line conditions ("any line with discount > 10%") are already the type's aggregate numeric fields, such as `max_discount`. Correlated per-line conditions ("item Z with quantity > 100 on the same line") are out of scope.

**Tech Stack:** Go, Postgres, Huma; React + TypeScript; Vitest; Playwright.

**Spec:** `docs/superpowers/specs/2026-10-19-approval-flows-design.md` (goal 4 for lines). Ticket #93. Parts 1, 2 and 4 are done on this branch. Read `internal/core/record/types.go` (`Field`, `FieldKind`), `record.check`, `internal/core/approval/{flow.go,service.go,types.go}` (`opsByKind`, `holds`, `checkExpr`, `Rules`), `web/src/shared/document/flowEditor/ConditionEditor.tsx` and `web/src/shared/ui/form/RemoteSelectField.tsx`.

## Global Constraints

- `approval` and `record` are core and know nothing of items. The product declares the field, computes its value and answers the searches.
- A doubtful condition never removes approval: a missing or malformed set value counts as holding, as for other fields.
- The value is computed by the module on the server, never taken from the client.
- One field value is a canonical string: distinct positive ids in ascending order, joined by `,` (e.g. `"3,17,42"`). The empty set is `""`.
- Migrations: none needed (`record.documents.fields` is already jsonb of strings).
- English identifiers. Comments short, say why, never reference docs or ADRs. i18n in `vi` and `en`. Tokens only in the UI; buttons from `allowed_actions`.
- Conventional Commits, ending with `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`. Stage explicit paths. Branch `approval-flows`.

## Review Focus

1. **A document saved before this change has no `items` field**: a `has` condition on it holds (doubt), and nothing errors. Test: `TestSetFieldMissingHolds` (Task 2).
2. **A malformed set value** (e.g. `"3,x"` written by a buggy module) is refused by `record.check` with `invalid_field`, so it never reaches storage. Test: `TestSetFieldValidated` (Task 2).
3. **The options endpoint doesn't leak catalogue data** to someone who can't manage the product's rules: `forbidden`. Test: `TestFieldOptionsNeedRulePermission` (Task 2).
4. **Editing a quote's lines changes the set**: removing the last line with item Z means a later submission no longer routes to the item rule. Test: `TestItemsFieldFollowsLines` (Task 3).
5. **The saved value's label shows in the editor before any search**, so the condition reads "Mặt hàng · có · TH-01 · Thép đặc chủng" on load. Test: e2e (Task 4).

---

### Task 1: Decision and glossary

**Files:**
- Modify: `docs/adr/0036-approval-flows.md`: in the part-3 bullet ("line fields declared by the record type"), state the decision. Line conditions are aggregate fields:
  - a `set` field holding the distinct ids the lines reference, tested with `has`;
  - numeric aggregates such as a maximum, tested with `gt`/`gte`.

  There are no per-line correlated conditions. Why: the condition model, the doubt rule and the stored snapshot all stay per document, and the customer's cases ("sells item Z", "a discount above X") need nothing more.
- Modify: `CONTEXT.md`: extend **Approval field** with the kinds (number, choice, org unit, set) and add **Set field** (vi: "Trường tập hợp"): the distinct catalogue ids a document's lines reference, e.g. the items of a quotation.
- Modify: `products/sales.md`, the approval fields paragraph: add `items`, the set of item ids on the lines.

- [ ] **Step 1:** Write the changes.
- [ ] **Step 2: Commit** `docs(approval): conditions on lines are set fields`.

---

### Task 2: Set fields in record and approval

**Files:**
- Modify: `internal/core/record/types.go`, `internal/core/record/service.go` (`check`, registration validation)
- Modify: `internal/core/approval/{flow.go,service.go,types.go,handler.go}`
- Test: `internal/core/record/` (the test file that covers `check`, or a new `fields_test.go`), `internal/core/approval/{flow_test.go,service_test.go}`

**Interfaces:**
- `record`:

```go
const Set FieldKind = "set" // distinct ids the lines reference, canonical "3,17,42"

// Field gains, for Set fields (both required for Set, forbidden otherwise; registration panics):
Search func(ctx context.Context, q string) ([]FieldOption, error) // up to 20 matches of q
Labels func(ctx context.Context, ids []int64) ([]FieldOption, error)

type FieldOption struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// SetValue writes ids as a set field's canonical value.
func SetValue(ids []int64) string
```

  `check` accepts a set value only if it is `""` or matches `^[1-9][0-9]{0,18}(,[1-9][0-9]{0,18})*$` with strictly ascending ids; otherwise `invalid_field`.
- `approval`:
  - `OpHas Op = "has"`, and `opsByKind[record.Set] = {OpHas}`. Add `has` to the `Expr.Op` enum tag.
  - `checkExpr`: a `has` value must be a positive integer, else `invalid_condition`.
  - `holds` leaf for `has`: an unreadable value (missing field, or one that fails the canonical pattern) gives `doubt`. Otherwise the result is whether the split list contains the value.
  - `Field` (API) gains `Searchable bool \`json:"searchable"\`` (true for set fields).
  - New `func (s *Service) FieldOptions(ctx context.Context, docType, key, q string, ids []int64) ([]record.FieldOption, error)`:
    - it needs the same permission as `Rules` for the type's product (`managesProduct`), else `forbidden`;
    - an unknown type, field, or a field without `Search` gives `not_found`;
    - with `ids`, it returns `Labels(ids)`; otherwise it returns `Search(q)`.

    Route: `GET /approval-rules/{doc_type}/fields/{key}/options?q=&ids=1,2`. A read: it stays readable while the product is disabled.

- [ ] **Step 1: Write the failing tests:**
  - record `TestSetFieldValidated`: `"3,17"` is OK; `""` is OK; `"17,3"`, `"3,3"`, `"0"`, `"3,x"` and `" 3"` each give `invalid_field`. `SetValue([]int64{17,3,3})` == `"3,17"`.
  - record: registering a `Set` field without `Search` panics.
  - approval (`flow_test.go`): `TestCheckFlowReasons` gains rows: `has` on a number field → `invalid_condition`; `has` with value `"abc"` → `invalid_condition`.
  - approval (`service_test.go`, with `test.doc` given a set field `parts` whose `Search`/`Labels` are stubs):
    - `TestHasRoutesBySet`: rule `[split one: ({field:parts, op:has, value:"17"} → boss), otherwise → []]`. `parts` = `"3,17"` routes to boss, and `"3"` posts at once.
    - `TestSetFieldMissingHolds`: a document without `parts` routes to boss.
    - `TestFieldOptionsNeedRulePermission`: `rules` (product admin) gets the search results, and `outsider` gets `forbidden`. With `ids`, the call returns the labels.
- [ ] **Step 2: Run** `go test ./internal/core/record/... ./internal/core/approval/...` → FAIL.
- [ ] **Step 3: Implement.** Don't commit `make gen`'s web outputs here: commit the regenerated `api/openapi.json` and `schema.gen.ts` only in Task 4, keeping web tsc green.
- [ ] **Step 4: Run** `go build ./...`, `go test ./internal/...` and `make lint` → PASS.
- [ ] **Step 5: Commit** `feat(approval): set fields and the has condition`.

---

### Task 3: Sales declares the items on its lines

**Files:**
- Modify: `internal/modules/sales/service.go`. Where the quote and order types register `Fields`, add `{Key: "items", Kind: record.Set, Label: "sales.doc.items", Search: …, Labels: …}`:
  - `Search` uses the item catalogue search the item picker already uses (find it in `service_item.go` or `queries.sql`), mapping rows to `record.FieldOption{ID, Code, Name}`, with at most 20 results;
  - `Labels` reads items by ids. Add a sqlc query if none exists.
- Modify: `internal/modules/sales/service_doc.go:303`: the header `Fields` gains `"items": record.SetValue(<item ids of the lines>)`.
- Modify: `internal/modules/sales/queries.sql` if a query is added (then run `make gen`; commit the sqlc store only).
- Test: `internal/modules/sales/service_test.go`

- [ ] **Step 1: Write the failing tests:**
  - `TestItemsFieldFollowsLines`: a quote with items A and B gets the field `items` = `SetValue([A,B])`. After an edit removes B, the field is `SetValue([A])`.
  - `TestItemRuleRoutesTheQuote`: the rule `[split one: ({field:items, op:has, value:<B>} → manager), otherwise → []]`. A quote with B routes to the manager, and a quote without B posts at once.
- [ ] **Step 2: Run** `go test ./internal/modules/sales` → FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `go test ./internal/...` and `make lint` → PASS.
- [ ] **Step 5: Commit** `feat(sales): approval conditions may test the items sold`.

---

### Task 4: The editor picks set values by search; e2e

**Files:**
- Regenerate: `api/openapi.json`, `web/src/shared/api/schema.gen.ts`
- Modify: `web/src/shared/document/flowEditor/ConditionEditor.tsx`: for a field with `searchable`, the value input is `RemoteSelectField`, with `queryFn` hitting `/approval-rules/{doc_type}/fields/{key}/options?q=`. Its current label comes from `?ids=<value>` (one query when the condition loads). Read `RemoteSelectField`'s props and `remoteOptions` to fit.
- Modify: i18n vi and en:
  - `shared.rule.op.has`: "có", or what reads naturally after a field label such as "Mặt hàng". Follow `content.md`.
  - `sales.doc.items`: "Mặt hàng" / "Items".
- Create: `web/e2e/approval-lines.spec.ts`, in the exclusive project, with snapshot and restore of the rule it touches, as `approval-actions.spec.ts` does.

- [ ] **Step 1: Implement.** Follow the `mmerp-ui` skill and finish with its checklist.
- [ ] **Step 2: e2e:**
  - `a quote selling the special item goes to the manager`:
    1. Seed through the API helpers: an item `TH-<run>` "Thép đặc chủng", and a manager.
    2. In the editor on the quote rule: a choose-one split. In the branch condition, field "Mặt hàng", op "có", value: type "TH-" and pick the item. The body has one step for the manager role. Save.
    3. Reload: the condition shows the item's label.
    4. A quote with that item is sent and waits in the manager's inbox. A quote without it is approved at once.
- [ ] **Step 3: Run** `make gen` (a second run shows no diff), `make lint`, `pnpm -C web test`, then the full `make e2e` → pass.
- [ ] **Step 4: Commit** `feat(web): approval conditions pick items by search`.
