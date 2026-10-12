# Components

What exists, how to use it, and how to build a new one. Imports are from `@/shared/ui/…` and `@/shared/document`. Every component here is on `/dev/ui` in each of its states.

Areas never import Mantine's `Table`, `Modal`, `Badge`, `Notification`, `NumberInput`, `@mantine/dates` or `@mantine/notifications`: ESLint points to the `shared/ui` replacement. Other Mantine primitives (`Button`, `ActionIcon`, `Select`, `Alert`, `Menu`, `Tabs`, `Tooltip`, `Stack`, `Group`, `Text`, `Title`) are used directly, with the variants below.

## Button

| Variant | Code | Use |
| --- | --- | --- |
| Primary | `<Button>` (filled, primary) | The single main action of a view: Save, Create, Approve |
| Default | `variant="default"` (white, thin border) | Every other action: Cancel, Export, Retry, Clear filters |
| Danger | `color="danger"` (filled) | Only the confirm button of a destructive confirmation dialog |
| Danger text | `variant="subtle" color="danger"` | Every destructive action that asks first: Delete draft, Cancel document, Revoke, Discard changes |
| Subtle | `variant="subtle"` | Low-emphasis action in content: Add row, Change approver |
| Light | `variant="light"` | Selected state of a toggle-like button (header icons on their own page) |

Sizes follow where the button sits, not how important it is:

| Level | Where | `size` | Height | Icon |
| --- | --- | --- | --- | --- |
| Page | Page header actions, `FormActions`, modal footer, empty and error states | default (`sm`) | 32px | `icon.button` |
| Section | `TabActions`, section headers, the `DocumentPage` side column | `xs` | 24px | `icon.text` |
| Inline | Right after a line of text | `compact-xs` | 22px | `icon.text` |

States: hover darkens (filled) or tints (default, subtle); `loading` shows a spinner and blocks clicks (`FormActions` shows it only after 300ms); `disabled` greys out. Keep a form's submit button enabled and let validation explain what is missing.

Do:

```tsx
<Button leftSection={<IconPlus {...icon.button} />} component={Link} to="/sales/customers/new">
  {t('sales.customers.create')}
</Button>
<Button size="xs" variant="default" onClick={openUpload}>{t('shared.attachments.add')}</Button>
```

Don't:

```tsx
<Button variant="outline" color="teal">…</Button>   // invented variant and colour
<Button size="md">…</Button>                          // off the two control heights
<Button>{t('a')}</Button><Button>{t('b')}</Button>    // two primary buttons in one view
<Button color="danger">{t('shared.export')}</Button>  // red for an action that loses nothing
```

Where each one goes, and how many per place: `actions.md`.

## Icon button

`ActionIcon variant="subtle" color="gray" size="lg"` with the icon at `icon.button`, wrapped in a `Tooltip`, with an `aria-label` that repeats the tooltip. Beside 32px buttons or inputs use `size="input-sm"` so it is as tall as they are; a remove-row button is grey, never red.

```tsx
<Tooltip label={t('shared.attachments.remove')}>
  <ActionIcon variant="subtle" color="gray" size="lg" aria-label={t('shared.attachments.remove')} onClick={remove}>
    <IconTrash {...icon.button} />
  </ActionIcon>
</Tooltip>
```

## More actions

`MoreActions` (`@/shared/ui/MoreActions`) is the "…" menu: children are `Menu.Item`s. `place="row"` (default) is quiet, for a table row's `menu`; `place="page"` is outlined and 32px, for page actions beyond three. Renders nothing without children.

```tsx
<MoreActions place="page">
  <Menu.Item leftSection={<IconShoppingCart {...icon.text} />} onClick={createOrder}>{t('sales.quote.create_order')}</Menu.Item>
</MoreActions>
```

## Section

`Section` (`@/shared/ui/Section`) is a titled region of a page that is not a form: a table, a group of facts. Title (`h2`, 16px), optional description, optional `actions` on the right (`xs` buttons); consecutive sections are separated by a rule. Inside a form use `FormSection`; in a document's side column the template frames its own panels.

```tsx
<Section title={t('hrm.calendar.weeks')} actions={<Button size="xs" variant="default" onClick={addWeek}>{t('hrm.calendar.add_week')}</Button>}>
  <DataTable … />
</Section>
```

## Status

| Component | Use |
| --- | --- |
| `DocumentStatus` (`shared/document`) | A document's `draft` / `pending_approval` / `posted` / `cancelled`. Colours are fixed: neutral, warning, success, danger. A document type may rename the label (`<doc_type>.status.<status>`), never recolour it |
| `StatusBadge tone=…` (`shared/ui/StatusBadge`) | A catalog record's state or a module's own progress: `positive`, `neutral`, `info`, `warning`, `negative` |

Both render a tinted badge: 20px tall, 12px text, weight 500, radius 4, sentence case, no dot, no icon. The label always names the state. Module progress (delivery, payment, ordered) uses `neutral` or `info`, so the four document colours keep their meaning; the area wraps it in its own small component (e.g. `sales/components/DocProgress.tsx`).

A status label outside the badge (a status filter's options) comes from `useStatusLabel()` (`shared/document`): `label(docType, status)`, with the same `docType` the area passes to `DocumentStatus` (its record type, e.g. `docTypeOf(kind)` in sales).

```tsx
<StatusBadge tone={active ? 'positive' : 'neutral'}>{t(active ? 'sales.customer.status.active' : 'sales.customer.status.inactive')}</StatusBadge>
```

## Alert (banner)

Mantine `Alert`, default (tinted) variant, `color` by meaning, always with an icon at `icon.button`. Use for a document-level error (`role="alert"`), a warning about the data on screen, or the read-only notice (`ReadOnlyBanner`). It sits at the top of the region it is about.

## Fields (`shared/ui/form`)

Every field is wired to React Hook Form and renders: label above (12px, 600, `dimmed`), `*` when required, description under the input, error under that. Inputs are 32px, radius 6, input border token (`gray-4`, see `tokens.md`), focus ring primary.

| Field | Data |
| --- | --- |
| `TextField` (`multiline`, `wide`) | Text; long text spans the row |
| `PasswordField` | Password, with show/hide |
| `CheckboxField` | Yes/no, inside `InputRow` when next to inputs |
| `DateField`, `MonthField` | A date (DD/MM/YYYY), a month |
| `SelectField` | One of a list; search box past 7 options; no check mark, the chosen option is tinted |
| `MoneyField` | Integer minor units, thousands separators, right-aligned |
| `DecimalField` | Fractional quantity with declared decimals |
| `IntegerField` | A whole number without digit grouping (port, year, levels) |
| `FileField` | One file |
| `SensitiveField` | Masked value with Show/Hide; reveal is audited |
| `OrgUnitField`, `OrgUnitSelect` | An org unit within the user's scope |

Every field takes `bare` for a table cell: no visible label (kept as `aria-label`), no description, numbers right-aligned without spin buttons.

| Form part | Use |
| --- | --- |
| `Form` | The form: validation, focus on the first error, server errors via `fieldOf`, leave guard, Ctrl/Cmd+S; tells an enclosing `UnsavedProvider` whether it holds unsaved edits |
| `UnsavedProvider`, `useUnsaved` | Whether a form of the page has unsaved edits; `DocumentPage` provides it, `DocumentActions` reads it. A page-level action that reloads the record reads `useUnsaved()` the same way |
| `FormSection` | Titled group of fields; `wide` puts the title on top and the content full width (for a table); at most 1200px wide |
| `FormRow`, `InputRow` | Something that spans the field grid; a control level with the inputs beside it |
| `FormActions` | The footer: sticky with Discard on a page (no `onCancel`), Cancel + submit in a modal (`onCancel`) |
| `LinesTable` + `LineColumn` | A document's lines: one row per line, typed cells (`cell(index, bare, line)`), Enter to the next cell, add/remove (`minLines`), `footer`; cards below 1024px |
| `LinesTotals` | Totals under the lines, right-aligned, the last row bold, with a `note` on where the figures come from |
| `InlineEditField` | One value of a record page that turns into an input on click/Enter; Enter or ✓ saves, Esc or ✕ cancels, blur keeps it open; `readOnly` without permission |

Filter inputs (`SearchInput`, `MonthFilter`, `YearSelect`, `OrgUnitSelect`, a plain `Select`) carry their name as placeholder plus `aria-label`, a 16px leading icon, and a width from `filterWidth` (`tables.md`).

## Data display

| Component | Use |
| --- | --- |
| `DataTable` | Every table. Columns declare `role` (`title`, `status`, `meta`, `hidden`), `numeric`, `sortable`, `band`, `strong`, `width`. Rows 40px, header 36px on the sunken tone, row hover tint, cards below 1024px. Props: `href` (first column links the row), `onRowClick` (side panel, settings modal), `menu` (row "…"), `selection` (checkbox column), `onResize` (drag or ←/→ on a column edge), `group`, `footer`, `maxHeight`. Rules in `tables.md` |
| `ListPage` | `filters` (left), `tools` (right: `useListView().tools`), `bulk` (replaces the bar while rows are selected), `paging` |
| `useListView(listKey, columns)` | The user's saved views and column choices for a list, stored on the server; returns `columns`, `onResize`, `tools` |
| `BulkBar` | "N selected", the bulk actions (children), Clear selection |
| `SidePanel` + `usePeek` | Read-only preview of a list row on the right, 480px, not modal, with "Open full page"; the open id on the URL (`?peek=`) |
| `FieldList` | Read-only label/value pairs, label left in `dimmed` |
| `CellGrid` | A grid of number inputs on two axes (timesheet). Inside a form, pass `invalidId={fieldId('<field>')}` and `errorId` and render the field's error under the grid: the first invalid cell then takes the focus from the footer's "Go to next field" |
| `TileGrid`, `StatTile`, `LinkTile` | Home and overview pages: a count that opens its list, a destination |
| `PersonName`, `PersonAvatar` | A person: initials on a colour derived from the name, round |
| `OrgUnitName` | An org unit with its kind icon (square, radius 4) |

## Feedback and overlays

| Component | Use |
| --- | --- |
| `FormModal` | A form of at most 6 fields; radius 12, fullscreen below 1024px |
| `useConfirm` | Before deleting, cancelling a posted document, rejecting; the confirm button repeats the verb; `danger: true` for destructive steps |
| `notifySuccess` | After a save: bottom right, closes after 3 seconds |
| `notifyUndo(message, undoLabel, onUndo)` | A reversible action that already took effect, with Undo, 5 seconds (`actions.md`) |
| `notifyError` | A background job failed: stays until closed. Errors of the current action go next to their cause, not here |
| `EmptyState`, `ErrorState`, `PageSkeleton`, `ContentSkeleton`, `ForbiddenPage`, `NotFoundPage`, `ReadOnlyBanner` | The states of a region (see `patterns.md`). `EmptyState`'s icon is a 24px glyph (`icon.empty`) in a 48px neutral circle; pass another Tabler icon with `icon` |
| `Menu` | Secondary actions behind "…"; radius 8, overlay shadow |
| `Tooltip` | Name of an icon-only control; radius 4 |
| `Tabs` | Tabs of a `RecordPage` or a wide `DocumentPage`; the selected tab lives in the URL |

## Building a new component

1. Check this file and `/dev/ui` first; extend an existing component with a prop before adding a new one.
2. Decide where it lives: no business concept → `shared/ui`; a document feature → `shared/document`; a product concept (employee picker, payroll breakdown) → that product's area, assembled from `shared/ui` pieces.
3. Build it on Mantine primitives and theme tokens. Styling that props cannot express goes in a CSS module next to it, using only `var(--mantine-…)` and `var(--mmerp-…)` variables; a size specific to the component is allowed there with a one-line comment saying why.
4. Pick the variant, size, radius and colour from the tables above and in `tokens.md`; a new look needs a new row here first, agreed before code.
5. Cover every state it can be in: loading, empty, error, disabled, selected, focus, read-only; keyboard reachable, `aria-label` on icon-only controls.
6. Add it to `/dev/ui` (`shared/ui/DevUiPage.tsx`) in each state, and add a row to this file.

```tsx
// shared/ui/Example.tsx
import { Paper, Stack, Text } from '@mantine/core'
import classes from './Example.module.css'

// Example frames a summary block on the page; a flat surface, framed, not shadowed.
export function Example({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Paper withBorder p="md">
      <Stack gap="xs">
        <Text size="sm" c="dimmed">{title}</Text>
        <div className={classes.body}>{children}</div>
      </Stack>
    </Paper>
  )
}
```

```css
/* Example.module.css: tokens only. */
.body {
  border-top: 1px solid var(--mantine-color-default-border);
  padding-top: var(--mantine-spacing-xs);
}
```
