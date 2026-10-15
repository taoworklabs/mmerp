# Tables and lists

Every table is `DataTable` (`@/shared/ui/DataTable`); a list screen is `ListPage` around it. Editable document lines are `LinesTable` (`forms.md`); a grid of cells on two axes is `CellGrid`.

## Anatomy

```
[🔍 Search      ] [Status: Posted ✕] [Org unit ▾] [Month ▾]  Clear filters   [Views ▾] [Columns]
  ☐   Identifier ↕   Text (wraps if short of room)   Date ↕   Amount ↕   Status       header 36px, header tone, 600; framed inside a page, bare as a list page's content
 ──────────────────────────────────────────────────────────────────────────
  ☐   DH-0001        Customer                     01/10       12.500   [Posted]  ⋯   rows 40px, hover tint
 ─────────────────────────────────────────────────────────────────────
1.234 records                     ‹ 1 2 3 ›                   50 / page ▾  ← paging
```

## Filter bar

- The search box, then one **filter chip** per filter, `xs` (8px) apart, 30px tall; saved views and columns (`tools`) sit at the bar's right end. Order: search, then the record's own choices (status, type), then its dimensions (org unit, legal entity), then the period.
- A chip is a `default` button named after the filter, with the 18px icon of any page-level button (it takes the icon component and sizes it) and a 16px chevron. Picking a value turns it into "Name: value", tinted (`variant="light"`), with its own ✕ that clears just that filter; a value too long for the chip (320px at most) ends in "…" and shows whole on hover. A filter that always holds a value (the legal entity of settings, a calendar's year: `clearable={false}`) keeps the plain look and has no ✕.
- An on/off filter ("Expiring in 30 days", "Only failed") is a `Button` that turns `variant="light"` with `aria-pressed` while on.
- While any filter is set, "Clear filters" (a `default` button) ends the chips: the page passes `onClearFilters` only then, and it resets every filter, search included. `useListParams` gives the page a `filterBar`: `active` (a filter is off its default), `clear`, and `bind(key)`, the `value`/`onChange` pair a chip takes.
- When the bar is full the search box narrows first (down to 160px); chips keep their full text and wrap to a second line rather than truncate. Below 1024px the search box takes a line of its own and the chips wrap under it, on the page; there is no filter drawer.

| Filter | Component |
| --- | --- |
| Search | `SearchInput` (applies on Enter or blur; `/` focuses it) |
| Status, type, any choice | `FilterChip` with `data`; `searchable` for long lists |
| Org unit | `OrgUnitFilter product="hrm" permission="hrm.leave.view"` (only units where the user has that permission, as a tree, with a search) |
| Legal entity | `FilterChip` with the entities |
| Month | `MonthFilter` (value `YYYY-MM`, shown 08/2026) |
| Year | `YearFilter` (always holds a year) |

- Every business list has a search and sorts on its identifier and date; documents also filter by status and their key dimension (employee, org unit, period).
- Filters, sort, page and page size live in the URL (`useListParams`); copying the URL reopens the same list. Changing a filter goes back to page 1.

```tsx
<ListPage
  …
  onClearFilters={filterBar.active ? filterBar.clear : undefined}
  filters={
    <>
      <SearchInput label={t('sales.customers.search')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
      <FilterChip
        label={t('sales.customer.status')}
        icon={IconCircleDot}
        data={…}
        {...filterBar.bind('active')}
      />
    </>
  }
>
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

Every column has a `kind`, what it holds (`text`, the default, is left out); `DataTable` sets the look and width from it, so the eye goes to names and figures first:

| `kind` | Holds | Look | Width |
| --- | --- | --- | --- |
| `text` (default) | Names of things, types, descriptions | Main text colour | Wraps when the table runs short |
| `person` | `PersonName` (avatar + name) | Main text colour | One line |
| `code` | Document numbers, employee or customer codes, logins, tax codes, phones | `dimmed`, tabular figures | One line |
| `date` | Dates, periods, months (`formatDate`, `formatMonthShort`: 08/2026), date and time | `dimmed`, tabular figures | One line |
| `number` | Quantities, money, rates | Right-aligned, tabular figures; money in minor units through `formatNumber`, the unit in the header ("Đơn giá (đ)") | One line |
| `status` | `DocumentStatus`, `StatusBadge` | The badge | One line |

Spare room goes to every column in proportion to its content, so the gaps between columns stay even.

| Column | Rule |
| --- | --- |
| Identifier | First, `role: 'title'`, sticky on horizontal scroll (its edge rule shows only then), never wraps, links the row (`href`); weight 500 in the main text colour whatever its kind (`code` for a number, `person` for a name), linked or not. Spare room is shared by every column in proportion to its content, so the gaps stay even |
| Status | Last before the menu, `role: 'status'`, `kind: 'status'`: the document status (`DocumentStatus`), with any module progress badge beside it in the same cell (`<Group gap="xs" wrap="nowrap">`, see `sales/pages/DocsPage.tsx`) |
| Results (net pay, total) | Last of the figures, `strong: true` |

- A zero in a `number` column and an empty cell show "–" in `dimmed` (a total row leaves an empty cell blank), so the eye stops on cells that hold something; render `''`, never a dash of your own. A width the user dragged wins over the kind's.
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

- Server-side, 50 rows by default, 20/50/100 to choose; one bar under the table: total count on the left, pages in the centre (ghost buttons, the current page filled primary), the page size on the right as a ghost select (no edge or fill until hovered).
- Small admin lists (users, org units) load whole without paging.
- Feeds (notifications) use a "Load more" button.
- Reconciliation tables load whole inside a capped scroll area (a list page's frame, or `maxHeight="70dvh"` on a document page) so group and grand totals are never split across pages.

## Figures and totals

- Many number columns: group with `band` (two-level header, each band opening with a rule); results last with `strong`; zero shown as "–" in `dimmed`.
- Totals: `group={{ of, total }}` adds a row naming each group and a total row after it ("Tổng"); `footer` adds a grand total row that sticks to the bottom of a capped scroll area.

## Inline editing in cells

List tables never edit in place. Editing happens on the record's page (its `EditablePanel` cards) or, for a settings row, in its modal. Tables made for entry are their own components: `LinesTable` for a document's lines, `CellGrid` for a grid on two axes (timesheet).

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
{ key: 'total', header: t('sales.doc.total'), role: 'meta', kind: 'number', render: (d) => formatNumber(d.total) }
<DataTable … onRowClick={(r) => setEditing(r)} />            // settings row with one action

// Don't
<Table>…</Table>                                               // Mantine Table in an area: lint fails
{ render: (d) => <Button>{t('…edit')}</Button> }     // a button per row: use menu
<Select aria-label={…} … />                                   // a Select in the filter bar: use FilterChip
render: (d) => d.total.toLocaleString()                        // formatting outside shared/i18n
```
