# Screen patterns

Code recipes, one per screen type: which template, which components, in which order. Where things sit is in `layout.md`; the rules behind each part are in `tables.md`, `forms.md`, `actions.md` and `feedback.md`. Complete working screens to copy from are named under each recipe.

| The screen shows | Template | Recipe |
| --- | --- | --- |
| A list of documents or master data | `ListPage` | List page |
| One document | `DocumentPage` | Document page |
| One master-data record (an employee, a customer, a user) | `RecordPage` | Record page |
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
      list={{
        query: list,                                                // loading, error, empty and the table come from ListPage
        table: { label: t('sales.customers.title'), columns: view.columns, onResize: view.onResize, rowKey: (c) => c.id,
          href: (c) => `/sales/customers/${c.id}`, sort: params.sort, onSort: (sort) => set({ sort }) },
        empty: { title: t('sales.customers.empty'), description: t('sales.customers.empty_description') },
        noResults: t('sales.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    />
  )
}
```

Adding to it, when the list needs them (`tables.md`):

- **Bulk actions**: offered when the list's `/actions` include the bulk action (e.g. `bulk_submit`); `selection` in `list.table`, `bulk={selected.size > 0 && <BulkBar …>}` on `ListPage`, and `useEffect(() => setSelected(new Set()), [params])` so a new filter, sort or page starts empty. Send every selected id: the job re-checks each document and reports the ones it skipped.
- **Document status filter**: `<DocumentStatusFilter docType={…} {...filterBar.bind('status')} />` (`@/shared/document`) offers the four statuses, labelled as the document type names them.
- **Side panel**: `usePeek()` (from `@/shared/ui/SidePanel`), `onRowClick: (r) => peek.open(r.id)` in `list.table`, and `<SidePanel>` as a child of `ListPage`, holding the area's own preview component (`import LeavePreview from '../components/LeavePreview'`, the one its manifest gives the inbox).
- **Errors of list actions** (bulk start, undo failure): an `ErrorState` as a child of `ListPage` (children come before the table).

## Create page

Working example: `web/src/sales/pages/NewCustomerPage.tsx` with `sales/components/CustomerForm.tsx`.

`Page centered` with a breadcrumb to the list → `Form` (`mode: 'onBlur'`, `errorFields` for field error codes) → `FormSection`s → `FormActions` with `cancelTo` (the list), without `onCancel`. On success: `useInvalidateRecord()` for the new record, `notifySuccess`, navigate to the new record (`replace: true`). The route checks the create permission and shows `ForbiddenPage` without it. Details in `forms.md`.

## Document page

Working example: `web/src/hrm/pages/LeavePage.tsx` with `hrm/components/LeaveDetails.tsx` (values in `EditablePanel` cards, the side column), `web/src/sales/pages/DocPage.tsx` with `sales/components/DocForm.tsx` (lines, `fullWidth`).

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
    actions={
      <Group gap="xs">
        <ApprovalDecision docType={docType} id={d.id} onError={setError} />        // Từ chối · Duyệt, for the actor at the current step
        <DocumentActions docType={docType} id={d.id} version={d.version} number={d.number} actions={d.allowed_actions} onError={setError}
          listPath={list} printLabel={t('sales.quote.print')} />                  // print (a job, template printing.<docType> unless printKind) when allowed; delete through the core route, then back to this list
      </Group>
    }
    error={error}
    approval={<ApprovalPanel docType={docType} id={d.id} decideElsewhere />}             // steps only; decisions are in the header; `subject` names who it is for (leave)
    sections={[attachments]}                                                       // useAttachmentSection
    activity={<ActivityFeed docType={docType} id={d.id} />}
    fullWidth                                                                      // only with a wide table (lines, grid, payroll)
  >
    <DocForm key={d.version} kind={kind} doc={d} />                                // view = edit; remounts on a new version
  </DocumentPage>
)
```

The page resolves its query with `loaded()` (skeleton, not found with a link back to the list, error). Without `fullWidth` the children are `EditablePanel` cards (`LeaveDetails`: `editable={allowed_actions.includes('edit')}`, `edit={(close) => <LeaveForm … onClose={close} />}`), keyed by `version`. Lines: `LinesTable` in a `FormSection wide` on a `fullWidth` page (`forms.md`). The person a document is for is an `EmployeeLink` in the description.

## Record page

Working examples: `web/src/hrm/pages/EmployeePage.tsx` with `hrm/components/EmployeeSummary.tsx` (key facts), `EmployeeProfile.tsx` (cards) and `EmployeeDigest.tsx` (right column); `web/src/sales/pages/CustomerPage.tsx` (a customer's quotations and orders as tabs); `web/src/core/pages/UserPage.tsx` (no activity; the main action, "Cấp vai trò", in the header).

```tsx
<RecordPage
  title={e.full_name}
  breadcrumbs={[{ label: t('hrm.employees.title'), to: '/hrm/employees' }]}        // ends with the page; no back button
  leading={<PersonAvatar name={e.full_name} size="lg" />}
  status={<EmployeeStatus status={e.status} />}                                    // beside the title
  description={[e.code, e.org_unit_name, e.user_login].filter(Boolean).join(' · ')}
  summary={<EmployeeSummary … />}                                                  // KeyFacts, 4–5 facts
  actions={/* secondary buttons */}
  more={/* MenuAction children; the destructive one `danger`, confirmed */}
  tabs={[
    { value: 'overview', label: t('shared.record.overview'), content: <RecordColumns main={<EmployeeProfile … />} side={<EmployeeDigest … />} /> },
    { value: 'contracts', label: t('…_count', { count }), content: <ContractsTab … /> }, // full-width DataTable
    …, attachments,
    { value: 'activity', label: t('shared.document.tab.activity'), content: <ActivityFeed … /> },
  ]}
  aliases={{ history: 'activity', discussion: 'activity' }}                         // old ?tab= links
>
  {move && <MoveModal … />}
</RecordPage>
```

The overview's left column holds the `EditablePanel` cards, pairs `FieldList stacked` (label over value, two to a row, an address `'wide'`); each card shows every value its form edits, those also in the header (name, code, manager) included, so what can be changed is where it is read. An empty value is a `MissingValue` (`children` of `EditablePanel` as a function receives `open`). The right column holds digests, each a `Panel plain` (no frame: the values in cards read first; `RecordColumns` parts them with rules) with a "Xem tất cả" link to its tab: the current item, a figure, and `ActivityFeed recent={5}` (comments and changes only, no filter) under "Hoạt động gần đây". A part the user may not read is left out, not shown empty.

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

