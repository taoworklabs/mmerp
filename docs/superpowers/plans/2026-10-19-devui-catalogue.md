# /dev/ui layered catalogue Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure `/dev/ui` into six layers (Start here, Foundations, Mantine used directly, mmerp components, Patterns, Page templates). Every component page follows one fixed anatomy.

**Architecture:** A pure data module `catalogue.ts` lists every entry (layer, path, name or label key, import source, related entries), and a Vitest test checks it against the ESLint Mantine allowlist and the vi translations. `index.tsx` builds the sidebar and routes from it. A `DocPage` frame renders the anatomy in a fixed order from the entry plus the page's own examples. The page files are regrouped by layer, and the existing examples move into them rather than being rewritten.

**Tech Stack:** React 19, Mantine 9 (`TableOfContents` for "on this page"), react-router, react-i18next, Vitest (node environment, no DOM).

**Spec:** `docs/superpowers/specs/2026-10-19-devui-catalogue-design.md`

## Global Constraints

- All work stays in `web/src/app/devui`, apart from the i18n files (`web/src/shared/i18n/locales/vi.json`, `en.json`, flat keys `shared.devui.*`, same keys in both) and the doc lines in `ui.md` line 9 and `.claude/skills/mmerp-ui/SKILL.md` line 14.
- No JSX literal text (eslint-plugin-i18next). Component names, import paths and code snippets live as data in `catalogue.ts` or in `const` objects, as `basics.tsx`'s `sample` already does.
- No colour literals (`scripts/check-colors.sh`), no `style` prop. Use CSS modules with theme variables: Do border `var(--mantine-color-success-6)`, Don't border `var(--mantine-color-danger-6)`.
- Layer 3 is exactly the `allowImportNames` list in `web/eslint.config.js` (today `Anchor, Button, Divider, Group, List, Loader, Skeleton, Stack, Text, Tooltip`). No Mantine page for anything outside it.
- No rule prose copied from the `mmerp-ui` skill. Each page gets a one-line description, a one-line "use when" and a one-line "don't use when"; nothing longer.
- Order inside `mantine`, `mmerp` and `templates`: alphabetical by name, case-insensitive. `foundations` and `patterns` keep the fixed order given below (plan call: these are not name lists).
- Commit per task, Conventional Commits, scope `web`, ending with `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`. Stage explicit paths only.
- Verification command for every task (from repo root): `pnpm -C web typecheck && pnpm -C web lint && node web/scripts/check-i18n.mjs && pnpm -C web test`. Expected: all pass.

## Review Focus

1. **A sample link inside a page still points to an old path** (`/dev/ui/links`, `/dev/ui/tiles`, …) and lands on the catch-all redirect. Covered by the `every /dev/ui link in the catalogue resolves` test (Task 2).
2. **The ESLint allowlist changes and layer 3 silently drifts.** Covered by `mantine layer covers exactly the area allowlist` (Task 2).
3. **An entry's title, description or use/avoid key is missing in vi**, so the page shows a raw key. Covered by `every entry has its i18n keys` (Task 2). `check-i18n.mjs` covers en.
4. **`/dev/ui` with no path, or an old bookmarked path**, must land on Start here and not on a blank page. Covered by a manual check (Task 2, Step 6).
5. **Narrow screens (375px):** the sidebar drawer still opens, the table of contents is hidden below `lg`, and Do/Don't stacks into one column. Covered by a manual check (Task 6, Step 4).

---

### Task 1: The page anatomy frame

**Files:**
- Create: `web/src/app/devui/DocPage.tsx`
- Modify: `web/src/app/devui/DevUi.module.css` (add `.tag`, `.do`, `.dont`, `.toc`)
- Modify: `web/src/shared/i18n/locales/vi.json`, `en.json`

**Interfaces:**
- Consumes: `Entry`, `entryByPath`, `Path` from `catalogue.ts` (Task 2). To avoid a cycle in task order, Task 1 defines the minimal `catalogue.ts` skeleton (types plus `entries` holding only `mantine/button`), and Task 2 fills it in.
- Produces:
  - `DocPage({ path, example, variants, doDont, code }: { path: Path; example: ReactNode; variants?: ReactNode; doDont?: ReactNode; code?: string })`. It renders, in this order: header (title = `entry.name ?? t(entry.label)`, the source tag, description `t('shared.devui.about_' + key(entry))`), then the sections Example, Usage (`use_`/`avoid_` keys), Variants, Do/Don't, Code, Related. A section whose content is absent is not rendered. Code shows `import { <name> } from '<source>'` when the entry has `source`, followed by `code` when given. Related lists `entry.related` as router links titled by each target's name or label; its heading is `sec_related` for components and `sec_uses` for patterns and templates. On `lg` and wider, a Mantine `TableOfContents` sits beside the content, fed by `scrollSpyOptions={{ selector: '[data-devui-section]' }}`.
  - `DoDont({ good, bad }: { good: ReactNode; bad: ReactNode })`: two columns (one below `sm`), green or red top border, labelled `t('shared.devui.do')` / `t('shared.devui.dont')`.
  - `key(entry: Entry): string`: the last path segment with `-` replaced by `_` (`mmerp/data-table` → `data_table`).
  - `Case` stays exported from `Demo.tsx` unchanged.
- i18n keys added: `sec_example`, `sec_usage`, `sec_variants`, `sec_do_dont`, `sec_code`, `sec_related`, `sec_uses`, `on_this_page`, `do`, `dont`, `source_mantine` ("Mantine · dùng thẳng" / "Mantine · used directly"), `source_mmerp` ("mmerp"), `mantine_docs` (link label), plus `about_button`, `use_button`, `avoid_button`.

- [ ] **Step 1:** Create the `catalogue.ts` skeleton with the `Layer`, `Entry` and `Path` types and `entryByPath` from Task 2 and a single entry `{ layer: 'mantine', path: 'mantine/button', name: 'Button', covers: ['Button'] }`.
- [ ] **Step 2:** Implement `DocPage`, `DoDont` and `key` in `DocPage.tsx`. Section headings are `Title order={2}` elements with `id={section}` and `data-devui-section`. The frame wraps today's `Demo` look: a centred `Page` with breadcrumbs to `/dev/ui`. Mantine pages add an `Anchor` to `https://mantine.dev/core/<name lowercased>/` after the description.
- [ ] **Step 3:** Rewrite `Buttons` in `basics.tsx` as `DocPage path="mantine/button"`. `example` is the primary button. `variants` are today's four `Case`s. `doDont` is good = one primary + one default button, bad = two primary buttons side by side. `code` is the convention string `<Button>  // primary\n<Button variant="default">  // secondary\n<Button variant="subtle" color="gray">  // cancel beside a primary\n<Button color="danger">  // destructive, after a confirm`.
- [ ] **Step 4:** Run the verification command. Expected: pass. Then `pnpm -C web dev --open /dev/ui/buttons` (the old route still serves `Buttons`). Expected: the six sections in order, and the TOC tracks scrolling at ≥1200px.
- [ ] **Step 5:** Commit: `feat(web): /dev/ui page frame with a fixed anatomy`.

### Task 2: The catalogue, its checks, and the layered sidebar

**Files:**
- Modify: `web/src/app/devui/catalogue.ts` (full entry list)
- Create: `web/src/app/devui/catalogue.test.ts`
- Create: `web/src/app/devui/start.tsx` (`Overview`)
- Modify: `web/src/app/devui/index.tsx`
- Modify: i18n files

**Interfaces:**
- Produces, in `catalogue.ts`:
  ```ts
  export type Layer = 'start' | 'foundations' | 'mantine' | 'mmerp' | 'patterns' | 'templates'
  export const layers: Layer[] // in that order
  export type Entry = { layer: Layer; path: string; name?: string; label?: string; source?: string; covers?: string[]; related?: string[] }
  export const entries = [...] as const satisfies readonly Entry[]
  export type Path = (typeof entries)[number]['path']
  export const entryByPath: (path: Path) => Entry
  ```
  `path` is `<layer>/<slug>` and the route is `/dev/ui/<path>`. Entries with `name` show it as-is; the others show `t(label)`.
- In `index.tsx`: `const pages: Record<Path, ComponentType>`, so tsc fails on a missing or extra page. Sidebar: per layer, a heading `"<n> · " + t('shared.devui.layer_<layer>')` with `t('shared.devui.layer_<layer>_note')` below it in dimmed `xs` text, then its entries. The index route and `*` redirect to `start/overview`. Delete `groups` and `Usage`; the template strip moves to Task 6 and stays as-is until then, keyed off `entry.layer === 'templates'`.

The full entry list (slug → current material it starts from; the page is converted in the named task):

| Layer | Entries |
| --- | --- |
| start | `overview` (new, label key) |
| foundations | `principles`, `colour`, `typography`, `spacing`, `surfaces`, `icons` (today's, label keys, this order) |
| mantine | `anchor` (Links, minus breadcrumbs), `button`, `layout-text` name `Group · Stack · Divider · List · Text`, covers those five (new), `loader` (new), `skeleton` (new), `tooltip` (Overlays' tooltip case) |
| mmerp | `activity-feed` `@/shared/document`; `approval-panel` `@/shared/document`; `attachment-panel` `@/shared/document` (name `useAttachmentSection`); `bulk-bar` `@/shared/ui/BulkBar`; `cell-grid` `@/shared/ui/CellGrid`; `data-table` `@/shared/ui/DataTable`; `document-preview` `@/shared/document`; `document-status` `@/shared/document`; `editable-panel` `@/shared/ui/form`; `field-list` `@/shared/ui/FieldList`; `filters` name `Filters` `@/shared/ui/form`; `form` name `Form · fields` `@/shared/ui/form`; `form-modal` `@/shared/ui/page`; `inline-edit-field` `@/shared/ui/InlineEditField`; `key-facts` `@/shared/ui/KeyFacts`; `lines-table` `@/shared/ui/form`; `more-actions` name `MoreActions` `@/shared/ui/MoreActions`; `notice` `@/shared/ui/Notice`; `notify` `@/shared/ui/notify`; `panel` `@/shared/ui/page`; `person-avatar` `@/shared/ui/avatar`; `query-view` `@/shared/ui/QueryView`; `section` `@/shared/ui/Section`; `side-panel` `@/shared/ui/SidePanel`; `states` name `States` `@/shared/ui/states`; `status-badge` `@/shared/ui/StatusBadge`; `tiles` name `Tiles` `@/shared/ui/Tile`; `tree-list` `@/shared/ui/TreeList`; `use-action` `@/shared/ui/useAction`; `use-confirm` `@/shared/ui/confirm` |
| patterns | `filter-search`, `create-edit`, `confirm-delete`, `states`, `bulk`, `document-approval` (label keys, this order) |
| templates | `auth` AuthPage, `document` DocumentPage, `inbox` InboxPage, `list` ListPage, `page` Page, `record` RecordPage (source `@/shared/ui/page`) |

Fill in `related` while converting each page (Tasks 3–6); Task 2 leaves it empty. Until a page is converted, its route points at today's component (for example `mmerp/status-badge` → `Badges`). `IconButtons`, `TabsDemo` and `Overlays`' popover case get no route; delete them in Task 3.

- [ ] **Step 1: Write the failing tests** in `catalogue.test.ts` (read files with `node:fs`, paths relative to `import.meta.url`):
  - `paths are unique and start with their layer`: every `e.path.startsWith(e.layer + '/')`, and `new Set(paths).size === entries.length`.
  - `mantine layer covers exactly the area allowlist`: parse `allowImportNames: [...]` from `web/eslint.config.js` with a regex. Expect `entries.filter(e => e.layer === 'mantine').flatMap(e => e.covers ?? [e.name])` sorted to equal the allowlist sorted.
  - `every mmerp entry names its import source`: `source` matches `/^@\/shared\//`.
  - `component layers are alphabetical`: for `mantine`, `mmerp` and `templates`, the names equal their copy sorted with `localeCompare(…, undefined, { sensitivity: 'base' })`.
  - `every entry has its i18n keys`: load `vi.json`. Every `label` exists. For layers `mantine | mmerp | patterns | templates`, `shared.devui.about_<key>`, `use_<key>` and `avoid_<key>` exist (`key` as in Task 1). `layer_<l>` and `layer_<l>_note` exist for every layer.
  - `related points at real entries`: every `related` item is some entry's `path`.
  - `every /dev/ui link in the catalogue resolves`: in each `web/src/app/devui/*.tsx`, every match of `/['"\`]\/dev\/ui\/([a-z0-9/-]+)/` is an entry path. (`'/dev/ui'` alone is allowed.)
- [ ] **Step 2:** Run `pnpm -C web test -- catalogue`. Expected: FAIL (entries incomplete, keys missing).
- [ ] **Step 3:** Fill in `entries`, rewire `index.tsx`, write `Overview` in `start.tsx`, add the i18n keys, and fix old sample links in the page files to the new paths. `Overview` has one short `Section` per layer: the layer note plus links to its first entries. Its last section points to the `mmerp-ui` skill and `ui.md` for the full rules, naming them as text. Until their pages are converted, the about/use/avoid keys may hold the old `_hint` text.
- [ ] **Step 4:** Run `pnpm -C web test -- catalogue`. Expected: PASS.
- [ ] **Step 5:** Run the verification command. Expected: pass.
- [ ] **Step 6:** Run `pnpm -C web dev`. Open `/dev/ui`, `/dev/ui/buttons` and `/dev/ui/nope`. Expected: each lands on Start here, and the sidebar shows six numbered layers with notes.
- [ ] **Step 7:** Commit: `feat(web): /dev/ui sidebar in six layers from one catalogue`.

### Task 3: Mantine layer and mmerp display components

**Files:**
- Create: `web/src/app/devui/mantine.tsx` (`AnchorDoc`, `ButtonDoc` moved from `basics.tsx`, `LayoutTextDoc`, `LoaderDoc`, `SkeletonDoc`, `TooltipDoc`)
- Create: `web/src/app/devui/display.tsx` (`DataTableDoc`, `CellGridDoc`, `FieldListDoc`, `KeyFactsDoc`, `TreeListDoc`, `StatusBadgeDoc`, `PersonAvatarDoc`, `TilesDoc`, `SectionDoc`, `PanelDoc`)
- Delete: `web/src/app/devui/basics.tsx`, `web/src/app/devui/data.tsx` (once their material has moved)
- Modify: `index.tsx` `pages`, i18n files

Each page is a `DocPage`. Its `variants` are today's `Case`s for that component. `related` is set in `catalogue.ts`. Specific moves:

- `DataTableDoc` takes today's `Table` and `GroupedTable`, plus a `Case` for the column tools from `useListView` (from `Bulk`, without selection). Related: `list`, `filter-search`, `bulk`, `query-view`.
- `StatusBadgeDoc` takes the record-status case. The count badge is dropped (Mantine `Badge` is outside the allowlist). `DocumentStatus` goes to its own page in Task 4.
- `TreeListDoc` keeps today's `Tree` and names `OrgUnitName` in its `code`.
- `PanelDoc` takes `Cards`' panel case. `KeyFactsDoc` takes the key-facts case. `RecordColumns` moves to the `record` template in Task 6.
- `LayoutTextDoc` has one `Case` each for `Group`/`Stack` gaps, `Divider`, `List`, and `Text` sizes and `c="dimmed"`. Its `code` gives the token-only convention (`gap="xs"`, never numbers).
- `SkeletonDoc` has a do/don't with good = `ContentSkeleton` (from `@/shared/ui/states`), bad = a hand-built stack of `Skeleton`s for a whole region. It relates to `mmerp/states`.
- `TooltipDoc` uses a `Tooltip` on a `Button` with an icon. `AnchorDoc` takes `Links` minus breadcrumbs, with `sample` kept.
- Do/Don't, at least on `ButtonDoc`, `DataTableDoc`, `StatusBadgeDoc` and `SkeletonDoc`. Every Don't must pass lint. A mistake lint would block is written as one line of text (`t('shared.devui.dont_<key>')`).

- [ ] **Step 1:** Add the about/use/avoid/related data and i18n for these 16 entries.
- [ ] **Step 2:** Implement the pages, wire them in `pages`, and delete the emptied files.
- [ ] **Step 3:** Run the verification command (the catalogue tests included). Expected: pass.
- [ ] **Step 4:** Open each new route in `pnpm -C web dev`. Expected: the anatomy is in order, with no raw `shared.devui.*` keys on screen.
- [ ] **Step 5:** Commit: `feat(web): /dev/ui Mantine and display component pages`.

### Task 4: mmerp form, feedback and document components

**Files:**
- Create: `web/src/app/devui/forms.tsx` (`FormDoc`, `FiltersDoc`, `InlineEditFieldDoc`, `LinesTableDoc`, `EditablePanelDoc`, `FormModalDoc`)
- Create: `web/src/app/devui/actions.tsx` (`MoreActionsDoc`, `NoticeDoc`, `NotifyDoc`, `StatesDoc`, `QueryViewDoc`, `UseActionDoc`, `UseConfirmDoc`, `BulkBarDoc`, `SidePanelDoc`)
- Create: `web/src/app/devui/documents.tsx` (`ActivityFeedDoc`, `ApprovalPanelDoc`, `AttachmentPanelDoc`, `DocumentPreviewDoc`, `DocumentStatusDoc`)
- Delete: `input.tsx`, `feedback.tsx`, `blocks.tsx` (once their material has moved; `Headers` moves to Task 6, so keep it until then, or move it into `templates.tsx` now)
- Modify: `index.tsx`, i18n files

Specific moves:

- `FormDoc` takes `Fields`. `FiltersDoc` takes `Filters` (including `DocumentStatusFilter`). `LinesTableDoc` takes `Lines`.
- `FormModalDoc` takes the form-modal and modal-footer cases. `UseConfirmDoc` takes the confirm case.
- `NoticeDoc` takes `Alerts`. `NotifyDoc` takes `ToastsDemo`.
- `QueryViewDoc` shows two static `Case`s (loading, empty). The state switcher moves to the `states` pattern (Task 5).
- `UseActionDoc` takes `Actions`.
- `BulkBarDoc` shows only the bar with a count. The full selection flow is the `bulk` pattern. `SidePanelDoc` opens a `SidePanel` from a button.
- `DocumentStatusDoc` takes the doc-status case from `Badges`. The other document pages take `ActivityDemo` and `DocumentPanels`' cases, inside `SampleDocument`.
- `MoreActionsDoc` takes `Menus`, and its Do/Don't is good = danger item last after a divider, bad = danger item first.

- [ ] **Step 1:** Add data and i18n for these 20 entries.
- [ ] **Step 2:** Implement, wire, and delete the emptied files.
- [ ] **Step 3:** Run the verification command. Expected: pass.
- [ ] **Step 4:** Open each new route. Expected: the anatomy is in order and no raw keys show. The toasts, confirm dialog and side panel open.
- [ ] **Step 5:** Commit: `feat(web): /dev/ui form, feedback and document component pages`.

### Task 5: Patterns

**Files:**
- Create: `web/src/app/devui/patterns.tsx` (`FilterSearchPattern`, `CreateEditPattern`, `ConfirmDeletePattern`, `StatesPattern`, `BulkPattern`, `DocumentApprovalPattern`)
- Modify: `index.tsx`, `catalogue.ts` (`related` = the components each pattern uses), i18n files

Each is a `DocPage` with `example` only, plus `doDont` where a legal mistake exists. Its `about_` key states the problem in one line, and its Related section ("Uses") links the components.

- `filter-search`: `SearchInput` + one `FilterChip` over a `DataTable` of `rows`, filtering in the browser. When nothing matches, the no-results `EmptyState` with its clear-filters button. Related: `filters`, `data-table`, `states`, template `list`.
- `create-edit`: a `Case` with a button that opens a `FormModal` holding a `Form` with two fields, and a `Case` with an `EditablePanel` edited in place. Related: `form`, `form-modal`, `editable-panel`.
- `confirm-delete`: a `MoreActions` with a danger `MenuAction`. Running it goes through `useAction` with `confirm`, and the failure is shown in a `Notice` (today's `Actions` flow). Related: `more-actions`, `use-action`, `use-confirm`, `notice`.
- `states`: today's `QueryViews` switcher moves here unchanged. Related: `query-view`, `states`.
- `bulk`: today's `Bulk` moves here unchanged. Related: `data-table`, `bulk-bar`, `side-panel`.
- `document-approval`: inside `SampleDocument`, `DocumentStatus`, `ApprovalPanel` and `ActivityFeed` sit side by side in `RecordColumns`. If `DocumentActions` renders offline against `app/devui/api.ts`, add it to the header. Otherwise leave a one-line `ponytail:` comment saying it is missing and why. Related: `document-status`, `approval-panel`, `activity-feed`, template `document`.

- [ ] **Step 1:** Add data and i18n.
- [ ] **Step 2:** Implement and wire.
- [ ] **Step 3:** Run the verification command. Expected: pass.
- [ ] **Step 4:** Open the six routes. Expected: each example works offline.
- [ ] **Step 5:** Commit: `feat(web): /dev/ui pattern pages`.

### Task 6: Template notes, the Page template, docs

**Files:**
- Modify: `web/src/app/devui/index.tsx` (`TemplateNote` replaces the old `Usage` strip)
- Modify: `web/src/app/devui/templates.tsx` (`PageTemplate` body becomes `Headers`' three cases; `RecordTemplate` unchanged)
- Delete: whatever is left of `blocks.tsx`, and `Demo`'s `Demo` export if nothing uses it (keep `Case`, `rows`, `Row`)
- Modify: `catalogue.ts` (`related` for templates = the mmerp components each one is built from), i18n files
- Modify: `ui.md` line 9 and `.claude/skills/mmerp-ui/SKILL.md` line 14, to describe the six layers in one sentence and say that the rules stay in the skill

`TemplateNote({ entry }: { entry: Entry })` renders above every `templates` entry, in today's sunken strip. It shows the one-line `about_` (the problem), "Use when" and "Avoid when" (existing `use_`/`avoid_` keys), and "Built from:" followed by `related` as links. Templates keep their full-page render, have no TOC, and are not nested in `DocPage`.

- [ ] **Step 1:** Implement `TemplateNote`, the `PageTemplate` body and the template `related` data, and add the i18n keys (`built_from`, the `about_` keys for templates).
- [ ] **Step 2:** Run the verification command. Expected: pass. `git grep -n "Demo\b" web/src/app/devui` shows no leftover imports of a deleted export.
- [ ] **Step 3:** Update `ui.md` line 9 and the `SKILL.md` line 14.
- [ ] **Step 4: Manual pass** at 1280px, 1024px and 375px (`pnpm -C web dev`, or Playwright screenshots of `/dev/ui/start/overview`, `/dev/ui/mmerp/data-table`, `/dev/ui/patterns/bulk`, `/dev/ui/templates/record`). Expected: the drawer opens at 375px, the TOC is hidden below `lg`, Do/Don't stacks to one column, and the template strip sits above the full page.
- [ ] **Step 5:** Run `make lint` and `make test` from the repo root. Expected: pass.
- [ ] **Step 6:** Commit: `feat(web): /dev/ui template notes and docs`.

## Shared/ui exports deliberately without a page

`ModalFooter` (shown inside `FormModalDoc`), `MenuAction` (inside `MoreActionsDoc`), `OrgUnitName` (inside `TreeListDoc`), `Toasts` (the host that `notify` needs, not used by screens), `ListView`/`useListView` (inside `DataTableDoc`), `SegmentedControl.module.css` (styling only). From `shared/document`: `DocumentScreen`, `CreatePage`, `ApprovalRules`, `DocumentHistory`, `DiscussionPanel`, `DocumentActions` need a live API beyond the offline samples. From `shared/jobs`/`shared/org`: `ExportButton`, `ImportDialog`, `OrgTree`. Add a page for any of these when the offline API can serve it.
