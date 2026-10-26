# Feedback

How a screen tells the user what is happening: loading, empty, errors, results. Every table, page and panel handles every state with the `shared/ui` component for it.

## Which channel

| Message | Channel | Stays |
| --- | --- | --- |
| A field's value is wrong | Under the field (`Form`) | Until fixed |
| A form or document cannot be saved for a reason that is not one field (period locked, conflict of rules) | `ErrorState` banner at the top of the form or page (announced) | Until the next attempt |
| An action on the page failed (lifecycle, delete, compute, bulk start, optimistic undo) | The page's banner: `DocumentPage`'s `error` prop; on a list, an `ErrorState` as the first child of `ListPage` | Until the next action |
| A region failed to load | `ErrorState` in place of the region, with Retry | Until retried |
| The data on screen needs attention (sources changed since compute, overtime over the limit) | `Notice tone="warning"` at the top of the region it is about | While true |
| The whole area cannot be changed (product off) | `ReadOnlyBanner`, shown by the app shell | While true |
| A save or action succeeded | `notifySuccess`: toast bottom right (top centre below 1024px), 3 seconds | 3 s |
| A reversible action took effect | `notifyUndo`: toast with Undo, 5 seconds | 5 s |
| A background job failed while the user was elsewhere | `notifyError`: toast that stays until closed | Until closed |
| The record changed under the user | "Data has changed · Reload" dialog (`useDocumentMutation`) | Until answered |
| Something must be answered now (delete?) | Confirmation dialog (`useConfirm`) | Until answered |

- **Errors of the current action never go in a toast**: they sit next to their cause, so they stay readable and do not vanish.
- **Every successful save, create and lifecycle action toasts**, except inline edits (the value itself shows the result).
- Every failed call is caught: an unhandled rejection shows nothing and is a bug.
- One banner of each kind per region; a new attempt clears the previous error (`onError(null)`).

```tsx
async function remove() {
  if (!(await ask({ … danger: true }))) return
  setError(null)
  try {
    await unwrap(api.DELETE(…))
    await qc.invalidateQueries({ queryKey: … })
    notifySuccess(t('hrm.calendar.saved'))
  } catch (err) {
    setError(errorText(t, err)) // rendered as an ErrorState at the top of the region
  }
}
```

## Loading

| Situation | Show |
| --- | --- |
| A page's code or data loads the first time | `PageSkeleton` (same frame as `Page`), `aria-busy` |
| A region loads (table, tab, panel) | `ContentSkeleton` in its place, no margins of its own |
| A list refetches (filter, sort, page) | The previous rows stay (`keepPreviousData`); no skeleton, no spinner |
| A button's action runs | The button is busy and cannot be pressed again; its spinner appears only after 300ms (`FormActions`, `loading` on `Button`) |
| Long work on the open record (compute a payroll, import into a timesheet) | A job, with a `Notice tone="info" icon={<Loader size="sm" />} announce="polite"` on that page while it runs |
| Long work on many records or a file (bulk action, export, print) | A job: a toast that it started; progress on the jobs screen; its end as a notification (`ExportButton` turns into a download link when the file is ready) |

Never a full-page spinner; never a spinner where a skeleton fits the layout.

## Empty

`EmptyState`: a dashed frame, a 24px icon in a 48px neutral circle, a short title, an optional description, and the next step.

| Case | Title | Action |
| --- | --- | --- |
| Nothing yet | What is missing ("Chưa có khách hàng") | The create button when the user may create and the page header does not already have it |
| Nothing matches the filters | "Không có kết quả" | "Xóa bộ lọc" (`variant="default"`) |
| An empty sub-table inside a page (no weeks yet, no dependants) | Same component, shorter text | Its section's add button if any |
| Out of scope or missing record | `NotFoundPage` with a link back to its list | — |

Two exceptions: the attachments panel shows its empty list as an 80px dashed frame that is itself the drop zone (click or drop files on it), and with files a quiet "+ Thêm file" under the list; an empty activity feed says so in one `dimmed` line under its comment box and filter.
| No permission | `ForbiddenPage`, chosen in the area's `routes.tsx` | — |

The description never depends on permissions; only the button does.

A list page gets these from `ListPage`'s `list`: it shows `ContentSkeleton`, `ErrorState` with Retry, the empty state, or, while a filter is set (`onClearFilters`), the no-results state with "Xóa bộ lọc" (`shared.list.clear_filters`). The screen passes only the texts:

```tsx
list={{ query: list, table: {…}, empty: { title: t('sales.customers.empty'), description: t('sales.customers.empty_description') }, noResults: t('sales.common.no_results'), paging: {…} }}
```

## Error levels

| Level | Component | Example |
| --- | --- | --- |
| Field | The field's error (`errorFields` for server codes) | "Mã đã tồn tại." under Code |
| Line of a table | The cell's field error (`lines.<i>.<field>`) | A quantity of 0 on line 3 |
| Section or form | `ErrorState` at the top of the form; the section's error count | "Kỳ tháng 3 đã khóa sổ…" |
| Region | `ErrorState` with Retry | A table that failed to load |
| Page | `ErrorState` (via `loaded()`), `NotFoundPage`, `ForbiddenPage` | A record that does not exist |

Error text comes from the API's code through `errorText(t, err)`; never a technical message, never the raw code.

## Do / Don't

```tsx
// Do
{error && <ErrorState message={error} />}
<QueryView query={list} empty={<EmptyState title={t('hrm.dependent.empty')} />}>{(rows) => <DataTable … rows={rows} />}</QueryView>
notifySuccess(t('sales.customer.created'))

// Don't
notifyError(errorText(t, err))           // the current action's error in a toast
await unwrap(api.DELETE(…))              // no try/catch: a failure shows nothing
{list.isPending && <Loader />}           // spinner where the table's skeleton belongs
<Text tone="dimmed">{t('…empty')}</Text>    // a hand-made empty state
if (list.isPending) body = <ContentSkeleton /> … // hand-written loading/error/empty: use QueryView
```
