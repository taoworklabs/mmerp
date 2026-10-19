# Components

What exists, how to use it, and how to build a new one. Imports are from `@/shared/ui/…` and `@/shared/document`. Every component here is on `/dev/ui` in each of its states.

Areas never import Mantine's `Table`, `Modal`, `Badge`, `Notification`, `NumberInput`, `@mantine/dates` or `@mantine/notifications`: ESLint points to the `shared/ui` replacement. Other Mantine primitives (`Button`, `ActionIcon`, `Select`, `Alert`, `Menu`, `Tabs`, `Tooltip`, `Stack`, `Group`, `Text`, `Title`) are used directly, with the variants below.

## Button

| Variant | Code | Use |
| --- | --- | --- |
| Primary | `<Button>` (filled, primary) | The single main action of a view: Save, Create, Approve |
| Default | `variant="default"` (white, thin border) | Every other action: Discard changes, Export, Retry, Clear filters |
| Danger | `color="danger"` (filled) | Only the confirm button of a destructive confirmation dialog |
| Danger text | `variant="subtle" color="danger"` | Every destructive action that asks first: Delete draft, Cancel document, Revoke |
| Danger outline | `variant="outline" color="danger"` | Only Reject (`ApprovalDecision`, the inbox's `ApprovalPanel`), paired with Approve |
| Subtle | `variant="subtle"` | Low-emphasis action in content: Add row, Change approver; `color="gray"` for the way out beside a primary button (Cancel, "Quay lại") |
| Light | `variant="light"` | Adding to a form from inside it, with `IconPlus`: "Add line" under lines (`LinesTable`), "Add step" (approval rule), "Add sensitive details"; the selected state of a toggle-like button (header icons on their own page) |

Every button has the one size, the default `sm` (30px), with `icon.button` (18px), wherever it sits: page header, tabs, section headers, the side column, footers, among content. Never set `size` on a button; a less important action is quieter in style, not smaller.

States: hover darkens (filled) or tints (default, subtle); `loading` shows a spinner and blocks clicks (`FormActions` shows it only after 300ms); `disabled` greys out. Keep a form's submit button enabled and let validation explain what is missing.

Do:

```tsx
<Button leftSection={<IconPlus {...icon.button} />} component={Link} to="/sales/customers/new">
  {t('sales.customers.create')}
</Button>
<Button variant="default" onClick={openUpload}>{t('shared.attachments.add')}</Button>
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

`ActionIcon variant="subtle" color="gray" size="lg"` with the icon at `icon.button`, wrapped in a `Tooltip`, with an `aria-label` that repeats the tooltip. Beside 30px buttons or inputs use `size="input-sm"` so it is as tall as they are; a remove-row button is grey, never red.

```tsx
<Tooltip label={t('shared.attachments.remove')}>
  <ActionIcon variant="subtle" color="gray" size="lg" aria-label={t('shared.attachments.remove')} onClick={remove}>
    <IconTrash {...icon.button} />
  </ActionIcon>
</Tooltip>
```

## More actions

`MoreActions` (`@/shared/ui/MoreActions`) is the "…" menu: children are `Menu.Item`s. `place="row"` (default) is quiet, for a table row's `menu`; `place="page"` is outlined and 30px, for page actions beyond three. Renders nothing without children.

```tsx
<MoreActions place="page">
  <Menu.Item leftSection={<IconShoppingCart {...icon.text} />} onClick={createOrder}>{t('sales.quote.create_order')}</Menu.Item>
</MoreActions>
```

## Section

`Section` (`@/shared/ui/Section`) is a titled region of a page that is not a form: a table, a group of facts. Title (`h2`, 16px), optional description, optional `actions` on the right (default-size buttons); consecutive sections are separated by a rule. Inside a form use `FormSection`; on a `DetailPage` every block is a `Panel` card instead.

```tsx
<Section title={t('hrm.calendar.weeks')} actions={<Button variant="default" onClick={addWeek}>{t('hrm.calendar.add_week')}</Button>}>
  <DataTable … />
</Section>
```

## Status

| Component | Use |
| --- | --- |
| `DocumentStatus` (`shared/document`) | A document's `draft` / `pending_approval` / `posted` / `cancelled`. Colours are fixed: neutral, warning, success, danger. A document type may rename the label (`<doc_type>.status.<status>`), never recolour it |
| `StatusBadge tone=…` (`shared/ui/StatusBadge`) | A catalog record's state or a module's own progress: `positive`, `neutral`, `info`, `warning`, `negative` |

Both render a tinted badge: 20px tall, 12px text, weight 500, radius 4, sentence case, no dot, no icon. The label always names the state. Module progress (delivery, payment, ordered) uses `neutral` or `info`, so the four document colours keep their meaning; the area wraps it in its own small component (e.g. `sales/components/DocProgress.tsx`).

A document list filters by status with `DocumentStatusFilter` (`shared/document`). Any other status label outside the badge comes from `useStatusLabel()` (`shared/document`): `label(docType, status)`, with the same `docType` the area passes to `DocumentStatus` (its record type, e.g. `docTypeOf(kind)` in sales).

```tsx
<StatusBadge tone={active ? 'positive' : 'neutral'}>{t(active ? 'sales.customer.status.active' : 'sales.customer.status.inactive')}</StatusBadge>
```

## Alert (banner)

Mantine `Alert`, default (tinted) variant, `color` by meaning, always with an icon at `icon.button`. Use for a document-level error (`role="alert"`), a warning about the data on screen, or the read-only notice (`ReadOnlyBanner`). It sits at the top of the region it is about.

## Fields (`shared/ui/form`)

Every field is wired to React Hook Form and renders: label above (12px, 600, `dimmed`), `*` when required, description under the input, error under that. Inputs are 40px among a form section's fields and in a modal's body, 30px elsewhere (filter bar, lines table), radius 6, input border token (`--mmerp-input-border`, see `tokens.md`), focus ring primary. Every field takes `short` (a third of a section row instead of half); `DateField` and `MonthField` show a calendar icon and their format as placeholder (not when `bare` or read-only); `TextField` takes `placeholder` for a "VD:" sample.

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
| `SensitiveField` | Masked value with a "Hiện" (Show) link that underlines on hover; once shown and not edited, an eye-off button inside the input masks it again; each reveal is audited |
| `SensitiveValue` | The read-only face of a sensitive value in a `FieldList`: masked (with `tail`, the last four digits the API sends: `•••••• 4821`), "Hiện" shows it as text with "Ẩn" beside it; each reveal is audited; `revealAll` reveals it for a card's "Hiện tất cả" |
| `EditablePanel` | A group of a record's or document's values in a `Panel` card: title, optional `icon`, its read view (`children`: a `FieldList` showing every value the form edits, or a function given `open`, for a `MissingValue`); `actions` before "Sửa" ("Hiện tất cả"); with `editable`, "Sửa" on the title line turns the card into `edit(close)` (the group's `Form` → `FormFields` → `FormActions onCancel={close}`) in place, focus into the first field, back once saved or cancelled. Never a `FormModal` for a group |
| `OrgUnitField` | An org unit within the user's scope; the chosen unit shows its kind icon, as in the list |
| `LegalEntityField` | A legal entity where the user holds a permission; the only one is picked already |
| `RemoteSelectField` | A record searched on the server (debounced, a page of 20), labelled `code · name`; keeps the label of its value when a search does not return it. Areas wrap it in their picker (`EmployeeField`, `CustomerField`, `ItemField`) |

Every field takes `bare` for a table cell: no visible label (kept as `aria-label`), no description, numbers right-aligned without spin buttons.

| Form part | Use |
| --- | --- |
| `Form` | The form: validation, focus on the first error, server errors at their field via `errorFields` (code → field, `lines.*.<field>` for a line), leave guard, Ctrl/Cmd+S; tells an enclosing `UnsavedProvider` whether it holds unsaved edits |
| `UnsavedProvider`, `useUnsaved` | Whether a form of the page has unsaved edits; `DocumentPage` provides it, `DocumentActions` reads it. A page-level action that reloads the record reads `useUnsaved()` the same way |
| `FormSection` | Titled group of fields: title 16/650, description 12px; fields on six columns from 1280px (half a row each, `short` a third, `wide` the whole row); `wide` on the section puts the title on top and the content full width (for a table); `compact` puts the title on top with four fields to a row (a short request: new leave, new overtime); at most 1200px wide; a read-only field is a grey fill without an edge |
| `FormFields` | The fields of a form inside a card (`EditablePanel`'s edit view): two to a row from 768px |
| `FormRow`, `InputRow` | Something that spans the field grid; a control level with the inputs beside it |
| `FormActions` | The footer: sticky on a page (no `onCancel`), with a shadow while the form runs on under it; editing, it shows once dirty with "Unsaved changes" and Discard; creating (`cancelTo`: the list), it shows from the start with Cancel. In a modal or an `EditablePanel` card (`onCancel`): Cancel + submit, in a `ModalFooter` in a modal. Cancel is quiet grey everywhere |
| `LinesTable` + `LineColumn` | A document's lines: one row per line, typed cells (`cell(index, bare, line)`), Enter to the next cell, add/remove (`minLines`), `footer`; cards below 1024px. `readOnly` with `lines` shows saved lines as text through each column's `value`, without a form |
| `LinesTotals` | Totals under the lines, right-aligned, the last row bold, with a `note` on where the figures come from |
| `InlineEditField` | One value of a record page that saves on its own (a user's email), apart from the record's groups; turns into an input on click/Enter; Enter or ✓ saves, Esc or ✕ cancels, blur keeps it open; `readOnly` without permission |

A list's filter bar holds `SearchInput` and filter chips: `FilterChip` (options from `data`; `icon` is the icon component, the chip sizes it; `searchable`; `clearable={false}` for a filter that always holds a value), `OrgUnitFilter`, `LegalEntityFilter` (always holds one: `useLegalEntities(scope, picked)` gives the options and the chosen one, the first by default), `MonthFilter`, `YearFilter`. A chip is named after its filter and reads "Name: value", tinted with its own ✕ (with a tooltip), once set; `{...filterBar.bind(key)}` from `useListParams` wires one to the URL; areas set no width (`tables.md`).

To switch between views of one feed (an activity feed's All · Comments · Changes · Views), use Mantine's `SegmentedControl`: the theme draws it as a thin frame of plain labels with the chosen one tinted and edged in primary (`SegmentedControl.module.css`); set nothing on it but `w="fit-content"`.

## Data display

| Component | Use |
| --- | --- |
| `DataTable` | Every table. Columns declare `role` (`title`, `status`, `meta`, `hidden`), `kind` (`text`, the default, `person`, `code`, `date`, `number`, `status`: look and width), `sortable`, `band`, `strong`, `width`; a zero figure or an empty cell shows "–". Framed (1px `border`, radius 6) wherever it is part of a page; as a list page's whole content it fills the page unframed (`ListPage` drops the frame); rows 40px, header 36px on the table header tone (12px, 600) over a rule, row hover tint, cards below 1024px or with `cards` (a `DetailTabs` tab). Props: `href` (first column links the row), `onRowClick` (side panel, settings modal), `menu` (row "…"), `selection` (checkbox column), `onResize` (drag or ←/→ on a column edge), `group`, `footer`, `maxHeight` (without it the table takes the height its parent gives, e.g. a list page's frame). Rules in `tables.md` |
| `TreeList` | A hierarchy (the org tree): rows 36px joined to their parent by guide lines, each branch folds (all open at first; from 1024px a folded one says how many units sit directly under it). Props: `nodes` (nested), `render` (the name), `meta` (quiet facts after it), `menu` (row "…"), `toggleLabel`. Unframed; below 1024px rows and toggles are 44px and `meta` is left out |
| `ListPage` | `tabs` (ways to view the same list, e.g. tree and table: a `Tabs` element under the header, its line replacing the header rule; the choice is on the URL), `filters` (left), `onClearFilters` (given while a filter is set: "Clear filters" ends the chips), `tools` (right: `useListView().tools`), `bulk` (replaces the bar while rows are selected), `list` (the list query, the `DataTable` props but `rows`, the `empty` and `noResults` texts and `paging`: it shows loading, error, empty, no results with "Clear filters", or the table, then the paging bar; children come before it). From 1024px it fills the window and only its content scrolls |
| `useListView(listKey, columns)` | The user's saved views and column choices for a list, stored on the server; returns `columns`, `onResize`, `tools` |
| `BulkBar` | "N selected", the bulk actions (children), Clear selection |
| `SidePanel` + `usePeek` | Read-only preview of a list row on the right, 480px, not modal, with "Open full page"; the open id on the URL (`?peek=`) |
| `Panel` (`shared/ui/page`) | A card on a `DetailPage`: 1px border, radius 6, 16px padding, `title` (600), optional `icon` and `actions` on its line |
| `DetailTabs`, `TabActions` | A record's own lists as tabs in one card of the side column (`{ value, label, content }`), the first open by default, the tab on the URL (replaced, not pushed); a tab's buttons in `TabActions`, right-aligned; tables in it use `DataTable` `cards` |
| `ActivityFeed` (`shared/document`) | A record's or document's comments and audit log in one timeline: comment box, filter (All · Comments · Changes · Views), days as uppercase headings with a rule; each line the time of day (3rem column), a 20px bullet (avatar for plain doings and comments; a coloured round icon for a status reached, a decision, a creation, a sensitive read) and a sentence (who 600, doing `dimmed`, what 600); a comment or two or more changed fields in a sunken well, tied to the bullet by a thread; consecutive reads fold; 30 lines, then "Show more". Passed as `activity` to `DocumentPage`; on a `DetailPage` it is a `Panel` of its own last in `side`, never a tab (on a `RecordPage` it is the "Hoạt động" tab, and `recent={5}` gives the overview's digest: comments and changes, no filter); the comment box is one line until focused; never build a separate history or discussion panel |
| `DocumentScreen` (`shared/document`) | A document's whole page on `DocumentPage`: `list` (breadcrumb, where a deleted draft leaves for), header `facts` joined by "·", `DocumentStatus`, `badges`, `ApprovalDecision`, the page's own buttons (`extraActions`, given the banner's `onError`), `DocumentActions` (print from `printLabel`/`printKind`), `ApprovalPanel decideElsewhere` (`subject`), attachments (`attachments`) and `ActivityFeed`. A detail page passes only its fields and form |
| `DocumentPreview` (`shared/document`) | An area's read-only preview of a document (inbox, side panel): loading and error states, `DocumentStatus`, an optional `notice`, then `rows` as a `FieldList`; a falsy row is left out |
| `CreatePage` (`shared/document`) | A new document's page: `<PageSkeleton>` while the actions load, `<ForbiddenPage>` without `action` (default `create`), then the form, given `onCreated` that refreshes the record, toasts `created` and opens it |
| `KeyFacts` | A record's 4–5 key figures under its title (`RecordPage` `summary`): label `xs` `dimmed` over value 500, side by side, two to a row below 768px |
| `MissingValue` | An empty value: "+ Bổ sung" for who may fill it (`onAdd` opens the form, `to` goes where it is entered), else "Chưa cập nhật" in `dimmed`. Never "—" for a value the user can add |
| `RecordPage`, `RecordColumns` (`shared/ui/page`) | One record with much to read: summary header, then its parts as full-width tabs on the header's rule (`tabs`, `aliases` for old values), the overview in `RecordColumns` (about 1.6/1) |
| `FieldList` | Read-only label/value pairs, label left in `dimmed` (`stacked`: label over value, two to a row from 768px, a row's third item `'wide'` spans both); a value may be a node (a badge, a `SensitiveValue`); a container fixes the label width with `--field-list-label` (`DetailPage` sets 11rem, so the cards' values line up down the page) |
| `CellGrid` | A grid of number inputs on two axes (timesheet). Inside a form, pass `invalidId={fieldId('<field>')}` and `errorId` and render the field's error under the grid: the first invalid cell then takes the focus from the footer's "Go to next field" |
| `TileGrid`, `StatTile`, `LinkTile` | Home and overview pages: a count that opens its list, a destination |
| `PersonName`, `PersonAvatar` | A person: initials on a colour derived from the name, round |
| `OrgUnitName`, `OrgUnitIcon` | An org unit with its kind icon (square, radius 4); `OrgUnitIcon` is the kind's tinted icon alone (`size`) |

## Feedback and overlays

| Component | Use |
| --- | --- |
| `FormModal` | A form of at most 6 fields (create on a list, a settings row, a separate action like transfer); never a group of a record's values, which edits in its card: 440px, one column (`wide`: 620px, for long text or a table), a header band (title 20/24 weight 650) under a rule, padding 24px at the sides and 16px above and below, radius 6, fullscreen below 1024px |
| `ModalFooter` | A modal's buttons, right-aligned in a band across the modal under a rule; every modal's buttons sit in it: `FormActions` in a modal, `useConfirm`, and modals with their own buttons |
| `useConfirm` | Before deleting, cancelling a posted document, rejecting; the confirm button repeats the verb; `danger: true` for destructive steps |
| `notifySuccess` | After a save: bottom right, closes after 3 seconds |
| `notifyUndo(message, undoLabel, onUndo)` | A reversible action that already took effect, with Undo, 5 seconds (`actions.md`) |
| `notifyError` | A background job failed: stays until closed. Errors of the current action go next to their cause, not here |
| `EmptyState`, `ErrorState`, `PageSkeleton`, `ContentSkeleton`, `ForbiddenPage`, `NotFoundPage`, `ReadOnlyBanner` | The states of a region (see `patterns.md`). `EmptyState`'s icon is a 24px glyph (`icon.empty`) in a 48px neutral circle; pass another Tabler icon with `icon` |
| `Menu` | Secondary actions behind "…"; radius 8, overlay shadow |
| `Tooltip` | Name of an icon-only control; radius 4 |
| `Tabs` | Tabs of a `fullWidth` `DocumentPage` and the view tabs of a `ListPage` (`tabs`); a record's own lists use `DetailTabs`; the selected tab lives in the URL |
| `InboxList`, `InboxGroup`, `InboxRow` | A feed: `InboxList` (`framed`: one frame, rows ruled apart, scrolling inside itself in a list page), `InboxGroup` (a sticky group heading on the sunken tone, e.g. a day), `InboxRow` (`icon` leads, `aside` ends, an 8px dot while unread) |

## Building a new component

1. Check this file and `/dev/ui` first; extend an existing component with a prop before adding a new one.
2. Decide where it lives: no business concept → `shared/ui`; a document feature → `shared/document`; a product concept (employee picker, payroll breakdown) → that product's area, assembled from `shared/ui` pieces.
3. Build it on Mantine primitives and theme tokens. Styling that props cannot express goes in a CSS module next to it, using only `var(--mantine-…)` and `var(--mmerp-…)` variables; a size specific to the component is allowed there with a one-line comment saying why.
4. Pick the variant, size, radius and colour from the tables above and in `tokens.md`; a new look needs a new row here first, agreed before code.
5. Cover every state it can be in: loading, empty, error, disabled, selected, focus, read-only; keyboard reachable, `aria-label` on icon-only controls.
6. Add it to `/dev/ui` (`app/devui`, one sidebar entry per component) in each state, and add a row to this file.

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
