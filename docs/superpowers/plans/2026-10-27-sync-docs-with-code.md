# Sync docs with the code Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The docs agents read first (`AGENTS.md`, `docs/roadmap.md`, `docs/architecture.md`, `docs/backend.md`, `docs/documents.md`, `CONTEXT.md`, `docs/techstack.md`) describe the code as it is on `main` today.

**Architecture:** Docs only, no code change. One branch `docs/sync-docs-with-code`, one commit per ticket (#49, #50, #51), one PR closing #48, #49, #50, #51.

**Tech Stack:** Markdown, Mermaid.

**Spec:** GitHub issue #48 and its tickets #49, #50, #51 (`gh issue view <n>`). The issues date from 2026-10-16; this plan adjusts them to `main` at `1db104b` (see "Changes to the spec").

## Changes to the spec (planner's calls)

- **#49:** the status lives in `AGENTS.md` (`CLAUDE.md` only includes it). Besides the "HRM review" section asked for, the roadmap also gets a "Core work after M14" section for ADR-0034 to ADR-0039, which landed after the issue was written and are otherwise missing from the roadmap.
- **#50:** the architecture diagram also shows `shared/partner`, `shared/item`, `shared/tax` (ADR-0039, after the issue).
- **#51:** "`techstack.md` lists `@mantine/hooks`" is obsolete: Mantine was removed in #139 (ADR-0037). The criterion becomes "no doc in scope names Mantine as in use". The `products/sales.md` criterion is already met (line 38 names `amount`, `max_discount` and `items`); verify only.

## Global Constraints

- Docs stay in English. Do not read or edit `docs/archive/`.
- Match the surrounding heading, table and bullet style of each file.
- Links to ADRs use the relative form already used in the file (`./adr/00NN-….md` from `docs/`, `./docs/adr/…` from the repo root).
- Commit messages: Conventional Commits, scope omitted (`docs: …`), ending with the `Co-Authored-By` line. Stage explicit paths only, never `git add -A`.
- PR: title `docs: sync docs with the code`, plain-text body (no `##` headings, no emoji), listing each ticket with its commit, then `Closes #48`, `Closes #49`, `Closes #50`, `Closes #51`.

## Review Focus

- A stale claim left elsewhere in the same file (e.g. `customfield` in the backend tree while removed from the table). Check: the grep in each task's verification step.
- Dead links to ADR files. Check: Task 4's link check.
- The status paragraph in `AGENTS.md` contradicting `roadmap.md` on what comes next. Check: Task 1 step 3 reads both.
- Mermaid syntax broken by the edit. Check: Task 2 step 2 renders the diagram.
- Saying a feature is built when it is not (journal entry mapping, initial import). Check: Task 3's grep.

---

### Task 1: Roadmap and project status (#49)

**Files:**
- Modify: `docs/roadmap.md` (M14 section at line 97; new sections before "## M10")
- Modify: `AGENTS.md:7` (Project status paragraph)

- [ ] **Step 1: Mark M14 done.** Under `## M14. Sales`, add as first line: `**Done** (2026-10-16): [ADR-0027](./adr/0027-sales-product.md).` — same form as M11–M13. Update the milestone diagram line so M14 is no longer the open end (keep the M10 on-hold line).
- [ ] **Step 2: Add two sections after M14, before M10**, no milestone numbers:
  - `## HRM review` — one short paragraph: the business review of HRM after M14 (`docs/hrm-review-plan.md`, rounds one and two) settled its decisions in ADR-0028 to ADR-0033; one bullet per ADR with its title from line 1 of the ADR file and a link.
  - `## Core work after M14` — one paragraph: core and frontend foundations built after Sales opened; one bullet per ADR-0034 to ADR-0039, same form.
  - Then one line: `**Next:** not chosen. M10 and the production-readiness gate stay on hold.`
- [ ] **Step 3: Rewrite the `AGENTS.md` status paragraph** so it says: M11–M14 done; HRM review (ADR-0028–0033) and core work after M14 (ADR-0034–0039: approval flows, own UI kit, data engine, shared master data) done; the next milestone is not chosen; M10 and the gate stay on hold. Keep the Phase 0 sentence and the out-of-scope sentence. Read the new roadmap lines and this paragraph side by side: they must say the same thing.
- [ ] **Step 4: Verify.**
  Run: `grep -n "Done\*\* (2026-10-16)\|## HRM review\|## Core work after M14\|not chosen" docs/roadmap.md AGENTS.md`
  Expected: the M14 done line, both headings, and "not chosen" in both files.
- [ ] **Step 5: Commit** `docs/roadmap.md AGENTS.md` as `docs: roadmap and project status after M14`.

### Task 2: Architecture and backend module lists (#50)

**Files:**
- Modify: `docs/architecture.md:56-82` (section 3 diagram)
- Modify: `docs/backend.md:7-63` (tree, waves table, core modules table) and `:68-81` (module shape)

- [ ] **Step 1: Architecture diagram.** In `modules`: add `sales["sales"]` beside `hrm`, keep `next["later products"]`; change the dotted "never import each other" edge to run `hrm -. … .- sales`. In `shared`: add `partner`, `item`, `tax` beside `posting`. In `core`: add `attachment`, `discussion`, `notification`, `printing`. Keep the `approval --> record` edge.
- [ ] **Step 2: Verify the diagram renders.** Run: `npx -y @mermaid-js/mermaid-cli -i <(sed -n '/## 3. Backend layers/,/^```$/p' docs/architecture.md | sed '1,2d;$d') -o /tmp/layers.svg` (or paste into mermaid.live). Expected: no parse error.
- [ ] **Step 3: `backend.md` lists.** Tree line 14: drop `customfield/`, add `notification/`. Core-modules table: delete the `customfield` row; add `| notification | in-app notifications and the email job; kinds registered by modules ([ADR-0025](./adr/0025-notifications.md)) |` after `discussion`. Keep `customfield` in the "Later" row of the waves table.
- [ ] **Step 4: Module shape.** In the tree under "Module shape", add `jobs.go  # job args and workers of the module's jobs` and a line after the tree: pure calculation files with no DB access (`calc.go` in Sales, `payroll_calc.go` in HRM, `password.go` in `iam`) sit beside the area they serve and are named for what they compute. Add one sentence: besides `types.go`, a module may export building helpers its callers compose, such as `printing.Page` and `printing.Column` in `internal/core/printing/page.go`.
- [ ] **Step 5: Verify.**
  Run: `grep -n "customfield" docs/backend.md; grep -c "notification\|jobs.go\|payroll_calc.go\|printing.Page" docs/backend.md; grep -n "sales\[\|attachment\[\|printing\[\|partner\[" docs/architecture.md`
  Expected: `customfield` only in the waves "Later" row; count ≥ 4; four diagram nodes found.
- [ ] **Step 6: Commit** `docs/architecture.md docs/backend.md` as `docs: module lists in architecture and backend match internal/`.

### Task 3: Glossary, techstack, documents and sales (#51)

**Files:**
- Modify: `docs/documents.md:40`
- Modify: `CONTEXT.md` (Core section; lines 70 and 117)
- Check: `docs/techstack.md`, `products/sales.md:38`

- [ ] **Step 1: `documents.md`.** Change the start of the line-40 bullet to: every API returning one document (detail responses) includes `allowed_actions`; list rows carry none until a row action needs it. Keep the rest of the bullet.
- [ ] **Step 2: `CONTEXT.md`.** Add to `## Core`, near the other list terms: `- **List view** (vi: "Chế độ xem danh sách"): a named filter of a list, saved by one user on the server together with their visible columns and widths; applying it opens the list's URL with that query ([ADR-0031](./docs/adr/0031-personal-list-views.md)).` Append ` Not built: accounting is out of scope.` to "Journal entry mapping"; replace ` Done in M10.` in "Initial import" with ` Not built yet (M10, on hold).`
- [ ] **Step 3: Techstack and sales check.** Confirm `grep -rni mantine docs/techstack.md docs/frontend.md docs/ui.md AGENTS.md` finds nothing that says Mantine is in use (mentions of its removal are fine), and that `products/sales.md:38` names `amount` and `max_discount`. Fix only if not.
- [ ] **Step 4: Verify.**
  Run: `grep -n "List view\|Not built" CONTEXT.md; grep -n "allowed_actions" docs/documents.md | head -2`
  Expected: the new entry, two "Not built" lines, and the reworded bullet.
- [ ] **Step 5: Commit** `docs/documents.md CONTEXT.md` (plus any file fixed in step 3) as `docs: glossary, documents and stack notes match the code`.

### Task 4: Whole-branch check and PR

- [ ] **Step 1: Link check.** Run: `grep -ohE "\./(docs/)?adr/[0-9]{4}-[a-z0-9-]+\.md" AGENTS.md CONTEXT.md docs/*.md | sort -u | sed 's#^\./##;s#^adr/#docs/adr/#' | while read f; do [ -f "$f" ] || echo "missing $f"; done`. Expected: no output.
- [ ] **Step 2: `make lint`** (the i18n and query scans must not care, but confirm). Expected: passes.
- [ ] **Step 3: Push and open the PR** with `gh pr create --base main` per Global Constraints. Do not merge.
