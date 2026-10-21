# `@mmerp/ui` layout blocks and primitive gaps: Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the primitive gaps a real screen exposed, move the look-only half of `web/src/shared/ui` into `@mmerp/ui` as layout blocks, and prove it with three HRM screen stories that use kit exports only.

**Architecture:** Primitives stay in `src/primitives/<folder>/`. Blocks go in a new `src/blocks/<group>/` tree (`page`, `form`, `list`, `table`), each a component plus CSS module plus story, exported from `src/index.ts`. Blocks take every text as a prop and every link through `render`/`renderLink`; they never navigate, persist, fetch or read form state. Screen stories in `src/screens/` compose them with no CSS of their own.

**Tech Stack:** as phase 1: React 19, TypeScript 5.9, `@base-ui/react` 1.9, CSS modules over `--mmerp-*` variables, Vitest 5 browser mode, Storybook 10.6 with `addon-a11y` and `addon-vitest`, pnpm 11, Node 24.

**Spec:** `docs/superpowers/specs/2026-10-21-ui-kit-layout-blocks-design.md`. Read it first; section letters and numbers (A1…A10, B1…B9) below refer to it. Background: `docs/superpowers/specs/2026-10-20-own-ui-kit-design.md`, `docs/adr/0037-own-ui-kit-on-base-ui.md`, and the throwaway spike `packages/ui/src/demo/` (uncommitted; read it for what each block must replace, then delete it in Task 11).

## Global Constraints

- Branch `feat/ui-kit-layout-blocks` (already created off `main`). One commit per task; each task is one ticket. Conventional Commits, scope `ui` (Task 1: `docs(ui)`). Messages end with `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`. Stage explicit paths only, never `git add -A` / `git add .`.
- Change nothing outside `packages/ui/` and the docs Task 1 names. `git diff main -- web Dockerfile Makefile .github/workflows/ci.yml` stays empty.
- No new dependency, runtime or dev.
- The kit knows nothing of mmerp: no i18n, router, React Hook Form, TanStack Query, API client or domain term (ESLint already bans the imports). Every user-facing text, aria label included, is a prop. `i18next/no-literal-string` runs on everything but `*.stories.tsx` and `*.test.tsx`.
- Colour literals only in `src/tokens/` (`scripts/check-colors.sh`).
- Look reference: today's `web/src/shared/ui` CSS modules and `theme.ts`, translated `--mantine-*` → `--mmerp-*` (`--mantine-spacing-x` → `--mmerp-space-x`, `--mantine-color-default-border` → `--mmerp-color-border`, `--mantine-color-dimmed` → `--mmerp-color-dimmed`, `--mantine-color-default-hover` → `--mmerp-color-gray-light`, `--mantine-primary-color-filled` → `--mmerp-color-primary-6`, `--mantine-color-body` → `--mmerp-color-body`, `--mmerp-elevation-surface*` → `--mmerp-color-elevation-surface*`, `--mmerp-table-header` → `--mmerp-color-table-header`, `--mmerp-shadow-footer` stays).
- Breakpoints in CSS are written as `64em` (drawer/cards) and `80em` (six-column forms), `48em` (two-column lists); `useMediaQuery` uses `breakpoint` from `tokens.ts`.
- Every component: spreads rest props on its root, merges `className` with `cx`, takes `ref` as a prop. Control roots carry `data-mmerp-control`.
- Stories: one file per component beside it. Titles: primitives keep their folder (`Data display/Table`…); blocks use `Blocks/<Name>`; screens `Screens/Employees`. Interactive behaviour gets `play` functions (`storybook/test`). a11y runs as `error`.
- Everything public is exported from `src/index.ts`.
- Verify every task with, from `packages/ui`: `pnpm typecheck && pnpm lint && pnpm test`. Task 11 also runs `pnpm build-storybook`.

## Review Focus

1. **Long text.** A 200-character title, employee name or chosen filter value ellipsises inside its box (`PageHeader` title one line, `DataTable` first column one line, `FilterChip` max 320px) and never widens the page. Tests: Task 6 (`PageHeader` long title), Task 9 (`FilterChip` long value), Task 10 (`DataTable` long first cell).
2. **Narrow screens.** Below 1024px: `DataTable` renders cards, `ActionBar` is fixed at the bottom with content kept clear of it, `AppShell` rail mode is ignored (drawer instead); below 768px (48em), `PageHeader` stacks its actions under the title. Tests: Task 4, Task 6, Task 7, Task 10 (unit tests with `page.viewport(800, 900)`).
3. **Empty and zero values.** A `DataTable` cell rendering `''`, `null` or `undefined` shows a dimmed "–" (but a total row leaves it blank); a `number` cell rendering `0` shows "–". Test: Task 10.
4. **Keyboard on tables.** Sort header toggles with Enter/Space and sets `aria-sort`; a resize handle moves 16px per arrow key, clamped to 48–800px, and reports once per key; select-all shows indeterminate when some rows are selected. Tests: Task 3, Task 10.
5. **Focus and overlays in list blocks.** `FilterChip` opens on click and Enter, Escape closes it and returns focus to the chip; with `searchable`, focus lands in the search box. Test: Task 5 (`ComboboxButton`), Task 9.

---

### Task 1: ADR-0037 amendment and design pointers

**Files:**
- Modify: `docs/adr/0037-own-ui-kit-on-base-ui.md` (Decision and Consequences)
- Modify: `docs/superpowers/specs/2026-10-20-own-ui-kit-design.md` (Goals 2–3, the anti-goal about a parallel set, "Direction so far")

- [ ] **Step 1:** In ADR-0037's Decision, replace the bullet "`@mmerp/ui` knows nothing of mmerp … built on `@mmerp/ui`." with three bullets, keeping status `proposed`:
  - the unchanged "knows nothing of mmerp" rule;
  - the look/behaviour split from the spec's "The boundary, amended" (look and layout in the kit: texts as props, links through `render`, transient UI state and browser APIs allowed, no fetching, persistence, form state or navigation; behaviour in `shared/ui`; one look, one home), stated as a rule, not a list;
  - the area import rule from the spec ("What areas may import"), with its two named groups.
  Add a line to the header: `Amended 2026-10-21: layout blocks move into the kit (see docs/superpowers/specs/2026-10-21-ui-kit-layout-blocks-design.md).` Add one Consequences bullet: the kit grows by the layout blocks; Storybook can show whole screens.
- [ ] **Step 2:** In the 2026-10-20 design, append to Goals 2 and 3, to the anti-goal "A second, parallel set of components…", and to the "`@mmerp/ui` holds tokens and primitives only…" line of "Direction so far": ` Amended 2026-10-21: see 2026-10-21-ui-kit-layout-blocks-design.md.`
- [ ] **Step 3: Commit** `docs(ui): amend ADR-0037 with the look/behaviour boundary`.

### Task 2: Small primitive fixes (A1, A5, A6, A7, A10)

**Files:**
- Modify: `src/primitives/typography/Anchor.module.css`, `Anchor.stories.tsx`
- Modify: `src/primitives/layout/Group.tsx`, `Group.module.css`, `Stack.tsx`, `Stack.module.css`, their stories
- Modify: `src/primitives/layout/Grid.tsx`, `Grid.module.css`, `Grid.stories.tsx`
- Modify: `src/tokens/tokens.ts`, `src/tokens/render.ts`, `src/tokens/render.test.ts`, `src/tokens/tokens.css` (regenerated), `src/primitives/data-display/Avatar.tsx`, `Avatar.stories.tsx`
- Modify: `src/primitives/input/TextInput.stories.tsx`

**Interfaces — Produces:**
- `GroupProps` and `StackProps` gain `shrink?: boolean` (root gets `min-width: 0`) and `fill?: boolean` (root gets `flex: 1 1 0%; min-width: 0`). Attributes `data-shrink`, `data-fill`.
- `GridColProps['span']` becomes `number | 'auto' | Partial<Record<'base' | 'sm' | 'md' | 'lg' | 'xl', number | 'auto'>>`. Breakpoints `sm 48em, md 64em, lg 80em, xl 96em`; implement with per-breakpoint custom properties (`--mmerp-span-base`, `--mmerp-span-md`…) read by `@media` rules in `Grid.module.css`, each falling back to the smaller one.
- `tokens.color.decorative`: `{ blue: '#228BE6', teal: '#12B886', violet: '#7950F2', orange: '#FD7E14', pink: '#E64980', cyan: '#15AABF', grape: '#BE4BDB', indigo: '#4C6EF5' }` (comment: decorative, never meaning). `render.ts` emits `--mmerp-color-<name>-light: <hex>1A;` and `--mmerp-color-<name>-light-color: <hex>;` for each. Export `type DecorativeColor = keyof typeof tokens.color.decorative`. `AvatarProps['color']` becomes `ColorName | DecorativeColor`.

- [ ] **Step 1: Write failing tests**
  - `render.test.ts`: `expect(css).toContain('--mmerp-color-teal-light: #12B8861A;')` and `toContain('--mmerp-color-teal-light-color: #12B886;')`.
  - `Anchor.stories.tsx`, story `AsButton` (`<Anchor render={<button type="button" />}>+ Bổ sung</Anchor>`), `play`: the button's computed `borderTopWidth` is `'0px'`, `backgroundColor` is `'rgba(0, 0, 0, 0)'`, `paddingLeft` is `'0px'`, and its `fontSize` equals that of a sibling `<p>` text.
  - `Group.stories.tsx`, story `ShrinkTruncates`: a 300px-wide box holding `<Group wrap={false}><Group shrink><Text truncate>{'x'.repeat(200)}</Text></Group><Button>A</Button></Group>`; `play`: the outer group's `scrollWidth` ≤ 300 and the text's `scrollWidth > clientWidth`. Same for `Stack` (`fill` inside a row Group).
  - `Grid.stories.tsx`, story `Responsive`: `<Grid><Grid.Col span={{ base: 12, md: 4 }} data-testid="c" /></Grid>`; a unit test `src/primitives/layout/Grid.test.tsx` renders it at `page.viewport(800, 600)` (col width = grid width) and `page.viewport(1300, 800)` (col width ≈ grid width × 4/12 minus gaps, assert within 2px).
  - `TextInput.stories.tsx`, story `RootClassName`: `play` asserts a `className` passed to `TextInput` lands on the field's root (the element containing the label), not on the `<input>`. This pins A6: width bounds come from `className` on the root; no new prop.
- [ ] **Step 2:** Run `pnpm test` — the new tests fail.
- [ ] **Step 3: Implement.** Anchor CSS: when the root is a `button`, reset `border`, `background`, `padding`, `margin`, `font: inherit`, `cursor: pointer`, keep its colour. Group/Stack props as above. Grid responsive span. Decorative tokens and Avatar colour type; run `pnpm gen`.
- [ ] **Step 4:** `pnpm typecheck && pnpm lint && pnpm test` pass; `git diff --exit-code src/tokens/tokens.css` after `pnpm gen` is clean.
- [ ] **Step 5: Commit** `feat(ui): anchor as button, shrinking groups, responsive grid spans, decorative avatar colours`.

### Task 3: Table cell looks, sort header, sticky first column (A2)

**Files:**
- Modify: `src/primitives/data-display/Table.tsx`, `Table.module.css`, `Table.stories.tsx`

**Interfaces — Produces** (later used by `DataTable`, Task 10):
- `Table.Th` and `Table.Td` gain `look?: 'secondary' | 'numeric' | 'strong'` (`secondary`: tabular, dimmed; `numeric`: tabular, right-aligned; `strong`: weight 600), `sticky?: boolean` (first column: `position: sticky; left: var(--mmerp-table-sticky-left, 0)`, opaque surface background, takes the row's hover tint), `fit?: boolean` (one line, `min-width: max-content` on its child).
- `Table.SortHeader(props: { label: ReactNode; state: 'ascending' | 'descending' | undefined; onSort(): void } & Omit<ThProps, 'children'>)`: a `th` with `aria-sort={state}` holding an `UnstyledButton` (label plus arrow-up / arrow-down / arrows-sort icon from `internal/icons.tsx`, add them there; weight and size inherit the header; text colour when sorted).
- `TableProps` gains `scrolled?: boolean`: shows the sticky column's right edge (`inset -1px 0 0 var(--mmerp-color-border)`) only while true. The caller (DataTable) sets it from the scroll position.
- `Table.Tr` honours `aria-selected="true"` with the primary light tint.
- Header cells: 36px tall, background `--mmerp-color-table-header`, xs semibold dimmed (already partly there); body rows 40px; last column padding-inline-end `--mmerp-space-md`.

- [ ] **Step 1: Write failing stories with `play`:**
  - `Sortable`: three `SortHeader`s driven by local state; click "Name" → its `aria-sort` is `ascending`; press Enter on it → `descending`; the other headers have no `aria-sort`.
  - `CellLooks`: a numeric cell's `textAlign` is `right` and `fontVariantNumeric` contains `tabular-nums`; a secondary cell's colour equals the colour of a probe with `color: var(--mmerp-color-dimmed)`.
  - `StickyFirstColumn`: a 400px scroll box with a 1200px table; after `scrollLeft = 200`, the first cell's `getBoundingClientRect().left` equals the box's left (±1px).
  - `SelectedRow`: a row with `aria-selected="true"` has the background of `var(--mmerp-color-primary-light)`.
- [ ] **Step 2:** `pnpm test` fails on them.
- [ ] **Step 3:** Implement the props above.
- [ ] **Step 4:** Checks pass.
- [ ] **Step 5: Commit** `feat(ui): table sort header, cell looks and sticky first column`.

### Task 4: AppShell rail and full-height main, compact NavLink (A3, A4)

**Files:**
- Modify: `src/primitives/layout/AppShell.tsx`, `AppShell.module.css`, `AppShell.stories.tsx`
- Modify: `src/primitives/navigation/NavLink.tsx`, `NavLink.module.css`, `NavLink.stories.tsx`
- Create: `src/primitives/layout/AppShell.test.tsx`

**Interfaces — Produces:**
- `AppShellProps` gains `navbarCollapsed?: boolean` and `navbarCollapsedWidth?: number` (default 56). Ignored on mobile (below 64em), where the navbar is the drawer.
- `main` becomes `display: flex; flex-direction: column; min-height: 0`, and the root is `height: 100dvh` with `main` the only scrolling area, so a child with `flex: 1; min-height: 0` fills the window below the header (this is what `Page variant="fill"` uses in Task 6).
- `NavLinkProps` gains `compact?: boolean`: 32px rows (`--mmerp-control-height: 2rem`, `padding-block: 0`), label and left section in `--mmerp-color-dimmed` until `active`, section margin-inline-end `--mmerp-space-xs`.

- [ ] **Step 1: Failing tests.**
  - `AppShell.test.tsx`: at `page.viewport(1300, 800)`, `navbarCollapsed` renders the navbar 56px wide; at `page.viewport(800, 900)` with `navbarCollapsed` and `navbarOpened`, the navbar is 240px wide (drawer). A child `<div style={{ flex: 1, minHeight: 0 }}>` of main is `800 − 48` px tall at a 800px-tall viewport.
  - `NavLink.stories.tsx`, `Compact`: `play` asserts height 32px, an inactive link's colour equals `var(--mmerp-color-dimmed)`, an active one does not.
- [ ] **Step 2:** Tests fail.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** Checks pass (the existing AppShell stories still pass a11y).
- [ ] **Step 5: Commit** `feat(ui): app shell rail and full-height main, compact nav links`.

### Task 5: Button.Group and ComboboxButton (A8, A9)

**Files:**
- Modify: `src/primitives/input/Button.tsx`, `Button.module.css`, `Button.stories.tsx`
- Create: `src/primitives/input/ComboboxButton.tsx`, `ComboboxButton.stories.tsx` (styles reuse `options.module.css`)

**Interfaces — Produces** (used by `FilterChip`, Task 9):
- `Button.Group(props: React.ComponentPropsWithRef<'div'>)`: `role="group"`, children joined edge to edge (inner radii 0, borders collapsed by `margin-inline-start: -1px`).
- `ComboboxButton(props: ComboboxButtonProps)` on Base UI `combobox` with the input inside the popup:
  ```ts
  type ComboboxButtonProps = {
    trigger: React.ReactElement // rendered as the Base UI trigger, e.g. a <Button>
    data: SelectItem[]
    value: string | null
    onChange(value: string): void
    nothingFound: React.ReactNode
    searchable?: boolean
    searchLabel?: string // aria-label and placeholder of the search box; required when searchable
    renderOption?: (item: SelectItem) => React.ReactNode
    filter?: (item: SelectItem, query: string) => boolean // default matchesQuery
    popupWidth?: number // default tokens.filterWidth.menu (240)
  }
  ```
  Choosing an item calls `onChange` and closes; the search text resets on close; the chosen item has `aria-selected` and a tint, no check mark.

- [ ] **Step 1: Failing stories with `play`.**
  - `Button.stories.tsx` `Grouped`: two buttons in a group; the second's `getBoundingClientRect().left` equals the first's right − 1 (±0.5).
  - `ComboboxButton.stories.tsx` `Default`: click the trigger → listbox visible; ArrowDown, Enter → `onChange` called with the first value; listbox gone; focus is back on the trigger. `Escape` after opening closes it with focus on the trigger.
  - `Searchable`: open → the search box has focus; type `nguyen` → only "Nguyễn Văn An" remains; type `zzz` → the `nothingFound` text shows.
- [ ] **Step 2:** Tests fail.
- [ ] **Step 3:** Implement; export both from `src/index.ts`.
- [ ] **Step 4:** Checks pass.
- [ ] **Step 5: Commit** `feat(ui): button group and combobox with a button trigger`.

### Task 6: Page and PageHeader (B1, B2)

**Files:**
- Create: `src/blocks/page/Page.tsx`, `Page.module.css`, `Page.stories.tsx`, `PageHeader.tsx`, `PageHeader.module.css`, `PageHeader.stories.tsx`, `PageHeader.test.tsx`

Look source: `web/src/shared/ui/page/Page.tsx`, `Page.module.css`, `ListPage.module.css` (`.frame`, `.content`).

**Interfaces — Produces:**
- `Page(props: { variant?: 'full' | 'centered' | 'detail' | 'fill'; header: React.ReactNode; children: React.ReactNode } & ComponentPropsWithRef<'div'>)`. Padding: inline `md` / `xl` from 64em / `xxl` from 80em, block `lg`; gap `lg` between header and children. `centered` max-width `--mmerp-content-width`, `detail` `--mmerp-detail-width`, both centred with header and content sharing the left edge. `fill`: full width, fills `AppShell` main (`flex: 1; min-height: 0`), children area `min-height: 0; overflow: auto; scrollbar-width: thin`, last child (paging) stays put. Export `Page.Content` for the scrolling area of `fill` (gap `lg`).
- `PageHeader(props: PageHeaderProps)`:
  ```ts
  type PageHeaderProps = {
    title: string
    breadcrumbs?: { label: string; render: React.ReactElement }[] // render: an <a>/<Link> element; label is its text
    breadcrumbsLabel?: string // aria-label of the nav; required with breadcrumbs
    current?: boolean // breadcrumb ends with the title (aria-current)
    back?: React.ReactElement // the back control, built by the caller (see BackButton)
    leading?: React.ReactNode
    status?: React.ReactNode
    description?: React.ReactNode
    actions?: React.ReactNode
    summary?: React.ReactNode
    tabs?: React.ReactNode // a Tabs.List; sits on the header's rule
    plain?: boolean
  }
  ```
  Title `Title order={1}` clamped to one line; status beside it, wrapping under a long title; description `sm` dimmed; below 48em actions stack under the title (column), from 48em in a row with `space-between`. Rule under the header (`border-bottom` and `padding-bottom: md`), none when `plain`, `padding-bottom: 0` with tabs (tabs `margin-top: xs; margin-bottom: -1px`).
- `PageHeader.BackButton(props: { label: string; render?: React.ReactElement } & ActionIconProps)`: round (`radius 1000`), `variant="default"`, arrow-left icon, wrapped in a `Tooltip` with `label`, `aria-label={label}`. History logic is the caller's (`onClick`).

- [ ] **Step 1: Failing tests.**
  - `PageHeader.stories.tsx`: `List` (title, description, two actions), `Record` (breadcrumbs, back, avatar leading, status badge, description, summary placeholder, tabs), `LongTitle` (200 chars). `play` on `LongTitle`: header `scrollWidth` ≤ its `clientWidth`, `h1` `scrollWidth > clientWidth`. `play` on `Record`: the last breadcrumb item has `aria-current="page"`; the back button has the accessible name passed in.
  - `PageHeader.test.tsx`: at `page.viewport(600, 800)` the actions' top is below the title's bottom; at `page.viewport(1300, 800)` they share a row.
  - `Page.stories.tsx`: `Fill` inside an `AppShell` with 60 rows of content and a footer line; `play`: after scrolling `Page.Content` to the bottom, the header's `getBoundingClientRect().top` is unchanged and the footer is fully inside the viewport.
- [ ] **Step 2:** Tests fail.
- [ ] **Step 3:** Implement; export `Page`, `PageHeader`, types.
- [ ] **Step 4:** Checks pass.
- [ ] **Step 5: Commit** `feat(ui): page frame and page header blocks`.

### Task 7: Panel, FieldList, KeyFacts, record layout (B3, B4, B5, B9)

**Files:**
- Create in `src/blocks/page/`: `Panel.tsx`, `Panel.module.css`, `Panel.stories.tsx`, `FieldList.tsx`, `FieldList.module.css`, `FieldList.stories.tsx`, `KeyFacts.tsx`, `KeyFacts.module.css`, `KeyFacts.stories.tsx`, `RecordLayout.tsx`, `RecordLayout.module.css`, `RecordLayout.stories.tsx`, `RecordLayout.test.tsx`
- Modify: `src/primitives/navigation/Tabs.tsx`, `Tabs.module.css` (scroll variant)

Look source: `web/src/shared/ui/page/Activity.tsx` (`Panel`), `FieldList.tsx`/`.module.css`, `KeyFacts.tsx`/`.module.css`, `page/RecordPage.module.css`, `page/DetailPage.module.css`.

**Interfaces — Produces:**
- `Panel(props: { title: string; icon?: React.ReactNode; actions?: React.ReactNode; plain?: boolean; children: React.ReactNode } & ComponentPropsWithRef<'section'>)`: `section` with `aria-label={title}`; framed (border, `radius-md`, padding `md`) or plain (no frame, padding-inline 0); title line min-height control height, title `Text weight={600}`, actions wrap under it on a narrow card; gap `sm`.
- `FieldList(props: { rows: FieldRow[]; stacked?: boolean })`, `type FieldRow = [string, React.ReactNode] | [string, React.ReactNode, 'wide']` (today's tuple, so phase 2 swaps the import only). `dl`; side by side: grid `var(--field-list-label, minmax(8rem, max-content)) minmax(0,1fr)`, gap `xs lg`; stacked: label over value, one column, two from 48em, gap `md lg`, `wide` spans the row. Labels `sm` dimmed (`dt`), values (`dd`, margin 0). Empty values are the caller's (`MissingValue` stays in `shared/ui`).
- `KeyFacts(props: { facts: [string, React.ReactNode][] })` (today's shape): `dl`, two columns, `repeat(auto-fit, minmax(10rem, 1fr))` from 48em, gap `sm`, margin `md 0`; each tile sunken surface, padding `sm md`, `radius-md`; label `xs` dimmed, value weight 500.
- `RecordColumns(props: { main: React.ReactNode; side: React.ReactNode; ratio?: 'record' | 'document' })`: one column below 64em; from 64em `minmax(0,1.6fr) minmax(0,1fr)` (`record`, default) or `minmax(0,62fr) minmax(0,38fr)` (`document`); gap `xl`, `align-items: start`; main column a stack with gap `md`; side column stack gap 0 whose children after the first get `border-top` and radius 0.
- `ActionBar(props: { children: React.ReactNode })`: renders nothing from 64em (CSS `display: none`); below, a bar `position: fixed; inset: auto 0 0 0; z-index: 100`, padding `sm md`, body background, top border, plus a 64px spacer in flow so content stays clear.
- `TabsListProps` gains `scroll?: boolean`: one row, `flex-wrap: nowrap; overflow-x: auto; scrollbar-width: none`, tabs `flex-shrink: 0`. Also `onRule?: boolean`: no bottom border of its own (it sits on `PageHeader`'s rule).

- [ ] **Step 1: Failing tests.**
  - `Panel` stories `Framed`, `Plain`, `WithActions`; `play`: the section is reachable by role `region` with the title as name.
  - `FieldList` stories `SideBySide`, `Stacked` (with one `wide` row); `KeyFacts` story with four facts.
  - `RecordLayout.test.tsx`: at `page.viewport(1300, 900)` `RecordColumns` main width / side width ≈ 1.6 (±0.05) and ≈ 62/38 with `ratio="document"`; at `page.viewport(800, 900)` side is below main; `ActionBar` is not displayed at 1300 and is at the viewport bottom at 800.
  - `Tabs.stories.tsx` `Scroll`: eight long tabs in a 300px box; `play`: list `scrollWidth > clientWidth` and every tab has the same `top`.
- [ ] **Step 2:** Tests fail.
- [ ] **Step 3:** Implement; export `Panel`, `FieldList`, `KeyFacts`, `RecordColumns`, `ActionBar`, `FieldRow`.
- [ ] **Step 4:** Checks pass.
- [ ] **Step 5: Commit** `feat(ui): panel, field list, key facts and record layout blocks`.

### Task 8: Form layout (B6)

**Files:**
- Create in `src/blocks/form/`: `FormSection.tsx`, `FormLayout.module.css`, `FormSection.stories.tsx`, `FormBar.tsx`, `FormBar.stories.tsx`, `FormLayout.test.tsx`

Look source: `web/src/shared/ui/form/Form.tsx` (`FormSection`, `FormFields`, `FormRow`, `InputRow`, `FormActions` bar) and `Form.module.css`.

**Interfaces — Produces:**
- `FormSection(props: { title: string; description?: string; icon?: React.ReactNode; badge?: React.ReactNode; layout?: 'side' | 'wide' | 'compact'; children: React.ReactNode })`: a `fieldset` with `legend` (h2 size, heading weight), `icon` and `badge` (the error count, filled by `shared/ui`) beside it, description `xs` dimmed with margin-top `xxs`. `side` (default): title column 4/12 and fields 8/12 from 80em, stacked below. `wide`/`compact`: title on top. Max width `--mmerp-content-width`; sections parted by a rule with `lg` padding.
- Field grid inside the section (`--mmerp-control-height: var(--mmerp-control-height-lg)`): `side` → one column, six from 80em with children spanning 3, `FormSection.Short` children 2, `FormSection.Row` the whole row; `compact` → two from 48em, four from 80em; `wide` → one column.
- `FormSection.Short`, `FormSection.Row` (wrappers carrying the span), `FormSection.InputRow` (controls level with inputs: `margin-top: auto; height: var(--mmerp-control-height)`, `space-between`).
- `FormFields(props: { children })`: the paired grid for a card's form — two columns from 48em, 40px controls, gap `md`.
- `FormBar(props: { children: React.ReactNode; status?: React.ReactNode })`: sticky footer `bottom: -1px`, padding `sm 0`, body background, top border; `data-stuck` with `--mmerp-shadow-footer` while pinned (`IntersectionObserver`, threshold 1); children right-aligned; `status` left of them. `FormBar.UnsavedDot`: 8px round primary dot, `aria-hidden`.

- [ ] **Step 1: Failing tests.**
  - `FormSection.stories.tsx`: `Side` (the employee "Thông tin chính" fields: full name in a `Row`, code, org unit, hire date, manager, login), `Compact`, `Wide`, `WithBadge`. `play` on `Side`: inputs are 40px tall.
  - `FormLayout.test.tsx` at `page.viewport(1400, 900)`: in `side`, the legend's left < the first input's left (two columns), a normal field is half the field area (±2px), a `Short` a third, a `Row` the whole; at `page.viewport(1000, 900)` the legend is above the fields.
  - `FormBar.stories.tsx` `Sticky`: a 2000px form in a scroll box; `play`: the bar's bottom equals the box's bottom and it has `data-stuck`; scrolled to the end, `data-stuck` is gone.
- [ ] **Step 2:** Tests fail.
- [ ] **Step 3:** Implement; export `FormSection`, `FormFields`, `FormBar`.
- [ ] **Step 4:** Checks pass.
- [ ] **Step 5: Commit** `feat(ui): form section, field grid and sticky form bar blocks`.

### Task 9: List blocks (B7)

**Files:**
- Create in `src/blocks/list/`: `FilterBar.tsx`, `FilterChip.tsx`, `PagingBar.tsx`, `MoreActions.tsx`, `BulkBar.tsx`, `List.module.css`, one story file each, `FilterBar.test.tsx`

Look source: `web/src/shared/ui/page/ListPage.tsx` and `.module.css`, `form/FilterChip.tsx` and `.module.css`, `form/SearchInput.tsx`, `MoreActions.tsx`, `BulkBar.tsx`.

**Interfaces — Consumes:** `Button.Group`, `ComboboxButton` (Task 5); `Pagination`, `Select`, `TextInput`, `Menu`, `Tooltip`, `ActionIcon`.

**Interfaces — Produces:**
- `FilterBar(props: { filters: React.ReactNode; tools?: React.ReactNode; bulk?: React.ReactNode; label: string })`: `role="search"` with `aria-label={label}`; chips wrap; from 64em `tools` at the right end; when `bulk` is given (wide screens) it replaces the whole bar. Filter inputs inside read like the tool buttons (default-border edge, weight 500, hover tint, primary edge on focus/open).
- `FilterBar.Search(props: Omit<TextInputProps, 'type'>)`: `type="search"`, search icon left, full width below 64em, from 64em `flex: 1 1 160px; min-width: 160px; max-width: 280px` (filter-width tokens). Applying on Enter/blur and the "/" shortcut stay in `shared/ui`.
- `FilterChip(props: { label: string; icon: React.ReactNode; data: SelectItem[]; value: string | null; onChange(value: string | null): void; valueText(label: string, value: string): string; clearLabel: string; nothingFound: React.ReactNode; clearable?: boolean; searchable?: boolean; searchLabel?: string; renderOption?: (item: SelectItem) => React.ReactNode })`. Unset: default variant, icon, label, chevron. Set and clearable: light variant, text `valueText(label, chosenLabel)` (with `title`), a second joined button (`Button.Group`) with an X, `aria-label={clearLabel}` and its tooltip. Max width 320px with the text ellipsised.
- `PagingBar(props: { total: number; page: number; pageSize: number; pageSizes: number[]; onPage(page: number): void; onPageSize(size: number): void; totalText: string; pageSizeText(n: number): string; pageSizeLabel: string; labels: PaginationLabels })`: grid `1fr auto 1fr`; `totalText` `sm` dimmed left; `Pagination` centred with round, quiet page buttons (no edge or fill until hover; current keeps its primary fill); the page-size `Select` right-aligned, 130px, quiet the same way. Renders nothing when `total` is 0.
- `MoreActions(props: { label: string; place?: 'row' | 'page'; children: React.ReactNode })`: `Menu.Root` with a `Tooltip`'d `ActionIcon` trigger (dots icon; `page`: default variant, `row`: subtle gray), popup aligned end; renders nothing without children.
- `BulkBar(props: { countText: string; label: string; clearLabel: string; onClear(): void; children: React.ReactNode })`: `role="toolbar"` `aria-label={label}`, count `sm` weight 600, actions, a subtle gray clear button with an X.

- [ ] **Step 1: Failing tests.**
  - `FilterChip.stories.tsx`: `Unset`, `Set`, `LongValue` (a 200-char option label), `Searchable`. `play`: on `Set`, clicking the clear button calls `onChange(null)`; on `LongValue`, the chip is ≤ 320px wide and its text `scrollWidth > clientWidth`; on `Unset`, Enter opens the list, Escape closes it and focus is back on the chip.
  - `PagingBar.stories.tsx`: `play` asserts the pages nav is horizontally centred in the bar (±2px) and choosing "20" calls `onPageSize(20)`.
  - `FilterBar.test.tsx`: at `page.viewport(1300, 800)` tools are at the bar's right end; with `bulk`, filters are absent and the toolbar present.
  - `MoreActions` story `Row`: click → menu opens, item click closes it. `BulkBar` story: clear button calls `onClear`.
- [ ] **Step 2:** Tests fail.
- [ ] **Step 3:** Implement; export all five plus `FilterBar.Search`.
- [ ] **Step 4:** Checks pass.
- [ ] **Step 5: Commit** `feat(ui): filter bar, filter chip, paging bar, more-actions menu and bulk bar`.

### Task 10: DataTable (B8)

**Files:**
- Create in `src/blocks/table/`: `DataTable.tsx`, `DataTable.module.css`, `DataTable.stories.tsx`, `DataTable.test.tsx`, `columns.ts` (pure helpers), `columns.test.ts`

Look source: `web/src/shared/ui/DataTable.tsx` and `.module.css` — port all of it except `react-router` and `react-i18next`.

**Interfaces — Consumes:** `Table` with `SortHeader`, `look`, `sticky`, `fit`, `scrolled` (Task 3); `Checkbox`; `ScrollArea`; `Paper`; `useMediaQuery`, `useElementSize`.

**Interfaces — Produces:**
- Types, kept identical to `web/src/shared/ui/DataTable.tsx` so phase 2 can swap the import: `ColumnRole`, `ColumnKind`, `Column<T>`, `Selection`.
- `DataTableProps<T>` = today's props with `href` replaced and labels added:
  ```ts
  renderLink?: (row: T, children: React.ReactNode) => React.ReactElement // the first cell's link (and the card title's); the whole row clicks it
  onRowClick?: (row: T) => void // as today; with renderLink absent, the first cell becomes a button
  labels: {
    actions: string // aria-label of the menu column
    selectAll: string
    selectRow: string
    resize(column: string): string
  }
  ```
  All other props as today: `columns, rows, rowKey, selection, onResize, menu, sort, onSort, label, group, footer, maxHeight, cards`.
- Row click: clicking outside `a, button, input, [role=menu]` calls `onRowClick`, or else clicks the first cell's link element (so the router link the caller rendered handles navigation; the kit never navigates).
- `columns.ts`: export `grouped`, `bandStart`, `bandWidth`, `looks`, `fits`, `clampWidth` (48–800), ported from today's file.

- [ ] **Step 1: Failing tests.**
  - `columns.test.ts`: `grouped` keeps runs in order and splits on a name change; `clampWidth(10) === 48`, `clampWidth(1000) === 800`; `bandWidth` counts consecutive columns of a band.
  - `DataTable.stories.tsx` (sample employees, 5 columns as in the employee list): `Basic`, `Sortable`, `Selectable`, `Resizable`, `Banded`, `GroupedWithTotals`, `Footer`, `RowMenu`, `Cards`, `LongFirstCell`, `EmptyCells`. `play`:
    - `Sortable`: clicking "Họ tên" calls `onSort('full_name')`; with `sort="full_name"` clicking calls `onSort('-full_name')`.
    - `Selectable`: selecting one row makes select-all `indeterminate`; select-all selects every key; clicking again clears them.
    - `Resizable`: focus the "Mã" handle, ArrowRight → `onResize('code', width + 16)` called once; the handle has `role="separator"` and the passed `resize('Mã')` label.
    - `EmptyCells`: a text cell rendering `''` and a number cell rendering `0` both show "–"; in the footer row an empty cell is blank.
    - `LongFirstCell`: the first cell stays one line (`scrollHeight` ≤ 41px).
    - row click on a non-link cell triggers the first cell's link `onClick` (spy on the rendered `<a>`).
  - `DataTable.test.tsx`: at `page.viewport(800, 900)` the table renders `role="list"` cards with the title column as the card title and `status` columns at its right; at `page.viewport(1300, 900)` a `table`.
- [ ] **Step 2:** Tests fail.
- [ ] **Step 3:** Implement; export `DataTable` and its types.
- [ ] **Step 4:** Checks pass.
- [ ] **Step 5: Commit** `feat(ui): data table block`.

### Task 11: Screen stories replace the spike

**Files:**
- Create: `src/screens/Employees.stories.tsx`, `src/screens/samples.ts`, `src/screens/screens.test.ts`
- Delete: `src/demo/` (untracked spike: `rm -r`, nothing to stage)

**Interfaces — Consumes:** everything exported by Tasks 2–10.

- [ ] **Step 1: Write `screens.test.ts` (fails until the stories exist):** using `import.meta.glob('./*.stories.tsx', { query: '?raw', import: 'default', eager: true })`, assert at least one file, and that none contains `.module.css`, `style=` or `style:`; and that every import specifier is `@storybook/*`, `storybook/*`, `react`, `@tabler/icons-react`, `../index.ts` or `./samples.ts`.
- [ ] **Step 2: Write `Employees.stories.tsx`**, title `Screens/Employees`, `parameters: { layout: 'fullscreen' }`, stories `List`, `Record`, `New`, one `App` with local route state as in the spike. Content and layout follow the real screens (`web/src/hrm/pages/EmployeesPage.tsx`, `EmployeePage.tsx`, `NewEmployeePage.tsx`, `components/EmployeeForm.tsx`, `EmployeeProfile.tsx`, `EmployeeSummary.tsx`, `EmployeeDigest.tsx`, `web/src/app/AppLayout.tsx`): shell with compact nav, groups and rail toggle; list as `Page variant="fill"` with `PageHeader`, `FilterBar` (`Search`, status `FilterChip`, clear button), `DataTable` with selection, row menu and sort, `PagingBar`; record as `Page variant="detail"` with breadcrumbs, back, avatar, status, `KeyFacts`, scrolling tabs on the rule, `RecordColumns` holding three `Panel`s with `FieldList` (one switching to `FormFields` on Edit) and three plain side panels; new form as `Page variant="centered"` with three `FormSection`s and a `FormBar`. Sample data in `samples.ts` (reuse the spike's).
- [ ] **Step 3:** `rm -r src/demo`.
- [ ] **Step 4:** `pnpm typecheck && pnpm lint && pnpm test && pnpm build-storybook` pass. Then `pnpm storybook`, open the three stories at 1440×900 and at 800×900, and note for the owner any place that still differs from the running app.
- [ ] **Step 5: Commit** `feat(ui): employee screen stories built from kit blocks only`, then confirm `git diff main --stat -- web Dockerfile Makefile .github/workflows/ci.yml` prints nothing.
