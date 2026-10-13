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
| 0 | Tint behind tinted badges, alerts, selected items (`variant="light"`) | Palest step keeps a full table of badges quiet |
| 6 | Filled buttons and badges, links | ≥ 4.5:1 with white text on every scale |
| 7 | Text on the tint; hover of a filled control | ≥ 5.5:1 on its own tint |

### Neutrals and surfaces

| Token | How to use | Value | Use for |
| --- | --- | --- | --- |
| text | default, `var(--mantine-color-text)` | `#292A2E` | Body text, values, identifiers (14.3:1) |
| text subtle | `c="dimmed"`, `var(--mantine-color-dimmed)` | `#505258` | Labels, captions, descriptions, column headers (7.8:1) |
| placeholder | automatic | `#6B6E76` | Input placeholders only (5.1:1) |
| border | `withBorder`, `var(--mantine-color-default-border)` | `#0B120E24` (translucent) | Dividers, table rules, frames, outlined buttons. Translucent so it reads on white and on the sunken tone alike |
| frame border | automatic on `Paper withBorder` and table rules (`gray-3`) | `#DDDEE1` | Same tone as `border`, drawn solid by Mantine |
| input border | automatic (`gray-4`) | `#B4B7BD` | Edges of inputs, checkboxes, radios and inline edit; filter bar inputs use `border` (`tables.md`) |
| surface | `var(--mmerp-elevation-surface)`, default `Paper` background | `#FFFFFF` | Tables, forms, cards, the page |
| surface sunken | `var(--mmerp-elevation-surface-sunken)` | `#F8F8F8` | Group and total rows, tile hover, grid cells that take no value |
| disabled | automatic | `#F0F1F2` | Disabled input background |
| neutral | `color="gray"` | tint `#F0F1F2`, text `#505258` | Neutral badges, quiet icon buttons, empty-state icon |

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
<Alert color="discovery">…</Alert> // discovery for something that is not new
```

## Typography

Font: Inter Variable, self-hosted. Lines sit on a 4px grid.

| Key | Size / line | Use |
| --- | --- | --- |
| `xs` | 12 / 16 | Captions, column headers, badges, field labels. The minimum size |
| `sm` | 14 / 20 | Body text, table cells, inputs, buttons, descriptions |
| `md` | 14 / 20 | Same as `sm` (Mantine's default size) |
| `lg` | 16 / 24 | Modal titles, larger body text |
| `xl` | 20 / 24 | Stat values in tiles |
| `h1` (`<Title order={1}>`) | 24 / 28, weight 650 | Page title, one per page, set by the page template |
| `h2` (`<Title order={2}>`) | 16 / 20, weight 650 | Section title |
| `h3` (`<Title order={3}>`) | 14 / 20, weight 650 | Sub-section title |

Weights: 400 body, 500 buttons, tabs, navigation, identifiers and column headers; 600 field labels, totals, emphasis; 650 headings. At most three heading levels on a page.

Field labels are 12px, weight 600, `dimmed`: lighter than the value they name. The theme sets this on every input; a screen sets nothing.

Numbers and money use `font-variant-numeric: tabular-nums` (`DataTable` numeric columns, `MoneyField` and `DecimalField` already do).

## Spacing

| Key | Value | Use |
| --- | --- | --- |
| `xxxs` | 2 | Between a line and its caption (name over date in a history entry) |
| `xxs` | 4 | Icon to its text, label to its input |
| `xs` | 8 | Controls in one strip (filter bar, button group), rows of a `FieldList` |
| `sm` | 12 | Inside a dense block, between filter bar and table in a drawer |
| `md` | 16 | Between fields, around card content, header to divider |
| `lg` | 24 | Between page regions (the page template's gap), between form sections |
| `xl` | 32 | Page side padding from 1024px, empty-state padding |

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
| `md` | 6 | Default: buttons, inputs, navigation items, alerts |
| `lg` | 8 | `Paper` (cards, tiles), menus, popovers, dropdowns |
| `xl` | 12 | Modals |
| round | — | Avatars, page numbers in the paging bar |

Components already carry their radius through theme defaults; set `radius` only on a new component, by this table.

## Elevation

| Level | Surface | Shadow | Use |
| --- | --- | --- | --- |
| Sunken | `--mmerp-elevation-surface-sunken` | none | A well inside the page: group and total rows, unavailable cells |
| Default | `--mmerp-elevation-surface` | none | The page and everything flat on it; framed with `withBorder` |
| Raised | surface | `shadow="xs"` | Reserved for a card that can be dragged; no screen has one yet |
| Overlay | surface | `md`, `lg`, `xl` (all `0 8px 12px #1E1F2126, 0 0 1px #1E1F214F`) | Menus, popovers, modals, toasts: Mantine applies it |

A card on the page is `Paper withBorder`, never a shadow.

## Sizes

| What | Value | Where |
| --- | --- | --- |
| Page-level control (button, input, select, pagination, `ActionIcon size="input-sm"`) | 30px, `size="sm"` (default) | `--mmerp-control-height`, applied in `global.css` |
| Section-level control | 24px, `size="xs"` | `--mmerp-control-height-compact` |
| Touch target below 1024px | 44px | `global.css` |
| Header | 48px | `app/AppLayout.tsx` |
| Navigation | 240px, 56px collapsed | `app/AppLayout.tsx` |
| Table row / header row | 40px / 36px | `DataTable.module.css` |
| Grid cell (timesheet) | 30px | `CellGrid.module.css` |
| Document side column, inbox list | 360px | `DocumentPage`, `InboxPage` |
| Content width (form sections, a document's main column) | 1200px | `--mmerp-content-width`, applied by `FormSection` and `DocumentPage` |
| Side panel, side-column tab | 480px | `readingWidth` (`page/Page.tsx`) |
| Breakpoints | `sm` 768, `md` 1024 (navigation becomes a drawer, tables become cards), `lg` 1280 (design size) | `theme.ts` |

A filter input's width is the one size an area sets, always from `filterWidth` (`theme.ts`): `search` 280, `select` 200, `unit` 240 (org unit, legal entity), `month` 160, `year` 120. `SearchInput`, `MonthFilter`, `YearSelect` and `OrgUnitSelect` already use theirs; a plain `Select` sets `w={{ base: '100%', md: filterWidth.select }}`.

## Icons

Tabler icons (`@tabler/icons-react`), outline, stroke 1.5, always through `icon`:

| Constant | Size | Use |
| --- | --- | --- |
| `icon.text` | 16 | In text, tables, inputs, `xs` buttons |
| `icon.button` | 18 | In `sm` buttons, alerts, icon buttons |
| `icon.empty` | 24 | Empty-state illustration icon |

Decorative icons are `aria-hidden` and always sit next to text.

## Motion

Mantine's own transitions only (menus, modals, toasts), 200ms at most; `respectReducedMotion` is on. No page transitions, no animation for decoration.
