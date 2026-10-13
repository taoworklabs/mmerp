# Actions

What the user can do, where it sits, how it confirms and how it answers. Which actions exist on a record is never decided by the screen: buttons come from the record's `allowed_actions`, collection actions (create, import, export, and bulk actions such as `bulk_submit`) from the list's `/actions` endpoint.

## Hierarchy

| Level | Look | Use |
| --- | --- | --- |
| Primary | `<Button>` (filled) | The one main action of a page, modal or region: Create, Save, Submit for approval, Approve. Rightmost |
| Secondary | `variant="default"` (outlined) | Other actions of the same place: Print, Export, Discard changes (asks first; the confirm is red), Retry |
| Quiet | `variant="subtle"` | Low-emphasis action among content: Add row, Change approver, Clear selection. The way out beside a primary button (Cancel in a form footer, "Quay lại" in a confirmation) is quiet grey: `variant="subtle" color="gray"` (`FormActions` and `useConfirm` do it) |
| Destructive | `variant="subtle" color="danger"` (red text) | Delete a draft, Cancel a document; always asks first |
| Destructive confirm | `color="danger"` (filled red) | Only the confirm button of a confirmation dialog for a destructive step |
| More | `MoreActions` ("…") | What does not earn a button |

- **One primary per place.** A page header has at most one; a modal's footer one; a section none or one.
- **At most three buttons** in a page header; the rest go into `MoreActions place="page"`, placed before the primary button. A table row puts every secondary action into its `menu` (`MoreActions` on the row) when it has two or more; a row with one action opens it on click.
- **One size** (`components.md`): every button is the default `sm`, 30px, wherever it sits; a lesser action is quieter in style, not smaller.
- Buttons are right-aligned in the page header, `TabActions`, section headers and footers.

```tsx
<Page
  title={d.number}
  actions={
    <Group gap="xs">
      <ExportButton … />                         {/* secondary */}
      <MoreActions place="page">
        <Menu.Item leftSection={<IconShoppingCart {...icon.text} />} onClick={createOrder}>{t('sales.quote.create_order')}</Menu.Item>
      </MoreActions>
      <DocumentActions … />                      {/* Delete (red text) · Withdraw · Submit (primary) */}
    </Group>
  }
>
```

## Lifecycle (status changes)

A document moves between `draft`, `pending_approval`, `posted`, `cancelled` only through `DocumentActions`, which draws a verb button for each transition in `allowed_actions`:

| Button | Look | Confirms |
| --- | --- | --- |
| Submit for approval ("Gửi duyệt") | Primary | No |
| Withdraw ("Rút lại") | Secondary | No |
| Delete draft ("Xóa nháp") | Red text | Yes, names the document |
| Cancel document ("Hủy chứng từ") | Red text | Yes, names the document |
| Approve / Reject | In `ApprovalPanel` and the inbox's decision bar; Reject asks for a reason | Reject yes |

- Status changes are explicit verb buttons, never a status dropdown: the next step depends on approval rules and permissions the server decides.
- When no lifecycle step is left, the area's follow-up action (Create order from a posted quotation) may be the page's primary button.
- Every lifecycle action answers with a toast on success and a banner at the top of the page on failure (`onError`).
- A document with unsaved edits must be saved before Submit or Withdraw: `DocumentActions` shows "Lưu thay đổi trước khi gửi duyệt hoặc rút lại." in the page banner instead of acting, while the save bar stays on screen. It knows through `UnsavedProvider` (set up by `DocumentPage`; every `Form` inside reports to it), so screens add nothing.

## Confirm or undo

| The action | Pattern |
| --- | --- |
| Cannot be taken back by the server: delete, cancel a document, reject, revoke a role, overwrite computed results (payroll compute), discard changes | **Confirm first** with `useConfirm`: title = the action, message names the object and the consequence, confirm button repeats the verb, `danger: true` for destructive |
| Writes a ledger line, an approval decision, a notification, or anything audited as a business event | Confirm when destructive, otherwise act; **never undo** |
| Reversible, personal or catalogue state with no side effects: mark read/unread, retire/reactivate a catalogue item, hide/show something | **Act at once, then `notifyUndo`** (5 seconds) calling the inverse operation |
| An edit inside an unsaved form (remove a line, clear a field) | Neither: the form's Discard covers it |

```tsx
const [ask, dialog] = useConfirm()
async function cancelDocument() {
  if (!(await ask({ title: t('shared.document.action.cancel'), message: t('shared.document.confirm.cancel', { number }), confirmLabel: t('shared.document.action.cancel'), danger: true }))) return
  await run(transition('cancelled'))
}
```

An undo needs an inverse the server already offers (reactivate is an edit with `active: true`); when there is none, the action confirms instead.

## Optimistic updates

Only for the reversible, side-effect-free actions above, and for personal display settings (`useListView` does it: a failed column change simply reverts, which the user sees; saving a named view reports its error in its own dialog). Documents, money and approvals always wait for the server: it computes numbers, totals and status.

```tsx
// The full version is the catalogue's Retire/Put back on sale in sales/pages/ItemsPage.tsx; toFields,
// markInactive and reactivate stand for the area's own helpers.
const retire = useMutation({
  mutationFn: (item: Item) => unwrap(api.PUT('/sales/items/{id}', { params: { path: { id: item.id } }, body: { ...toFields(item), active: false } })),
  onMutate: async (item) => {
    await qc.cancelQueries({ queryKey: salesKeys.items.all() })
    const before = qc.getQueriesData({ queryKey: salesKeys.items.all() })
    qc.setQueriesData({ queryKey: salesKeys.items.all() }, (old) => markInactive(old, item.id)) // the list shows it at once
    return { before }
  },
  onError: (err, _item, ctx) => {
    ctx?.before.forEach(([key, data]) => qc.setQueryData(key, data)) // put it back
    setError(errorText(t, err))                                      // banner at the top of the list
  },
  onSuccess: (_r, item) => notifyUndo(t('sales.item.retired', { name: item.name }), t('shared.undo'), () => reactivate.mutate(item)),
  onSettled: () => qc.invalidateQueries({ queryKey: salesKeys.items.all() }),
})
```

A failed optimistic action restores the previous state and shows its error as a banner on the page, like any action error; never a toast.

## Keyboard

The shortcuts, all of them:

| Key | Where | Does |
| --- | --- | --- |
| `/` | A list | Focuses its search box |
| Ctrl/Cmd+S | A form on a page | Saves |
| Esc | Modal, menu, drawer, side panel, inline edit | Closes or cancels |
| Enter | Inline edit | Saves |
| Enter | `LinesTable` cell | Next cell |
| Arrows, Enter | `CellGrid` (timesheet) | Move between cells |
| ←/→ | A focused column edge | Narrower/wider by 16px |

No other single-key or global shortcuts; approving or rejecting never has a key. Every action stays reachable by Tab and Enter, with the focus ring visible.

## Do / Don't

```tsx
// Do
<Button variant="subtle" color="danger" onClick={() => void remove()}>{t('hrm.calendar.delete')}</Button>  // asks inside remove()
<Button>{t('shared.document.action.submit')}</Button>                                                          // the one primary

// Don't
<Button color="teal" variant="outline">…</Button>                // invented variant
<Button color="danger" onClick={deleteNow}>…</Button>             // filled red outside a confirm dialog, no confirmation
<Select data={['draft', 'posted']} onChange={setStatus} />        // status changed by a dropdown
notifyUndo(t('…cancelled'), t('shared.undo'), uncancel)           // undo on a lifecycle action
```
