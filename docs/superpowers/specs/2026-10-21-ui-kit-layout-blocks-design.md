# `@mmerp/ui`: layout blocks and the primitive gaps a real screen exposed

Date: 2026-10-21. Status: agreed with the owner in conversation; this document is the description to plan from. Follows [2026-10-20-own-ui-kit-design.md](./2026-10-20-own-ui-kit-design.md) (phase 1, merged in #106) and amends the boundary in [ADR-0037](../../adr/0037-own-ui-kit-on-base-ui.md).

## Why

Phase 1 built every primitive in the parity inventory, each with its stories. Looking through Storybook, the owner felt the kit did not match the web app, the tables above all, and could not tell whether it was ready to replace Mantine.

To find out, a throwaway spike (`packages/ui/src/demo/EmployeeDemo.stories.tsx`, uncommitted) rebuilt three real HRM screens from kit primitives only: the employee list, the employee page and the new-employee form, inside the app shell. Everything the kit could not do was tagged `GAP` in the spike. Two kinds of gap came out:

1. **Primitive gaps.** About ten primitives lack a prop, a variant or a fix that real screens need (table below).
2. **No layout blocks.** Page header, panel, label–value list, key facts, form section, filter bar, paging bar and the data table all had to be rewritten by hand. Under ADR-0037 they belong to `shared/ui`, so Storybook can never show a screen that looks like the app, and the owner cannot judge the kit before phase 2.

The owner wants the kit to be provable on its own: a screen assembled in Storybook from kit parts should look and behave like the same screen in the app. So the owner agreed (2026-10-21) to move the presentational half of `shared/ui` into the kit, and to fix the primitive gaps.

## The boundary, amended

ADR-0037's core rule stays: **`@mmerp/ui` knows nothing of mmerp.** No i18n, router, React Hook Form, TanStack Query, API client, domain term, or user-facing text.

What changes is the sentence "`shared/ui` stays the layer of mmerp components (tables, forms, page templates…)". It splits in two:

- **Look and layout go in `@mmerp/ui`.** A component belongs in the kit when it only arranges and styles what it is given. Every text comes in as a prop (labels, aria labels, "+ Add", "Back"). Every link is rendered through a `render` prop or an `href`, never through a router. It may hold transient UI state (open, hovered, stuck, scrolled, a column's width while dragging) and may use browser APIs (`ResizeObserver`, `IntersectionObserver`, `matchMedia`). It never fetches, persists (no `localStorage` keys of its own), reads form state, or navigates.
- **Behaviour stays in `shared/ui`.** Translations, routing (the back button's `navigate(-1)`, row links), React Hook Form wiring (error counts, dirty state, Ctrl+S, leave confirmation), queries and list state on the URL, saved column widths (`useListView`), confirmations and notifications. A `shared/ui` component built on a kit block fills in its texts and behaviour and adds nothing to its look.

One look, one home: once a block is in the kit, its CSS lives only there. Phase 2 deletes the matching CSS from `shared/ui` rather than keeping a copy.

The ADR states this as a rule, not a closed list. This work moves B1–B9 (below); the other look-only components of `shared/ui` (`Section`, `Tile`, `ModalFooter`, `SidePanel`, `InlineEditField`, `TreeList`, `CellGrid`, `LinesTable`, and the look of `InboxPage` and `DocumentPage`) follow later under the same rule. Until then they stay in `shared/ui` as they are. (owner, 2026-10-21)

**What areas may import.** Areas keep importing behaviour and mmerp texts through `shared/ui`. The area allowlist on `@mmerp/ui` grows to take the blocks that need no mmerp text or behaviour, so that `shared/ui` never re-exports them under another name: `Panel`, `FieldList`, `KeyFacts`, the B9 record layout, and later `Section` and `Tile` when they move. Blocks that need mmerp labels or routing (`PageHeader` with its back button, `DataTable`, `FilterChip`, `PagingBar`, `BulkBar`, `MoreActions`, the form layout with its error badge and footer) are used by areas only through their `shared/ui` component, which fills them in. The ADR amendment records this; the ESLint allowlist itself changes in phase 2, when `web/` starts importing the kit. (owner, 2026-10-21)

## Goals

1. Amend ADR-0037 (still `proposed`) with the boundary above, and update the 2026-10-20 design's statements that say the kit holds primitives only.
2. Close the primitive gaps (section A).
3. Add the layout blocks (section B) to the kit, each with stories, a11y checks and, where interactive, `play` functions.
4. Replace the spike with permanent **screen stories** (`Screens/Employees`: list, record page, new-employee form) built only from kit exports, with **no CSS module of their own and no inline styles**. They are the proof: if a screen story needs local CSS, the kit is still missing something.

## A. Primitive gaps

| # | Primitive | Gap found in the spike | Expected |
| --- | --- | --- | --- |
| A1 | `Anchor` | Rendered as a `<button>` ("+ Add", "Next error") it keeps the browser's button border, background and padding | Resets button styles in its own CSS module |
| A2 | `Table` | Bare HTML table only | Sortable header cell (label, icon by state, `aria-sort`, a button inheriting the header's font and colour); cell looks for secondary (codes, dates: tabular, dimmed), numeric (tabular, right) and strong; sticky first column while scrolling sideways with its edge shown only once scrolled; selectable rows (`aria-selected` style). The frame and scroll container go with `DataTable` (B8) |
| A3 | `AppShell` | No collapsed rail; main cannot let a page fill the window | `navbarCollapsed` (rail width, default 56px); main lays out as a column so a page can take the rest of the window and scroll only its content. The header's inner layout (logo, title, icons right) stays the caller's |
| A4 | `NavLink` | Rows are the 30px control height | A compact variant: 32px rows, label and icon dimmed until active, as the app menu has today |
| A5 | `Group`, `Stack` | Cannot shrink, so a long title cannot truncate inside them | A way to let the element shrink (`min-width: 0`) and grow (`flex: 1`); builder picks the prop names |
| A6 | `TextInput`, `Select`, `Combobox`, `DateInput`, `MonthPickerInput` | No width, so the filter bar wraps them in `div`s | Width bounds on the field's root (the search box: grows from 160px to 280px). Builder picks between `className` on the root and token-valued props |
| A7 | `Grid` | `span` is fixed | `span` also takes a per-breakpoint object (`{ base: 12, lg: 4 }`) |
| A8 | `Button` | No group | `Button.Group`: buttons joined edge to edge (the filter chip and its clear button) |
| A9 | `Combobox` | Input target only | A button-target mode with the search box inside the popup, for a filter chip over a long list |
| A10 | `Avatar` | Only the seven semantic ramps | A set of decorative colours (the app uses eight), added as tokens, documented as carrying no meaning |

Not gaps (checked): field labels (12px, semibold, dimmed) match the app; `UnstyledButton` already resets buttons.

## B. Layout blocks moving into the kit

Each block's look comes from today's `shared/ui` component of the same role (named in brackets). Its props follow the shared component's, minus what is behaviour, plus label props for every text.

| # | Block | From `shared/ui` | Notes |
| --- | --- | --- | --- |
| B1 | `Page` | `page/Page.tsx` (frame, `pageFrame`, `centered`) | Page padding by breakpoint (md / xl / xxl inline, lg block); centred at content width or detail width; a list variant that fills the window (works with A3) |
| B2 | `PageHeader` | `page/Page.tsx` (`PageHeader`, `BackButton`) | Breadcrumbs, back button, leading slot (avatar), title with status, description, actions (stacked under the title on narrow screens), summary slot, tabs on the header's rule, `plain`. The back button is a slot or a `render`-able link; the history logic stays in `shared/ui` |
| B3 | `Panel` | `page/Activity.tsx` (`Panel`) | Titled region with icon and actions on the title line; framed and plain variants; plain panels in a column are parted by rules |
| B4 | `FieldList` | `FieldList.tsx` | Side-by-side and stacked layouts, wide rows. `MissingValue` (texts, router link) stays in `shared/ui` |
| B5 | `KeyFacts` | `KeyFacts.tsx` | Sunken tiles, label over value |
| B6 | Form layout | `form/Form.tsx` (`FormSection`, `FormFields`, `FormRow`, `InputRow`, the footer bar) | Section with title column and six-column field grid (short, half, full spans), the `wide` and `compact` variants, the paired grid used in a card, 40px controls inside; a section title slot for the error-count badge; the sticky footer bar with its "stuck" shadow and an unsaved-dot slot. Error counting, dirty state and submit stay in `shared/ui` |
| B7 | List blocks | `page/ListPage.tsx` (filter row, paging), `form/FilterChip.tsx`, `form/SearchInput.tsx` (look only), `MoreActions.tsx` | Filter bar (chips wrap, tools at the right end from 1024px, a bulk slot replacing it); `FilterChip` built on A8 and A9 (unset: name and chevron; set: "Name: value", tinted, its own clear button; texts as props); `PagingBar` (count left, pages centred, quiet page-size select right, round quiet page buttons); `MoreActions` (the "…" menu, page and row looks, its label as a prop); `BulkBar` (`BulkBar.tsx`: the toolbar that takes the filter bar's place while rows are selected; count text, toolbar label and clear label as props) |
| B9 | Record layout | `page/RecordPage.module.css`, `page/DetailPage.module.css` | The two-column body (main and side; 1.6fr/1fr for a record page, 62fr/38fr for a document page, one column below 1024px, side parts parted by rules); the record tab row that scrolls sideways instead of wrapping (a `Tabs.List` variant); the action bar fixed to the bottom of the screen below 1024px, with the spacer that keeps content clear of it. Which tab is open, and on the URL, stays in `shared/ui` (my call, from the builder check: without it the record screen story would need local CSS) |
| B8 | `DataTable` | `DataTable.tsx` | Everything it does today except routing and i18n: columns with kind and role, sort headers, selection with select-all, resize handles (live width while dragging; the final width is reported through a callback, saving it stays in `shared/ui`), bands, groups with totals, footer row, row menu slot, sticky header and footer inside a capped height, the card list below 1024px. The first cell's link and the row click take a `renderLink`/`onRowClick`-style API instead of `react-router`; every label comes in as a prop |

Stays in `shared/ui`, unchanged by this work: `ListPage`, `RecordPage`, `DetailPage`, `DocumentPage`, `InboxPage` (they wire URL state, queries and tabs on the URL, and become compositions of B1–B9 in phase 2; the look-only parts of their CSS are B1, B7 and B9), `Form` and `FormActions` behaviour, `useListView`, `MissingValue`, `confirm`, `notify`, `QueryView`, `states`, `StatusBadge`/`DocumentStatus` tones, `avatar.tsx`'s name hashing.

## Non-goals

- No change to `web/`, the Dockerfile, the Makefile or `ci.yml`. The phase-1 rule holds: `make lint/test/e2e/image` never see the kit. Only `packages/ui` and the docs named here change.
- No phase-2 migration, and no decision on when phase 2 starts.
- No i18n, router, form library or data fetching in the kit, even "optional".
- No new runtime dependency.
- No pixel-perfect comparison tooling or screenshot tests.

## Anti-goals (failure even if it works)

- A kit component with literal user-facing text, or importing anything from `web/`.
- A block whose look ends up defined in two places once phase 2 runs (the kit's CSS plus a `shared/ui` override restyling it).
- A screen story that needs its own CSS or inline styles to look right: that hides a gap instead of closing it.
- A block that only fits the employee screens: blocks follow the shared components' full prop surface (bands, groups, totals, `wide`/`compact` sections), not just what the three screen stories use.

## Constraints

- Everything from the phase-1 design still applies: CSS modules plus `--mmerp-*` variables, the colour-literal check, `i18next/no-literal-string` (stories exempt), a11y addon as `error`, the `ui.yml` workflow green, tokens regenerated with no diff.
- Today's look is the reference: values come from `shared/ui`'s CSS modules and `theme.ts` as they are now, translated from `--mantine-*` to `--mmerp-*`.
- Behaviour that is accessibility, not mmerp logic, belongs in the kit: `aria-sort`, the resize handle's `role="separator"` and arrow keys, select-all's indeterminate state, focus moving into a panel's form when it opens if the block offers that.
- The repo workflow: one branch (`feat/ui-kit-layout-blocks`, off `main`), one commit per ticket, one PR into `main`; GitHub issues for the spec and each ticket are created when the plan is written.

## Tickets (suggested order)

1. **ADR and docs.** Amend ADR-0037 with the boundary and the area import rule; update the 2026-10-20 design's "knows nothing" and "presets layer" statements to point here.
2. **Primitive gaps A1–A10**, each with stories (and `play` functions for A2 sorting/selection, A3 collapse, A9).
3. **Page blocks**: B1 `Page`, B2 `PageHeader`, B3 `Panel`, B4 `FieldList`, B5 `KeyFacts`, B9 record layout.
4. **Form layout**: B6.
5. **List blocks**: B7.
6. **`DataTable`**: B8.
7. **Screen stories**: `Screens/Employees` (list, record page, new form) from kit exports only; delete the spike.

## Success

- The three screen stories exist with no CSS module and no `style` prop of their own, and the owner, putting them beside the running app, accepts them as the same screens.
- Every new block and changed primitive has stories; the `ui.yml` workflow passes (tsc, ESLint, colour check, token diff, Vitest, story tests with a11y, Storybook build).
- `git diff main -- web Dockerfile Makefile .github/workflows/ci.yml` is empty.

## Decisions so far

- Move the presentational half of `shared/ui` into the kit; the "knows nothing of mmerp" rule stays. (owner, 2026-10-21, on my recommendation)
- Do the work on its own branch. (owner, 2026-10-21)
- Scope is B1–B9 plus `BulkBar`; the other look-only `shared/ui` components follow later under the same rule, written in the ADR as a rule rather than a list. (owner, 2026-10-21, on my recommendation)
- Areas may import look-only blocks straight from `@mmerp/ui`; blocks needing mmerp texts or behaviour reach areas through `shared/ui`. (owner, 2026-10-21, on my recommendation)
- The two-column record body, the sideways-scrolling record tabs and the narrow-screen action bar join the scope as B9. (my call, after the builder check)
- The test for kit vs `shared/ui` is the look/behaviour split written above, including "no persistence, no navigation, no form state, no fetching". (my call)
- `DataTable` moves whole, minus routing and i18n, rather than leaving its rendering in `shared/ui`: the table was the owner's first complaint. (my call)
- Page templates (`ListPage`, `RecordPage`, …) stay in `shared/ui`: they wire URL state and queries. (my call)
- Screen stories are kept permanently as the proof, replacing the spike, and must use no local CSS. (my call)
- `web/` stays untouched, as in phase 1. (my call, carrying the phase-1 rule forward)

## Left to the builder

- Prop names and compound shapes (`PageHeader` slots vs props, `DataTable` column type, `Group` shrink/grow names, A6's width API).
- Splitting a ticket that grows too large to review.
- Whether `Page`'s fill-window list variant is a prop of `Page` or a separate block.
- Sample content of the screen stories (any plausible Vietnamese HRM data; the spike's is fine to reuse).
