# Review checklist

Run every item against the change before calling it done. An item that does not apply is marked so, not skipped.

## Tokens

- [ ] No colour code, `rgb(`, `hsl(` outside `theme.ts` (`make lint` checks).
- [ ] No pixel or rem number for spacing, font size or radius in a component (a filter input's width is the one exception); only theme keys (`gap="xs"`, `fz="xs"`, `radius="lg"`) or `var(--mantine-…)` / `var(--mmerp-…)`. A component-specific size sits in a `shared/ui` CSS module with a comment.
- [ ] Colours named by meaning (`color="danger"`, `c="dimmed"`), never by hue (`"red"`, `"orange"`) or step (`"gray.6"`).
- [ ] Icons through `icon.text` / `icon.button` / `icon.empty`.
- [ ] No `style` attribute and no CSS file in an area.

## Components

- [ ] Built from an existing `shared/ui` template and components; anything new and generic is in `shared/ui` with a `/dev/ui` entry and a row in `components.md`.
- [ ] One primary button per view; others `variant="default"`; `color="danger"` only for a step that loses data.
- [ ] Buttons sized by level: default (32px) in the page header, form footer, modal footer; `size="xs"` (24px) in sections, tabs and the side column. Controls in one strip share one height.
- [ ] Status through `DocumentStatus` (documents) or `StatusBadge` (everything else).
- [ ] Selection shown by the primary tint and primary text only.
- [ ] Shadows only on overlays; cards are `Paper withBorder`.

## Layout (`layout.md`)

- [ ] The screen type is named (list, document, record, create page, modal, settings, dashboard, report, inbox) and uses that type's template; no page-level layout of its own.
- [ ] Create in a `FormModal` only for ≤ 6 fields whose result lives in a list or is worked on its own page; otherwise a page.
- [ ] A document with a wide table (editable lines, grid, payroll) uses `fullWidth`; forms and main columns rely on the 1200px cap, not their own width.
- [ ] Breadcrumb points to the record's own list.

## Tables (`tables.md`)

- [ ] Identifier first and linked, numbers `numeric`, status columns last, every column has a `role`.
- [ ] Search and sort on identifier and date; filter widths from `filterWidth`; filters, sort, page, tab, side panel on the URL.
- [ ] Rows: one action opens on row click (`onRowClick`), two or more in the "…" menu; no buttons per row.
- [ ] Lists people reuse use `useListView` with a stable `listKey`.
- [ ] Bulk actions only with a real bulk operation: `selection` + `BulkBar`, run as a job, confirmed with the count, selection cleared on filter/sort/page change.
- [ ] Side panel only read-only, with its link to the page.
- [ ] Reconciliation tables: bands, totals, capped frame, zero as "–".

## Forms (`forms.md`)

- [ ] Labels above, `*` on required, `mode: 'onBlur'`, server field errors through `fieldOf`, others as the form banner.
- [ ] `FormActions` without `onCancel` on a page, with it in a modal; submit never disabled to signal errors.
- [ ] Same data, same field (`MonthField`, `IntegerField`, `MoneyField`…).
- [ ] Lines in `LinesTable` inside `FormSection wide`; totals only from the server, with their note.
- [ ] `InlineEditField` only for single master-data values; never documents, money, rates, sensitive fields.

## Actions (`actions.md`)

- [ ] One primary per place, rightmost; at most three header buttons, the rest in `MoreActions place="page"`.
- [ ] Buttons come from `allowed_actions` (records) or the list's `/actions` (create, import, export); nothing derived from status or role.
- [ ] Destructive actions are red text and confirm; what the server cannot take back confirms; reversible side-effect-free actions use `notifyUndo`; lifecycle actions never undo.
- [ ] Optimistic updates only for reversible, side-effect-free actions and display settings; failures roll back and show a banner.
- [ ] Only the listed shortcuts; every action reachable with Tab and Enter.

## Feedback (`feedback.md`)

- [ ] Every region has loading (skeleton), empty (and empty because of filters), error with Retry; forbidden in `routes.tsx`; not found on a detail page; read-only through `allowed_actions`.
- [ ] Every save, create and lifecycle action toasts on success; every failure is caught and shown next to its cause, never in a toast.
- [ ] Long work runs as a job with progress, not a spinner.

## Text

- [ ] Every string is an i18n key with `vi` and `en`; terms match `CONTEXT.md`.
- [ ] Buttons are specific verbs in sentence case; confirm buttons repeat the verb.
- [ ] Errors say what happened and what to do, next to their cause.
- [ ] Numbers, money and dates formatted through `@/shared/i18n`.

## Access and devices

- [ ] Everything works by keyboard; the focus ring is visible; modals trap and return focus.
- [ ] Icon-only buttons have an `aria-label` and a tooltip; decorative icons are `aria-hidden`.
- [ ] Colour is never the only signal: states carry text, errors carry text and an icon.
- [ ] Checked at 1280px and 1024px (`/dev/ui` for shared components); self-service screens also at 375px.

## Finish

- [ ] `make lint` and `make test` are green.
- [ ] A changed token or rule is updated in `theme.ts`, `references/tokens.md` and `ui.md` together.
