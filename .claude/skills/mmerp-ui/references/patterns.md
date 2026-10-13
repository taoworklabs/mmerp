# Screen patterns

Code recipes, one per screen type: which template, which components, in which order. Where things sit is in `layout.md`; the rules behind each part are in `tables.md`, `forms.md`, `actions.md` and `feedback.md`. Complete working screens to copy from are named under each recipe.

| The screen shows | Template | Recipe |
| --- | --- | --- |
| A list of documents or master data | `ListPage` | List page |
| One document | `DocumentPage` | Document page |
| One master-data record with tabs | `RecordPage` | Record page |
| A form that creates a record | `Page` + `Form` | Create page |
| A small configuration list | `ListPage` + `FormModal` | Settings list |
| Counts and entry points | `Page` + `TileGrid` | Dashboard |
| Figures across many rows, read-only | `ListPage` without paging | Report |
| A queue to work through | `InboxPage` | Inbox |
| Sign-in, outside the app shell | `AuthPage` | — |

## List page

Working example: `web/src/sales/pages/CustomersPage.tsx`; with a status filter and per-row actions, `web/src/sales/pages/ItemsPage.tsx`. Before writing a new list, check the area's `pages/` for one that already lists the same records (`hrm/pages/LeavesPage.tsx` lists leave requests).

```tsx
const sorts = ['code', '-code', 'name', '-name'] as const satisfies readonly CustomerSort[]
const defaults = { filters: { q: '', active: '' }, sort: 'code', sorts: [...sorts], pageSize: 50 }

export function CustomersPage() {
  const { t } = useTranslation()
  const canCreate = useCustomerActions().data?.includes('create') ?? false
  const [params, set, filterBar] = useListParams(defaults)         // filters, sort, page on the URL
  const { q, active } = params.filters as typeof defaults.filters
  const list = useQuery({ queryKey: salesKeys.customers.list({ … }), queryFn: () => unwrap(api.GET('/sales/customers', { … })), placeholderData: keepPreviousData })
  const columns: Column<CustomerListItem>[] = [ /* identifier first, status last: tables.md */ ]
  const view = useListView('sales.customers', columns)              // saved views and columns

  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.items.length === 0) body = /* EmptyState: feedback.md */
  else body = (
    <DataTable label={t('sales.customers.title')} columns={view.columns} onResize={view.onResize}
      rows={list.data.items} rowKey={(c) => c.id} href={(c) => `/sales/customers/${c.id}`}
      sort={params.sort} onSort={(sort) => set({ sort })} />
  )

  return (
    <ListPage
      title={t('sales.customers.title')}
      description={t('sales.customers.description')}
      action={canCreate && <Button component={Link} to="/sales/customers/new" leftSection={<IconPlus {...icon.button} />}>{t('sales.customers.create')}</Button>}
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={<>
        <SearchInput label={t('sales.customers.search')} placeholder={t('sales.customers.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
        <FilterChip label={t('sales.customer.status')} icon={IconCircleDot} data={…} {...filterBar.bind('active')} />
      </>}
      tools={view.tools}
      paging={list.data && { page: params.page, pageSize: params.pageSize, total: list.data.total, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) }}
    >
      {body}
    </ListPage>
  )
}
```

Adding to it, when the list needs them (`tables.md`):

- **Bulk actions**: offered when the list's `/actions` include the bulk action (e.g. `bulk_submit`); `selection` on `DataTable`, `bulk={selected.size > 0 && <BulkBar …>}` on `ListPage`, and `useEffect(() => setSelected(new Set()), [params])` so a new filter, sort or page starts empty. Send every selected id: the job re-checks each document and reports the ones it skipped.
- **Side panel**: `usePeek()` (from `@/shared/ui/SidePanel`), `onRowClick={(r) => peek.open(r.id)}`, and `<SidePanel>` as a child of `ListPage` after the table, holding the area's own preview component (`import LeavePreview from '../components/LeavePreview'`, the one its manifest gives the inbox).
- **Errors of list actions** (bulk start, undo failure): an `Alert` as the first child of `ListPage`, above the table.

## Create page

Working example: `web/src/sales/pages/NewCustomerPage.tsx` with `sales/components/CustomerForm.tsx`.

`Page centered` with a breadcrumb to the list → `Form` (`mode: 'onBlur'`, `fieldOf` for field error codes) → `FormSection`s → `FormActions` with `cancelTo` (the list), without `onCancel`. On success: invalidate the list's queries, `notifySuccess`, navigate to the new record (`replace: true`). The route checks the create permission and shows `ForbiddenPage` without it. Details in `forms.md`.

## Document page

Working example: `web/src/sales/pages/DocPage.tsx` with `sales/components/DocForm.tsx` (lines), `web/src/hrm/pages/LeavePage.tsx` (no lines).

```tsx
const doc = useDoc(kind, id)                                                       // the area's query hook
const [error, setError] = useState<string | null>(null)
const state = loaded(doc, t, { to: list, label: t('sales.quotes.back') })         // skeleton · not found with a link back · error
if (!state.ok) return state.fallback
const d = state.data
return (
  <DocumentPage
    title={d.number}
    breadcrumbs={[{ label: t('sales.quotes.title'), to: '/sales/quotes' }]}       // its own list
    description={<Group gap="xs" component="span"><span>{d.customer.name}</span><span aria-hidden>·</span><DocumentStatus docType={docType} status={d.status} /></Group>}
    actions={d.allowed_actions.length > 0 && (                                    // nothing to show without actions
      <Group gap="xs">
        {d.allowed_actions.includes('print') && (
          <ExportButton kind={`printing.${docType}`} params={{ id: d.id }} label={t('sales.quote.print')} icon={IconPrinter} />   // printing is a job
        )}
        <DocumentActions docType={docType} id={d.id} version={d.version} number={d.number} actions={d.allowed_actions} onError={setError}
          onDelete={async (version) => {                                           // the area's delete route, then back to the list
            await unwrap(api.DELETE('/sales/quotes/{id}', { params: { path: { id: d.id }, query: { version } } }))
            await qc.invalidateQueries({ queryKey: salesKeys.docs.all('quote') })
            navigate(list, { replace: true })
          }} />
      </Group>
    )}
    error={error}
    approval={<ApprovalPanel docType={docType} id={d.id} />}
    sections={[attachments, discussion]}                                           // useAttachmentSection, useDiscussionSection
    history={<DocumentHistory docType={docType} id={d.id} />}
    fullWidth                                                                      // only with a wide table (lines, grid, payroll)
  >
    <DocForm key={d.version} kind={kind} doc={d} />                                // view = edit; remounts on a new version
  </DocumentPage>
)
```

The page resolves its query with `loaded()` (skeleton, not found with a link back to the list, error). Lines: `LinesTable` in a `FormSection wide` (`forms.md`).

## Record page

Working example: `web/src/sales/pages/CustomerPage.tsx`, `web/src/hrm/pages/EmployeePage.tsx`.

`RecordPage` with `leading` (avatar or icon), a description of key facts, `tabs` (`{ value, label, content }`, the selected one on the URL). The first tab holds the record's form (view = edit) or, for values changed one at a time, `InlineEditField`s; other tabs hold `DataTable`s with their own `TabActions` (default-size buttons). Attachments and discussion are the last tabs, in a 480px reading column centred under the tabs.

## Settings list

Working example: `web/src/sales/pages/ItemsPage.tsx` (row click edits; the menu holds Edit and Retire, with Undo).

`ListPage` with "Create" in the header; `DataTable` with `onRowClick={(r) => setEditing(r)}` (no `href`: the first cell becomes the keyboard's button) and a `menu` only when rows have two or more actions; one `FormModal` for create and edit with `FormActions onCancel`; success toast, list invalidated. More than six fields or tables of its own: a page instead.

## Dashboard

Working example: `web/src/hrm/pages/OverviewPage.tsx`.

`Page` + `TileGrid` of `StatTile` (label, count, link to the list it counts; a failed count shows its error and Retry) and `LinkTile`.

## Report

`ListPage` without `paging`: `filters` (period, legal entity, org unit, on the URL), the export as its `action`; one `DataTable` with `band`ed columns, `group` totals, a `footer` grand total, no `maxHeight` (the list page's frame keeps the header and totals in view); zero as "–". Read-only, loaded whole (`tables.md`). The payroll table (`web/src/hrm/components/PayrollTable.tsx`) is the model for the columns, bands and totals; it sits on a document page, so it caps itself with `maxHeight="70dvh"` instead.

## Inbox

Working example: `web/src/core/pages/InboxPage.tsx`.

`InboxPage` with an `InboxList` of `InboxRow`s (number, type · submitter, relative time; unread bold) and the selected item's detail: the area's record preview, then `ApprovalPanel` with its decision bar. The selection is on the URL; after a decision the next item opens.

## Feed

Working example: `web/src/core/pages/NotificationsPage.tsx`.

`ListPage` with a framed `InboxList` that scrolls inside itself, newest first, under sticky day headings (`InboxGroup`: "Hôm nay", "Hôm qua", then the date in the tenant's time zone). Each `InboxRow` leads with the event's round icon in the colour of what it says (pending `warning`, done `success`, refused or failed `danger`, a mention `info`) and ends with the relative time (exact time in a tooltip) and the unread dot. Opening a row marks it read, then goes to its record. "Đánh dấu tất cả đã đọc" is the header action; older items load under the frame.

