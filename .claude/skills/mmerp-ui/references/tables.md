# Tables and lists

Every table is `DataTable` (`@/shared/ui/DataTable`); a list screen is `ListPage` around it. Editable document lines are `LinesTable` (`forms.md`); a grid of cells on two axes is `CellGrid`.

## Anatomy

```
[🔍 Search      ] [Status ▾] [Org unit ▾] [Month ▾]          [Views ▾] [Columns]   ← filters … tools
┌──┬──────────────┬─────────────┬──────────┬────────────┬──────────┬───┐
│☐ │ Identifier ↕ │ Text        │ Date ↕   │   Amount ↕ │ Status   │ ⋯ │  header 36px, sunken tone
├──┼──────────────┼─────────────┼──────────┼────────────┼──────────┼───┤
│☐ │ DH-0001      │ Customer    │ 01/10    │     12.500 │ [Posted] │ ⋯ │  rows 40px, hover tint
└──┴──────────────┴─────────────┴──────────┴────────────┴──────────┴───┘
1.234 records                                     [50 / page ▾] ‹ 1 2 3 › ← paging
```

## Filter bar

- Inputs only, `xs` (8px) apart, 32px tall, in this order: search, then selects, then period filters. Saved views and columns (`tools`) sit at the bar's right end.
- Each input: placeholder = its name, the same text as `aria-label`, a 16px leading icon, `clearable` when optional, one width per kind from `filterWidth` in `theme.ts`:

| Kind | Component | Width |
| --- | --- | --- |
| Search | `SearchInput` (applies on Enter or blur; `/` focuses it) | `filterWidth.search` (280), built in |
| Status, type, any select | Mantine `Select` | `filterWidth.select` (200) |
| Org unit, legal entity | `OrgUnitSelect product="hrm" permission="hrm.leave.view"` (only units where the user has that permission; width built in), `Select` | `filterWidth.unit` (240) |
| Month | `MonthFilter` (value `YYYY-MM`) | `filterWidth.month` (160), built in |
| Year | `YearSelect` | `filterWidth.year` (120), built in |

- Every business list has a search and sorts on its identifier and date; documents also filter by status and their key dimension (employee, org unit, period).
- Filters, sort, page and page size live in the URL (`useListParams`); copying the URL reopens the same list. Changing a filter goes back to page 1.
- Below 1024px the bar folds into a bottom drawer behind a Filter button (`ListPage` does it).

```tsx
<Select
  aria-label={t('sales.customer.status')}
  placeholder={t('sales.customer.status')}
  leftSection={<IconCircleDot {...icon.text} />}
  w={{ base: '100%', md: filterWidth.select }}
  data={…}
  value={active || null}
  onChange={(v) => set({ filters: { active: v ?? '' } })}
  clearable
/>
```

## Saved views and columns

`useListView(listKey, columns)` (`@/shared/ui/ListView`) gives a list the signed-in user's own settings, stored on the server per user and per list:

- **Views**: a named copy of the list's URL query (filters and sort; never the page or the open side panel). Picking a view opens the list as saved; the Views button shows the active view's name; "Save current view" names the current filters; the active view can be deleted from the same menu.
- **Columns**: show or hide any column but the first (the identifier always shows), drag a column's right edge to resize it (or focus the edge and use ←/→, 16px a step), "Restore defaults" puts the designed layout back. Changes apply at once and save in the background.

`listKey` is `<area>.<list>` (`sales.customers`, `hrm.leaves`), lowercase, never renamed once released: saved settings are keyed by it.

```tsx
const view = useListView('sales.customers', columns)
…
<ListPage … filters={…} tools={view.tools}>
  <DataTable columns={view.columns} onResize={view.onResize} … />
</ListPage>
```

Every list with more than four columns or with filters people reuse gets `useListView`; small settings lists do not.

## Columns

| Column | Rule |
| --- | --- |
| Identifier | First, `role: 'title'`, sticky on horizontal scroll, never wraps, links the row (`href`); a person shows `PersonName` |
| Text | Left-aligned; long text wraps |
| Dates | Left-aligned, `formatDate` |
| Numbers, money | `numeric: true` (right-aligned, tabular figures); money in minor units through `formatNumber`, the unit in the header ("Đơn giá (đ)") |
| Status | Last before the menu: the document status (`DocumentStatus`), then any module progress badge; each `role: 'status'` |
| Results (net pay, total) | Last of the figures, `strong: true` |

- Each column declares its `role` for the card layout below 1024px: `title`, `status`, `meta` (a line under the title) or `hidden`.
- `sortable: true` only where the API sorts; sorting is server-side; the header button toggles ascending/descending.
- Headers hold only the label and the sort button; no filters or buttons in headers.

## Rows

| Interaction | Behaviour |
| --- | --- |
| Hover | Tint (never the only sign of anything) |
| Click on the identifier link | Opens the record's page; middle click and Ctrl+click open a new tab |
| Click elsewhere on the row | Follows `href`, or calls `onRowClick` when given: the list's side panel, or the edit modal of a settings row |
| Keyboard | Tab reaches the identifier (a link, or a button with `onRowClick` and no `href`), Enter opens it |
| "…" at the row end (`menu`) | Secondary actions as `Menu.Item`; only when a row has two or more actions; never a button per row |

### Side panel

A list whose records are often checked without being changed opens a read-only preview on row click:

```tsx
const peek = usePeek() // ?peek=<id> on the URL; Back closes it
const open = rows.find((r) => String(r.id) === peek.id)
…
<DataTable … href={(r) => `/hrm/leaves/${r.id}`} onRowClick={(r) => peek.open(r.id)} />
<SidePanel title={open?.number ?? ''} href={`/hrm/leaves/${open?.id}`} opened={!!open} onClose={peek.close}>
  {open && <LeavePreview id={open.id} />} {/* the preview the area declares in its manifest for the inbox */}
</SidePanel>
```

The panel is 480px on the right, not modal (the list stays usable; another row switches it), closes with ✕, Esc or Back, and edits nothing: its "Open full page" button goes to the page.

## Selection and bulk actions

Only when the list has at least one bulk action:

```tsx
const [selected, setSelected] = useState<Set<string | number>>(new Set())
…
<ListPage
  filters={…}
  bulk={selected.size > 0 && (
    <BulkBar count={selected.size} onClear={() => setSelected(new Set())}>
      <Button variant="default" onClick={…}>{t('hrm.leaves.bulk_submit')}</Button> {/* a new key per bulk action */}
    </BulkBar>
  )}
>
  <DataTable … selection={{ selected, onChange: setSelected }} />
</ListPage>
```

- Offered only when the list's `/actions` include the bulk action. A checkbox column comes first; the header box selects or clears every row of the page (indeterminate when some are). Selection covers the visible page only: the page owns `selected` and clears it whenever filters, sort or page change (`useEffect(() => setSelected(new Set()), [params])`).
- While rows are selected, `BulkBar` replaces the filter bar: "N selected", the actions, Clear selection.
- A bulk action runs as a job (one document per transaction): confirm with the count when it changes data ("Gửi duyệt 12 đơn?"). Send every selected id, whatever its status: the job re-checks each document's permission and state, acts on those it can, and reports the skipped ones in its result. Once the job starts: clear the selection and `notifySuccess` that it started ("Đã bắt đầu gửi duyệt 12 đơn. Theo dõi ở màn hình tác vụ."); its end arrives as a notification. A failure to start shows as an `Alert` above the table.
- Wide screens only; below 1024px no selection.

## Paging

- Server-side, 50 rows by default, 20/50/100 to choose; total count on the left of the paging bar.
- Small admin lists (users, org units) load whole without paging.
- Feeds (notifications) use a "Load more" button.
- Reconciliation tables load whole inside a capped frame (`maxHeight="70dvh"`) so group and grand totals are never split across pages.

## Figures and totals

- Many number columns: group with `band` (two-level header, each band opening with a rule); results last with `strong`; zero shown as "–" in `dimmed`.
- Totals: `group={{ of, total }}` adds a row naming each group and a total row after it ("Tổng"); `footer` adds a grand total row that sticks to the bottom of a capped frame.

## Inline editing in cells

List tables never edit in place. Editing happens on the record's page or in its modal. Tables made for entry are their own components: `LinesTable` for a document's lines, `CellGrid` for a grid on two axes (timesheet).

## States

| State | Render |
| --- | --- |
| First load | `ContentSkeleton` in place of the table |
| Refetch (filter, page) | Keep the previous rows (`placeholderData: keepPreviousData`), no skeleton |
| Empty | `EmptyState`: what is missing and, when the user may create, a create button unless the header has one |
| Empty because of filters | `EmptyState` "No results" with a Clear filters button (`variant="default"`) |
| Error | `ErrorState` with Retry |

## Do / Don't

```tsx
// Do
{ key: 'total', header: t('sales.doc.total'), role: 'meta', numeric: true, render: (d) => formatNumber(d.total) }
<DataTable … onRowClick={(r) => setEditing(r)} />            // settings row with one action

// Don't
<Table>…</Table>                                               // Mantine Table in an area: lint fails
{ render: (d) => <Button size="xs">{t('…edit')}</Button> }     // a button per row: use menu
w={{ base: '100%', md: 190 }}                                  // a width outside filterWidth
render: (d) => d.total.toLocaleString()                        // formatting outside shared/i18n
```
