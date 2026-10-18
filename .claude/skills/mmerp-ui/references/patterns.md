# Screen patterns

Code recipes, one per screen type: which template, which components, in which order. Where things sit is in `layout.md`; the rules behind each part are in `tables.md`, `forms.md`, `actions.md` and `feedback.md`. Complete working screens to copy from are named under each recipe.

| The screen shows | Template | Recipe |
| --- | --- | --- |
| A list of documents or master data | `ListPage` | List page |
| One document | `DocumentPage` | Document page |
| One master-data record | `DetailPage` | Record page |
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
- **Errors of list actions** (bulk start, undo failure): an `Alert` as a child of `ListPage` (children come before the table).

## Create page

Working example: `web/src/sales/pages/NewCustomerPage.tsx` with `sales/components/CustomerForm.tsx`.

`Page centered` with a breadcrumb to the list → `Form` (`mode: 'onBlur'`, `fieldOf` for field error codes) → `FormSection`s → `FormActions` with `cancelTo` (the list), without `onCancel`. On success: invalidate the list's queries, `notifySuccess`, navigate to the new record (`replace: true`). The route checks the create permission and shows `ForbiddenPage` without it. Details in `forms.md`.

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

Working example: `web/src/hrm/pages/EmployeePage.tsx`, `web/src/sales/pages/CustomerPage.tsx`, `web/src/core/pages/UserPage.tsx`.

```tsx
<DetailPage
  title={e.full_name}
  breadcrumbs={[{ label: t('hrm.employees.title'), to: '/hrm/employees' }]}
  leading={<PersonAvatar name={e.full_name} size="lg" />}
  description={/* key facts joined by "·", then the status badge */}
  actions={/* at most three buttons */}
  more={/* Menu.Item children for the rest, optional */}
  side={<>
    <DetailTabs tabs={[{ value: 'contracts', label: …, content: <ContractsTab … /> }, …, attachments]} />   // first tab default, on the URL
    <Panel title={t('shared.document.tab.activity')}><ActivityFeed docType={employeeDocType} id={e.id} /></Panel>
  </>}
>
  <EmployeeProfile key={e.id} employee={e} onSaved={saved} />                       // EditablePanel cards
</DetailPage>
```

The main column is an area component (`EmployeeProfile`, `CustomerProfile`): one `EditablePanel` per group, its read view a `FieldList` showing every value its form edits (name, code, unit included). Its `edit` renders the group's form in the card: `Form` → `FormFields` (two fields to a row) with the same field groups as the create form (`MainFields`, `GeneralFields`…) → `FormActions onCancel={close}`, from a form hook holding the whole record (`useEmployeeForm`, `useCustomerForm`), since the API saves the record as a whole. No `FormModal` for a group; a modal stays for a separate action (`MoveModal`: transfer, terminate). A tab holds a `DataTable` with `cards` and its own `TabActions`. A record with no lists of its own puts its attachments straight in a `Panel` (`CustomerPage`).

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

