# Remove Mantine: `web/` on `@mmerp/ui` only — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `web/` imports every UI primitive, hook and look-only block from `@mmerp/ui`; the four `@mantine/*` packages, the Mantine theme, provider, MCP server and vendored skills are gone.

**Architecture:** Phase 2 of the own-UI-kit spec. `web/` links `packages/ui` (`link:`), consumed from TypeScript source. Mantine and the kit coexist until the last task; each task migrates one family of components across all of `web/src`, then adds those names to an ESLint ban on `@mantine/core` (the ratchet) so nothing goes back. `shared/ui` components whose look now lives in the kit (B1–B9, `BulkBar`) are rebuilt as thin behaviour wrappers over the kit blocks, and their CSS modules are deleted.

**Tech Stack:** React 19, Vite, Vitest, Playwright, `@mmerp/ui` (Base UI underneath), React Hook Form, dayjs, ESLint flat config, dependency-cruiser.

**Spec:** `docs/superpowers/specs/2026-10-20-own-ui-kit-design.md` (section "Phase 2", "Enforcement", "Success"), amended by `docs/superpowers/specs/2026-10-21-ui-kit-layout-blocks-design.md` (section "The boundary, amended", table B1–B9). ADR: `docs/adr/0037-own-ui-kit-on-base-ui.md`.

## Global Constraints

- One branch `refactor/remove-mantine` off `main`, one commit per task, one squash-merged PR that closes the spec issue and every task issue. PR body plain text per house style.
- `make lint`, `make test`, `make e2e` and `make image` pass after **every** task, not only at the end.
- Only `packages/ui` imports `@base-ui/react`. `web/` never does.
- `@mmerp/ui` imports nothing from `web/src`, holds no i18n, router, React Hook Form, TanStack Query or user-facing text. Kit changes needed by real call sites are allowed; they keep the kit's `pnpm typecheck && pnpm lint && pnpm test` green.
- `shared/ui` never re-exports a kit component under another name. Areas import look-only primitives and blocks straight from `@mmerp/ui`, the rest through `shared/ui`.
- Area allowlist on `@mmerp/ui`: `Anchor`, `Button`, `Divider`, `Group`, `List`, `Loader`, `Skeleton`, `Stack`, `Text`, `Tooltip`, plus the look-only blocks `Panel`, `FieldList`, `KeyFacts`, `RecordColumns`, `ActionBar`.
- E2E specs keep their steps and outcome assertions; only selectors may change, and only when the DOM shape changed while the user does the same thing. Selects that are typeable today (searchable `Select`, `OrgUnitField`, `RemoteSelectField`) stay typeable.
- Dates typed as `dd/mm/yyyy` keep working; parsing stays in `shared/ui/form/parseDate`.
- Controls are 30px, 40px inside modals and form sections; 44px touch target below 1024px.
- No colour, size or spacing literal outside `packages/ui/src/tokens` and the kit's CSS modules; `shared/ui` CSS reads `--mmerp-*` only.
- Production JS bundle no larger than before Task 1 (measure `web/dist/assets/*.js` gzip total before Task 1 and after Task 9).
- Each task ends with the `mmerp-ui` checklist on the screens it touched and a pass through `/dev/ui`.

## Review Focus

1. **Base UI popups inside a still-Mantine overlay, and the reverse.** A Base UI modal `Dialog` makes everything outside it inert, so a Mantine dropdown portalled to `body` (Select, Combobox, DateInput, Menu, Popover, Tooltip) opened inside a migrated dialog cannot be clicked. That is why inputs (Task 5) and dates (Task 6) move **before** overlays (Task 7), and Task 7 moves Modal, Drawer, Menu, Popover and Tooltip together. Test: e2e on a `FormModal` with a select, a date field and a remote select passes after Tasks 5, 6 and 7.
2. **Two copies of React.** `packages/ui/node_modules` has its own `react` and `@types/react`; resolving kit source from there breaks hooks at runtime ("Invalid hook call") and JSX types in `tsc`. Test: Task 1's Vitest smoke test renders a kit `Button` with a hook-using parent, and `tsc` passes on a file passing a kit element as `ReactNode` to a web component.
3. **Keyboard use after the swap.** Menus (arrows, Escape, typeahead), dialogs (focus trap and return), tabs (arrows), comboboxes (type, arrow, Enter), date fields (arrows, PageUp/Down). Test: one Playwright spec per family in the task that migrates it (see Tasks 5, 6, 7, 8).
4. **40px controls in modals and form sections.** Today set by Mantine theme overrides; after the swap only the kit's `--mmerp-control-height` scope does it. Test: in Task 7, an e2e assertion that a `TextInput` inside a `FormModal` has `offsetHeight === 40`, and one on a list filter has 30.
5. **Toasts with Undo and persistence across navigation.** `notify` keeps its signature; the Undo action must still fire and the toast must survive a route change. Test: Task 8 e2e that deletes a row, sees the toast, clicks Undo, and the row is back.

---

### Task 1: Wiring, tokens, lint scaffolding, ADR accepted

**Files:**
- Modify: `web/package.json` (add `"@mmerp/ui": "link:../packages/ui"`), `web/pnpm-lock.yaml`
- Modify: `web/vite.config.ts` (`resolve.dedupe: ['react', 'react-dom']`; Vitest inherits it)
- Modify: `web/tsconfig.json` (`paths` so `react`, `react-dom`, `react/jsx-runtime` resolve to `web/node_modules` from kit source)
- Modify: `Makefile` (`web/node_modules` depends on `packages/ui/node_modules`), `Dockerfile` (web stage copies `packages/ui` and runs `pnpm -C packages/ui install --frozen-lockfile` before the web install, keeping the `../packages/ui` relative path)
- Modify: `.github/workflows/ci.yml` only if a job installs web deps outside `make`
- Modify: `web/src/app/bootstrap.tsx` (import `@mmerp/ui/styles.css` before the Mantine stylesheets; drop the separate `@fontsource-variable/inter` import, the kit stylesheet brings it)
- Modify: `web/src/shared/ui/theme.ts` (every number comes from `tokens` of `@mmerp/ui`; the `--mmerp-*` resolver entries that `tokens.css` already defines are removed)
- Modify: every `web/src/**/*.module.css` and `web/src/shared/ui/global.css`: `--mantine-*` → `--mmerp-*` (mapping: `--mantine-spacing-X` → `--mmerp-space-X`, `--mantine-radius-X` → `--mmerp-radius-X`, `--mantine-color-default-border` → `--mmerp-color-border`, `--mantine-color-dimmed` → `--mmerp-color-text-dimmed`, `--mantine-primary-color-filled` → `--mmerp-color-primary-6`; for the rest read `packages/ui/src/tokens/tokens.css` and pick the role of the same meaning)
- Modify: `web/eslint.config.js` (ban `@base-ui/react` and `@base-ui/react/*` in `web/src`; add an empty `importNames: []` ratchet on `@mantine/core` for non-area files and keep the area rule's own list in sync with it)
- Modify: `web/lint-fixtures/eslint/*` and its `expected.json` (one fixture importing `@base-ui/react`)
- Modify: `web/scripts/check-colors.sh` only if its exemption names `theme.ts` values that moved
- Modify: `docs/adr/0037-own-ui-kit-on-base-ui.md` (Status `accepted · 2026-10-26`)
- Test: `web/src/shared/ui/kit.test.tsx`

**Interfaces:**
- Produces: `@mmerp/ui` resolvable from any `web/src` file; the ratchet list in `web/eslint.config.js` named `mantineMigrated` (a `const string[]` reused by the general and the area rule), which later tasks append to.

- [ ] **Step 1:** Record the baseline: `pnpm -C web build` then `gzip -c web/dist/assets/*.js | wc -c`; write the number in the PR draft notes. With `make dev` + `make seed`, save Playwright screenshots of `/dev/ui` and the seeded employee list, employee record, sales order list and sales order to `/tmp/mantine-baseline/` for the owner's before/after comparison.
- [ ] **Step 2: Write the failing test** `web/src/shared/ui/kit.test.tsx`: `renders a kit Button inside a component that uses a hook` — render `<Wrapper/>` where `Wrapper` calls `useState` and returns `<Button>Lưu</Button>` from `@mmerp/ui`; assert `getByRole('button', { name: 'Lưu' })` exists.
- [ ] **Step 3:** Run `pnpm -C web vitest run src/shared/ui/kit.test.tsx`. Expected: FAIL, cannot resolve `@mmerp/ui`.
- [ ] **Step 4:** Add the link, dedupe and `paths`; install with `pnpm -C packages/ui install && pnpm -C web install`.
- [ ] **Step 5:** Run the test again and `pnpm -C web typecheck`. Expected: PASS, no "Invalid hook call", no duplicate-React type errors.
- [ ] **Step 6:** Do the stylesheet, `theme.ts` and `--mantine-*` → `--mmerp-*` changes. Run `grep -rn -- '--mantine-' web/src` — expected: only `theme.ts` (Mantine's own resolver), nothing in CSS modules.
- [ ] **Step 7:** Add the ESLint rules and the fixture. Run `make lint`. Expected: PASS, fixture reports exactly `expected.json`.
- [ ] **Step 8:** Run `make test`, `make e2e`, `make image`, `make smoke`. Expected: PASS. Look at `/dev/ui` foundations: values match the kit's Storybook tokens.
- [ ] **Step 9: Commit** `build(web): link @mmerp/ui and read tokens from it` (stage explicit paths).

### Task 2: Layout, typography, buttons, hooks

**Files:**
- Modify: every `web/src/**/*.tsx` importing any of `Box`, `Stack`, `Group`, `Flex`, `Grid`, `SimpleGrid`, `Center`, `Divider`, `Paper`, `Text`, `Title`, `Anchor`, `Code`, `List`, `Button`, `ActionIcon`, `UnstyledButton`, `Badge`, `ThemeIcon`, `Loader`, `Skeleton`, `VisuallyHidden` from `@mantine/core`, or anything from `@mantine/hooks`
- Modify: `web/src/shared/document/flowEditor/FlowWorkspace.tsx` (`useMantineTheme` → `tokens` from `@mmerp/ui`)
- Modify: `web/eslint.config.js` (append the 22 names to `mantineMigrated`; ban `@mantine/hooks`; area `allowImportNames` moves from `@mantine/core` to `@mmerp/ui` with the list in Global Constraints)
- Modify: `web/lint-fixtures/eslint/*` + `expected.json` (an area importing `DataTable` from `@mmerp/ui` is rejected; `Button` is allowed)
- Modify: `web/.dependency-cruiser.cjs` only if it whitelists `@mantine/*` by name

**Interfaces:**
- Consumes: Task 1's `mantineMigrated`.
- Prop mapping (Mantine → kit), applied everywhere: `c="dimmed"` → `tone="dimmed"`, `c="red"`/`"danger"` → `tone="danger"` (other `c` values to the `Tone` of the same meaning); `fw={500|600|650}` → `weight={…}`; `fz` → `size`; `ta` → `align` (`left`→`start`, `right`→`end`); `component={X}` → `render={<X …/>}`; `Flex` → `Group` (responsive `direction` becomes a class in the caller's CSS module); `p`, `px`, `py`, `m*`, `w`, `h`, `miw`, `maw`, `bg`, `lh` → the kit prop if one exists (`Paper` padding, `Group`/`Stack` `gap`), otherwise a class in the calling file's CSS module using `--mmerp-*` variables. Areas pass none of these (they could not before either).

- [ ] **Step 1:** Add the names to `mantineMigrated` and the area allowlist change first. Run `pnpm -C web lint`. Expected: FAIL listing every file still importing them — this is the work list.
- [ ] **Step 2:** Migrate `web/src/shared/**` and `web/src/app/**` files from the list, then `web/src/core`, `hrm`, `sales`. After each directory run `pnpm -C web typecheck`.
- [ ] **Step 3:** Run `pnpm -C web lint && pnpm -C web typecheck`. Expected: PASS, `grep -rn "from '@mantine/hooks'" web/src` empty.
- [ ] **Step 4:** Run `make test` and `make e2e`. Expected: PASS (selectors changed only where the DOM did).
- [ ] **Step 5:** `mmerp-ui` checklist on employee list, employee record, sales order; `/dev/ui` pass. Buttons are 30px, dimmed text is the dimmed token.
- [ ] **Step 6: Commit** `refactor(web): layout, text, buttons and hooks from @mmerp/ui`.

### Task 3: Page, record and form blocks

**Files:**
- Modify: `web/src/shared/ui/page/Page.tsx` (`Page`, `PageHeader`, `BackButton` wrap kit `Page`/`PageHeader`; `navigate(-1)` and texts stay here), `RecordPage.tsx`, `DetailPage.tsx`, `DocumentPage.tsx` (body via kit `RecordColumns`/`ActionBar`; tab-on-URL logic stays)
- Modify: `web/src/shared/ui/form/Form.tsx` (`FormSection`, `FormFields`, `FormRow`, `InputRow`, footer → kit `FormSection`/`FormFields`/`FormBar`; error count, dirty state, Ctrl+S stay)
- Delete: `web/src/shared/ui/FieldList.tsx`, `FieldList.module.css`, `KeyFacts.tsx`, `KeyFacts.module.css` (callers import kit `FieldList`/`KeyFacts`; `MissingValue` moves into a `shared/ui` file of its own if `FieldList.tsx` held it)
- Modify/Delete: `page/Page.module.css`, `RecordPage.module.css`, `DetailPage.module.css`, `DocumentPage.module.css`, `form/Form.module.css` — delete every rule whose look the kit block now owns; delete the file if empty
- Modify: all callers of the deleted files

**Interfaces:**
- Consumes: kit exports `Page`, `PageHeader`, `Panel`, `FieldList`, `KeyFacts`, `RecordColumns`, `ActionBar`, `FormSection`, `FormFields`, `FormBar` (see `packages/ui/src/index.ts`).
- Produces: `shared/ui` `Page`, `PageHeader`, `RecordPage`, `DetailPage`, `DocumentPage`, `Form` keep their current public props (changed only where a prop passed a Mantine type through).

- [ ] **Step 1:** Run `make e2e` on main-equivalent state and note it green (these screens are the e2e's main path; it is the test for this task).
- [ ] **Step 2:** Rebuild `Page.tsx` on the kit blocks; run `pnpm -C web typecheck && make e2e`. Expected: PASS.
- [ ] **Step 3:** Rebuild the record/detail/document pages and `Form.tsx`; delete `FieldList`/`KeyFacts` and the CSS the kit owns. Run `grep -rn "FieldList.module\|KeyFacts.module" web/src` — expected: empty.
- [ ] **Step 4:** Run `make lint`, `make test`, `make e2e`. Expected: PASS.
- [ ] **Step 5:** Checklist on a record page, a document page and a form page at 1280px and 375px (action bar fixed at the bottom below 1024px, record tabs scroll sideways); compare with the kit's `Employees` story.
- [ ] **Step 6: Commit** `refactor(web): page, record and form layout from @mmerp/ui blocks`.

### Task 4: List blocks and data table

**Files:**
- Modify: `web/src/shared/ui/DataTable.tsx` (wraps kit `DataTable`: router link via its `renderLink`, saved widths from `useListView` via its resize callback, every label from i18n), `ListView.tsx`, `page/ListPage.tsx`
- Modify: `web/src/shared/ui/form/FilterChip.tsx`, `MoreActions.tsx`, `BulkBar.tsx` (wrap kit `FilterChip`, `MoreActions`, `BulkBar`), `SearchInput.tsx` (kit `FilterBar.Search` if it fits)
- Delete: `web/src/shared/ui/DataTable.module.css`, `form/FilterChip.module.css`, the look rules of `page/ListPage.module.css`
- Test: existing `web/src/shared/ui/listView.test.ts`, `page/listState.test.ts`, e2e list specs

**Interfaces:**
- Consumes: kit `DataTable`, `Column`, `Selection`, `DataTableLabels`, `FilterBar`, `FilterChip`, `PagingBar`, `MoreActions`, `BulkBar`.
- Produces: `shared/ui` `DataTable`, `ListView`, `ListPage` keep their public props; `Column` definitions in areas are unchanged or changed by one mechanical rename across all areas.

- [ ] **Step 1:** Add a Vitest test `web/src/shared/ui/DataTable.test.tsx` → `reports a resized column width to the list view`: render the shared `DataTable` with two columns, fire the kit's resize callback for column `name` with `180`, assert the `useListView` setter received `{ name: 180 }`. Run it — Expected: FAIL (wrapper not built yet).
- [ ] **Step 2:** Rebuild `DataTable.tsx` on the kit block. Run the test — Expected: PASS.
- [ ] **Step 3:** Rebuild filter chip, more actions, bulk bar, paging, list page; delete their CSS. Run `make lint && make test && make e2e`. Expected: PASS (sorting, selection, bulk actions, paging, card list below 1024px in e2e).
- [ ] **Step 4:** Checklist on the employee list and sales order list at 1280px and 375px; groups with totals and sticky header look like the kit story.
- [ ] **Step 5: Commit** `refactor(web): lists and data table on @mmerp/ui blocks`.

### Task 5: Form inputs

**Files:**
- Modify: `web/src/shared/ui/form/fields.tsx`, `RemoteSelectField.tsx`, `OrgUnitField.tsx`, `SearchInput.tsx`, `SensitiveField.tsx`, `LegalEntity.tsx`, `YearFilter.tsx`, `LinesTable.tsx`, `EditablePanel.tsx`, `web/src/shared/ui/InlineEditField.tsx`, `web/src/shared/document/AttachmentPanel.tsx` (`FileButton` → kit `FileInput` or a hidden `<input type="file">` behind a kit `Button`), every other user of `TextInput`, `PasswordInput`, `Textarea`, `NumberInput`, `Checkbox`, `Select`, `MultiSelect`, `Combobox`, `useCombobox`, `SegmentedControl`, `FileInput`, `FileButton`, `Input`, `InputWrapper`
- Delete: `web/src/shared/ui/SegmentedControl.module.css` if the kit's look replaces it
- Modify: `web/eslint.config.js` (append those names to `mantineMigrated`)
- Test: `web/e2e/keyboard-inputs.spec.ts` (new)

**Interfaces:**
- Consumes: kit `Field`, `TextInput`, `PasswordInput`, `Textarea`, `NumberInput`, `Checkbox`, `Select` (`SelectItem`), `Combobox`, `ComboboxButton`, `MultiSelect`, `matchesQuery`, `SegmentedControl`, `FileInput`.
- Produces: the React Hook Form field components in `shared/ui/form` keep their props; `RemoteSelectField` keeps typing-to-search.

- [ ] **Step 1: Write the failing e2e** `keyboard-inputs.spec.ts` → `remote select is typeable and keyboard-selectable`: on the employee create form, focus the org unit field, type the first three letters of a seeded unit, press ArrowDown then Enter, assert the field shows that unit; then `searchable select accepts typed text` on a searchable `Select` the same way. It passes on Mantine today — keep it as the regression guard and run it now: Expected PASS.
- [ ] **Step 2:** Append the names to `mantineMigrated`; run `pnpm -C web lint` for the work list (Expected: FAIL listing files).
- [ ] **Step 3:** Migrate `fields.tsx` and the field components, then the remaining callers. Run `pnpm -C web typecheck && pnpm -C web test`. Expected: PASS (incl. `RemoteSelectField.test.ts`, `parseDate.test.ts`).
- [ ] **Step 4:** Run `make lint && make e2e`. Expected: PASS incl. `keyboard-inputs.spec.ts`; the inputs inside a still-Mantine `Modal` open and select (Review Focus 1).
- [ ] **Step 5: Commit** `refactor(web): form inputs from @mmerp/ui`.

### Task 6: Dates

**Files:**
- Modify: `web/src/shared/ui/form/fields.tsx` (`DateField` on kit `DateInput`, `parseDate` passed in, locale names and first day of week from i18n/dayjs), `MonthFilter.tsx` (kit `MonthPickerInput`/`MonthPicker`), the third `@mantine/dates` user
- Modify: `web/src/app/bootstrap.tsx` (remove `DatesProvider` and `@mantine/dates/styles.css`)
- Modify: `web/eslint.config.js` (ban `@mantine/dates` outright)
- Test: `web/e2e/keyboard-inputs.spec.ts` (add cases)

**Interfaces:**
- Consumes: kit `DateInput`, `MonthPickerInput`, `MonthPicker`, `Calendar`, `CalendarLocale`, `FirstDayOfWeek`.
- Produces: `DateField` and `MonthFilter` keep their props and value format (`YYYY-MM-DD` / `YYYY-MM` strings as today).

- [ ] **Step 1: Write the e2e cases** in `keyboard-inputs.spec.ts`: `date field accepts dd/mm/yyyy typed by hand` (type `05/03/2026` in a leave request's start date, blur, submit, assert the saved record shows `05/03/2026`); `date picker moves by keyboard` (open, ArrowRight, Enter, assert the next day is chosen); `month filter changes the list month`. Run on current code: Expected PASS.
- [ ] **Step 2:** Migrate the three files and `bootstrap.tsx`; add the ban. Run `pnpm -C web lint && pnpm -C web typecheck`. Expected: PASS, `grep -rn "@mantine/dates" web/src` empty.
- [ ] **Step 3:** Run `make test && make e2e`. Expected: PASS. Check Vietnamese weekday names and Monday-first week on `/dev/ui` forms.
- [ ] **Step 4: Commit** `refactor(web): date fields from @mmerp/ui`.

### Task 7: Overlays

**Files:**
- Modify: `web/src/shared/ui/page/FormModal.tsx`, `ModalFooter.tsx`, `confirm.tsx`, `SidePanel.tsx`, `MenuAction.tsx`, and every user of `Modal`, `Drawer`, `Menu`, `Popover`, `Tooltip`, `ScrollArea` (all migrated in this one task — Review Focus 1)
- Modify: `web/src/shared/ui/global.css` and `theme.ts` (remove the Mantine overrides that set 40px inside modals; the kit `Dialog` body scope sets `--mmerp-control-height` to the large control)
- Modify: `web/eslint.config.js` (append to `mantineMigrated`)
- Test: `web/e2e/keyboard-overlays.spec.ts` (new)

**Interfaces:**
- Consumes: kit `Dialog` (`DialogSize`), `Drawer`, `Menu`, `Popover`, `Tooltip`, `ScrollArea`.
- Produces: `FormModal`, `confirm`, `SidePanel`, `MenuAction` keep their signatures.

- [ ] **Step 1: Write the e2e** `keyboard-overlays.spec.ts`: `dialog traps focus and returns it` (open a `FormModal` from a button, Tab past the last control lands on the first, Escape closes, focus is back on the opening button); `row menu works by keyboard` (focus a row's more-actions button, Enter, ArrowDown, Escape closes); `controls are 40px in a modal and 30px in a filter` (assert `offsetHeight` 40 for a `TextInput` in `FormModal`, 30 for the list search input); `select and date field work inside a dialog` (open a `FormModal`, pick an option from a select and a date from the picker, submit, assert saved). Run on current code: Expected PASS.
- [ ] **Step 2:** Migrate all overlay users in one pass; append names to the ratchet. Run `pnpm -C web lint && pnpm -C web typecheck`. Expected: PASS.
- [ ] **Step 3:** Run `make test && make e2e`. Expected: PASS incl. `keyboard-overlays.spec.ts`.
- [ ] **Step 4: Commit** `refactor(web): dialogs, menus, popovers and tooltips from @mmerp/ui`.

### Task 8: Navigation, shell, feedback, data display, toasts

**Files:**
- Modify: `web/src/app/AppLayout.tsx` (+ `.module.css`: kit `AppShell`, `NavLink`, `Burger`), `web/src/shared/ui/notify.tsx`, `Toasts.tsx` (kit `Toaster`/`toast`; `notify` signature unchanged), `Notice.tsx` (kit `Alert`), `Activity.tsx` (kit `Timeline`), `avatar.tsx` (kit `Avatar`; name hashing stays), `InboxPage.tsx`, and every user of `Tabs`, `NavLink`, `Breadcrumbs`, `Pagination`, `Burger`, `AppShell`, `Table`, `Timeline`, `Avatar`, `Indicator`, `Alert`, `Notification`, `TableOfContents`, `Collapse` (`ElementPanel.tsx`: render the advanced block conditionally)
- Modify: `web/src/app/bootstrap.tsx` (mount kit `Toaster` in place of Mantine `Notifications`; remove `@mantine/notifications/styles.css`)
- Modify: `web/eslint.config.js` (append names; ban `@mantine/notifications`)
- Test: `web/e2e/toasts.spec.ts` (new)

**Interfaces:**
- Consumes: kit `AppShell`, `NavLink`, `Burger`, `Tabs`, `Breadcrumbs`, `Pagination`, `TableOfContents`, `Table`, `Timeline`, `Avatar`, `Indicator`, `Alert`, `toast`, `Toaster`, `ToastOptions`.
- Produces: `notify(…)` and `useAction` behave as today (position, durations, Undo).

- [ ] **Step 1: Write the e2e** `toasts.spec.ts` → `undo restores a deleted row` (delete a draft row, toast appears, click Undo, row is back) and `toast survives navigation` (trigger a save that navigates, toast still visible on the new page); plus `tabs move by arrow keys` on a record page. Run on current code: Expected PASS.
- [ ] **Step 2:** Migrate shell, navigation, feedback and display users; swap the toaster. Run `pnpm -C web lint && pnpm -C web typecheck`. Expected: PASS; `grep -rn "from '@mantine/" web/src` lists only `theme.ts` and `bootstrap.tsx`'s `MantineProvider`.
- [ ] **Step 3:** Run `make test && make e2e`. Expected: PASS. Check the shell below 1024px (burger opens the nav, 44px targets).
- [ ] **Step 4: Commit** `refactor(web): shell, navigation, feedback and toasts from @mmerp/ui`.

### Task 9: Remove Mantine

**Files:**
- Modify: `web/package.json`, `web/pnpm-lock.yaml` (remove `@mantine/core`, `dates`, `hooks`, `notifications`)
- Delete: `web/src/shared/ui/theme.ts` (or keep only non-Mantine exports if any caller needs them; it is gone if empty); remove `MantineProvider` and `@mantine/core/styles.css` from `bootstrap.tsx`
- Modify: `web/eslint.config.js` (drop the ratchet and the `@mantine/*` rules; keep the `@base-ui/react` ban and the area allowlist), `web/lint-fixtures/**` + `expected.json`
- Modify: `web/src/app/devui/mantine.tsx` → rename to `kit.tsx`: a short page listing the area allowlist with a link to Storybook (`make storybook`, `:6006`); update `catalogue.ts`, `index.tsx`, `catalogue.test.ts`
- Modify: `.mcp.json` (remove the `mantine` server), `skills-lock.json`; delete `.claude/skills/mantine-combobox`, `mantine-custom-components`, `mantine-form`
- Modify: `AGENTS.md` (Agent setup: drop the Mantine MCP and skills paragraph; mention Storybook as the primitive reference) — use the `house-style` skill
- Modify: `docs/techstack.md`, `docs/ui.md`, `docs/frontend.md`, `docs/architecture.md` if it names Mantine, `.claude/skills/mmerp-ui/SKILL.md` and `references/{tokens,components,forms,tables,checklist}.md`: every Mantine prop, variable or component name → the `@mmerp/ui` equivalent
- Modify: `web/scripts/check-colors.sh` (exemption no longer names `theme.ts`)

- [ ] **Step 1:** Remove the packages: `pnpm -C web remove @mantine/core @mantine/dates @mantine/hooks @mantine/notifications`. Run `pnpm -C web typecheck`. Expected: FAIL only in `theme.ts`/`bootstrap.tsx` — fix them.
- [ ] **Step 2:** Rework `/dev/ui` layer 3 and its test; config, skills, docs as listed.
- [ ] **Step 3:** Verify the spec's success checks:
  - `grep -rni mantine web/src web/e2e web/package.json .claude .mcp.json AGENTS.md docs/*.md` → empty (ADRs and `docs/superpowers/**` keep history).
  - `pnpm -C web why @mantine/core` → nothing; `grep -rn "@base-ui/react" web/src` → empty.
  - `make gen` (no diff), `make lint`, `make test`, `make e2e`, `make image`, `make smoke` → PASS.
  - Bundle gzip total ≤ the Task 1 baseline.
- [ ] **Step 4:** With `make reset-db && make seed`, take the same screenshots as Task 1 Step 1 into `/tmp/mantine-after/` and put them before the owner side by side; the owner accepts the differences before the PR is marked ready.
- [ ] **Step 5: Commit** `chore(web): remove Mantine`.

### Wrap-up

- [ ] Run the `code-review` skill on the branch, fix findings, then open the PR `refactor(web): replace Mantine with @mmerp/ui` (plain-text body: one bullet per task with its commit, the deliberate look changes, no migrations, `Closes #<spec>` and each task issue).
