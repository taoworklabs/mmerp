# Tokens

Every value below is defined in `web/src/shared/ui/theme.ts` (or `global.css` where noted). If you change one there, change it here and in `ui.md` in the same commit.

How to reach a token:

| From | Use |
| --- | --- |
| A Mantine prop in any component | The theme key: `gap="xs"`, `p="md"`, `c="dimmed"`, `color="danger"`, `radius="lg"`, `fz="xs"`, `fw={600}` |
| A CSS module in `shared/ui` | The CSS variable: `var(--mantine-spacing-xs)`, `var(--mantine-color-default-border)`, `var(--mmerp-elevation-surface-sunken)` |
| An icon | The `icon` constant: `<IconPlus {...icon.button} />` |

## Colour

Each colour is a ten-step scale, palest (0) to darkest (9). Components read it by role, so a screen names the colour (`color="success"`), never the step.

| Step | Role | Why |
| --- | --- | --- |
| 0 | Tint behind alerts, selected items, set filter chips (`variant="light"`) | Palest step keeps large tinted areas quiet |
| 1 | Tint behind badges (neutral: step 2) | A step deeper, so a status reads at a glance in a column of rows |
| 6 | Filled buttons and badges, links | ≥ 4.5:1 with white text on every scale |
| 7 | Text on the tint; hover of a filled control | ≥ 5.5:1 on its own tint |
| 8 | Badge text | ≥ 7:1 on the badge tint |

### Neutrals and surfaces

| Token | How to use | Value | Use for |
| --- | --- | --- | --- |
| text | default, `var(--mantine-color-text)` | `#292A2E` | Body text, values, identifiers (14.3:1) |
| text subtle | `c="dimmed"`, `var(--mantine-color-dimmed)` | `#505258` | Labels, captions, descriptions, column headers (7.8:1) |
| placeholder | automatic | `#6B6E76` | Input placeholders only (5.1:1) |
| border | `withBorder`, `var(--mantine-color-default-border)` | `#0B120E24` (translucent) | Dividers, table rules, frames, outlined buttons. Translucent so it reads on white and on the sunken tone alike |
| frame border | automatic on `Paper withBorder` and table rules (`gray-3`) | `#DDDEE1` | Same tone as `border`, drawn solid by Mantine |
| input border | automatic on fields (`global.css`), `var(--mmerp-input-border)` | `#0B120E33` (translucent) | Edges of text fields and inline edit, a step stronger than `border`; the filter bar's search box uses `border`, like the chips and tool buttons beside it (`tables.md`) |
| control border | automatic (`gray-4`) | `#B4B7BD` | Edges of checkboxes and radios: small, so they keep the stronger edge |
| surface | `var(--mmerp-elevation-surface)`, default `Paper` background | `#FFFFFF` | Tables, forms, cards, the page |
| surface sunken | `var(--mmerp-elevation-surface-sunken)` | `#F8F8F8` | Group and total rows, tile hover, grid cells that take no value |
| table header | `var(--mmerp-table-header)` | `#F4F5F6` | A table's column header row (`DataTable`, `LinesTable`, `CellGrid`): a step deeper than the sunken rows |
| disabled | automatic | `#F0F1F2` | Disabled input background |
| neutral | `color="gray"` | tint `#F0F1F2`, text `#505258` (badges: `#DDDEE1`, `#3B3D42`) | Neutral badges, quiet icon buttons, empty-state icon |

### Primary and semantic scales

| Colour | `color=` | Filled (6) | Tint (0) | Text on tint (7) | Means |
| --- | --- | --- | --- | --- | --- |
| primary | default | `#1868DB` | `#E9F2FE` | `#1558BC` | The one primary action, links, focus, selected item |
| success | `"success"` | `#5B7F24` | `#EFFFD6` | `#4C6B1F` | Posted, active, done |
| warning | `"warning"` | `#BD5B00` | `#FFF5DB` | `#9E4C00` | Pending approval, needs attention |
| danger | `"danger"` | `#C9372C` | `#FFECEB` | `#AE2E24` | Errors, destructive actions, cancelled |
| info | `"info"` | `#1868DB` | `#E9F2FE` | `#1558BC` | Neutral information; same scale as primary |
| discovery | `"discovery"` | `#964AC0` | `#F8EEFE` | `#803FA5` | Something new or in beta; nothing else |

Field errors and the required `*` use `var(--mantine-color-error)` = `#AE2E24`.

Selected state, everywhere (navigation, menus, dropdown options, inbox rows): tint `var(--mantine-primary-color-light)` and text `var(--mantine-primary-color-light-color)`. Nothing else marks selection.

Avatars take their colour from `PersonAvatar` (derived from the name); set no colour on a person yourself.

Do:

```tsx
<Text size="sm" c="dimmed">{t('sales.customers.description')}</Text>
<Alert color="warning" icon={<IconAlertTriangle {...icon.button} />}>{t('hrm.payroll.sources_changed')}</Alert>
```

Don't:

```tsx
<Text c="#666">…</Text>            // colour code: lint fails outside shared/ui
<Text c="gray.6">…</Text>          // a step instead of a role: use c="dimmed"
<Alert color="orange">…</Alert>    // a hue instead of a meaning: use color="warning"
// Accent hues (grape, blue, teal, orange, red, green) only name a kind: PersonAvatar, OrgUnitIcon, the file-type icons of AttachmentPanel.
<Alert color="discovery">…</Alert> // discovery for something that is not new
```

## Typography

Font: Inter Variable, self-hosted. Lines sit on a 4px grid.

| Key | Size / line | Use |
| --- | --- | --- |
| `xs` | 12 / 16 | Captions, column headers, badges, field labels, form section descriptions. The minimum size |
| `sm` | 14 / 20 | Body text, table cells, inputs, buttons, page descriptions |
| `md` | 14 / 20 | Same as `sm` (Mantine's default size) |
| `lg` | 16 / 24 | Larger body text |
| `xl` | 20 / 24 | Modal titles (weight 650), stat values in tiles |
| `h1` (`<Title order={1}>`) | 24 / 28, weight 650 | Page title, one per page, set by the page template |
| `h2` (`<Title order={2}>`) | 16 / 20, weight 650 | Section title |
| `h3` (`<Title order={3}>`) | 14 / 20, weight 650 | Sub-section title |

Weights: 400 body, 500 buttons, tabs, navigation and identifiers; 600 field labels, column headers, totals, emphasis; 650 headings. At most three heading levels on a page.

Field labels are 12px, weight 600, `dimmed`: lighter than the value they name. The theme sets this on every input; a screen sets nothing.

Numbers and money use `font-variant-numeric: tabular-nums` (`DataTable` `number`, `code` and `date` columns, `MoneyField` and `DecimalField` already do).

## Spacing

| Key | Value | Use |
| --- | --- | --- |
| `xxxs` | 2 | Between a line and its caption (a name over a date) |
| `xxs` | 4 | Icon to its text, label to its input |
| `xs` | 8 | Controls in one strip (filter bar, button group), rows of a `FieldList` |
| `sm` | 12 | Inside a dense block, between filter bar and table in a drawer |
| `md` | 16 | Between fields, around card content, header to divider |
| `lg` | 24 | Between page regions (the page template's gap), between form sections |
| `xl` | 32 | Page side padding from 1024px, empty-state padding |
| `xxl` | 48 | Page side padding from 1280px; a record page's tabs and tab content, in from the frame's edge |

The scale is closed: if a gap seems to need 6, 10 or 20, the layout needs another level instead (a `Stack` inside a `Stack`).

```tsx
<Stack gap="xxxs">
  <Text size="sm">{name}</Text>
  <Text size="xs" c="dimmed">{formatDate(day)}</Text>
</Stack>
```

## Radius

| Key | Value | Use |
| --- | --- | --- |
| `xs` | 2 | Not assigned to any component yet |
| `sm` | 4 | Badges, tooltips, small icon tiles (`ThemeIcon` up to 28px) |
| `md` | 6 | Default: buttons, inputs, navigation items, alerts, `Paper` (cards, tiles), table and feed frames, the activity feed's detail well, modals |
| `lg` | 8 | Menus, popovers, dropdowns |
| `xl` | 12 | Not assigned to any component's frame (round icon tiles use it) |
| round | `100%` | Avatars, page numbers in the paging bar, dots |

Components already carry their radius through theme defaults; set `radius` only on a new component, by this table.

## Elevation

| Level | Surface | Shadow | Use |
| --- | --- | --- | --- |
| Sunken | `--mmerp-elevation-surface-sunken` | none | A well inside the page: group and total rows, unavailable cells; a table's header row sits a step deeper (`--mmerp-table-header`) |
| Default | `--mmerp-elevation-surface` | none | The page and everything flat on it; framed with `withBorder` |
| Raised | surface | `shadow="xs"` | Reserved for a card that can be dragged; no screen has one yet |
| Overlay | surface | `md`, `lg`, `xl` (all `0 8px 12px #1E1F2126, 0 0 1px #1E1F214F`) | Menus, popovers, modals, toasts: Mantine applies it |
| Pinned footer | body | `--mmerp-shadow-footer` (`0 -6px 8px -6px #1E1F2126`, upwards only) | A page form's footer while the form runs on under it (`FormActions`) |

A card on the page is `Paper withBorder`, never a shadow.

## Sizes

| What | Value | Where |
| --- | --- | --- |
| Control (button, input, select, pagination, `ActionIcon size="input-sm"`), wherever it sits | 30px, `size="sm"` (default); 40px among a form section's fields and in a modal's body (a lines table and `ModalFooter` keep 30px) | `--mmerp-control-height`, applied in `global.css`, raised in `Form.module.css` and the `Modal` body (`theme.ts`) |
| Touch target below 1024px | 44px | `global.css` |
| Header | 48px | `app/AppLayout.tsx` |
| Navigation | 240px, 56px collapsed; items 32px | `app/AppLayout.tsx`, `AppLayout.module.css` |
| Table row / header row | 40px / 36px | `DataTable.module.css` |
| Grid cell (timesheet) | 30px | `CellGrid.module.css` |
| Scrollbar (`ScrollArea`) | 8px; in a table it runs between the sticky header and footer | `theme.ts`, `DataTable.module.css` |
| Inbox list | 360px | `--mmerp-side-width` |
| Content width (form sections, a form page, a `fullWidth` document's tab) | 1200px | `--mmerp-content-width`, applied by `FormSection` and `Page centered` |
| Record or document page (`DetailPage`, `DocumentPage`) | 1440px, header included; main and side column about 62/38, one column below 1024px; card labels 11rem (`--field-list-label`) | `--mmerp-detail-width`, `DetailPage.module.css` |
| Side panel, side-column tab | 480px | `--mmerp-aside-width`, `readingWidth` (`page/Page.tsx`) |
| Breakpoints | `sm` 768, `md` 1024 (navigation becomes a drawer, tables become cards), `lg` 1280 (design size) | `theme.ts` |

Filter bar sizes live in `filterWidth` (`theme.ts`) and only the filter components read them: `search` 280 and `searchMin` 160 (the search box, narrowing when the bar is full), `chip` 320 (the widest a filter chip grows before its value truncates), `menu` 240 (a chip's dropdown). Areas set no size apart from the column widths of a lines table (`forms.md`).

## Icons

Tabler icons (`@tabler/icons-react`), outline, stroke 1.5, always through `icon`:

| Constant | Size | Use |
| --- | --- | --- |
| `icon.text` | 16 | In text, tables, inputs |
| `icon.button` | 18 | In every button, alerts, icon buttons |
| `icon.empty` | 24 | Empty-state illustration icon |

Decorative icons are `aria-hidden` and always sit next to text.

## Motion

Mantine's own transitions only (menus, modals, toasts), 200ms at most; `respectReducedMotion` is on. No page transitions, no animation for decoration.
