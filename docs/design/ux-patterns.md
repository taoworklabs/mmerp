# UX patterns: audit and direction

Status: **decided on 2026-10-12.** Nothing in `web/src` has changed yet. Visual tokens and components are already settled in `design-direction.md` and the `mmerp-ui` skill; this document is about behaviour and structure. Every proposal below uses the tokens and components named in that skill; what does not exist yet is listed in section 6.

Reference: the public Atlassian Design System guidance (components: form, inline edit, modal dialog, dynamic table, flag, banner, empty state, page header) read on 2026-10-12, and the publicly visible structure of Jira. Only behavioural and structural principles are taken; no asset, illustration or copy.

## 1. Audit of the current screens

### Screen inventory

| Type | Screens | Template |
| --- | --- | --- |
| Home, overview | `/`, `/hrm/overview` | `Page` + `TileGrid` |
| List of documents | leaves, overtimes, contracts, timesheets, payrolls, quotations, orders | `ListPage` + `DataTable` |
| List of master data | employees, customers, items, users | `ListPage` + `DataTable` |
| Document detail (view = edit) | leave, overtime, contract, quotation, order | `DocumentPage` (360px side column) |
| Wide document (grid / reconciliation) | timesheet (`CellGrid`), payroll (banded `DataTable`) | `DocumentPage fullWidth` (side column as tabs) |
| Record detail | employee, customer, user | `RecordPage` (tabs) |
| Create page | employee, contract, leave, overtime, customer, quotation, order, approval rule | `Page` + `Form` |
| Create modal | timesheet, payroll, item, user, org unit, role grant | `FormModal` |
| Settings / configuration | leave types, contract types, legal parameters, work calendar, org units, period locks, settings, mail server, print templates, approval rules | `ListPage` + `FormModal` (rules: own page) |
| Queue | approval inbox, notifications | `InboxPage`, `ListPage` + `InboxList` |
| Jobs | jobs | `ListPage` |
| Sign-in | login | `AuthPage` |
| Report | none yet (payroll is the closest) | — |

### Layout

- **App shell:** header 48px (home logo, area name, notifications with unread count polled every minute, inbox with pending count not polled, jobs, account menu). Navigation 240/56px holds only the current area's menu, grouped, groups collapsible and remembered per user; collapses on its own below 1280px until the user chooses; drawer below 1024px. Areas are switched only through the home page. No global search, no area switcher.
- **Page header** (every template): breadcrumb, title, one-line description, actions right, divider. Actions wrap under the title below 768px.
- **Content width:** full width everywhere, no maximum. Form sections switch to a label column + two field columns from 1280px.
- **Document detail:** main column (the form) + 360px side column (approval, attachments, discussion, history); below 1024px the side column becomes URL tabs and actions a fixed bottom bar. Wide documents put approval and history in tabs at every size.

### Forms

- Labels above, `*` on required, help under the input; validation on blur and on submit (`mode: 'onBlur'`), errors under the field, API errors mapped to fields by code, the rest as a banner on top of the form.
- Failed submit: focus the first invalid field, error counts on section titles, "N to fix · next" in the footer.
- Page forms: sticky footer that appears only when dirty or after a failed submit (Unsaved · Discard · Save), Ctrl+S, leave-with-changes guard. Modal forms: Cancel · Save right-aligned.
- Viewing and editing a draft are the same page; read-only when `allowed_actions` lacks `edit`; the form is remounted on every new `version`.
- Repeated rows (contract allowances, quotation/order lines, payroll adjustments): each row is a block of stacked fields in the two-column section grid, "Add line" as an `xs` light button, a trash icon per row, no reorder, no confirmation.
- Quotation/order lines: item picker (server search, 20 results), quantity, unit price, discount %, VAT. **No line or document totals while editing**: totals are the server's stored values, shown only after saving, labelled "as of the last save".

### Tables

- Server-side sort and paging (50 by default; 20/50/100) on business lists; small admin lists load whole.
- Row click opens the record (the first cell is a real link). Secondary actions in a "…" row menu.
- No multi-select, no bulk actions, no column configuration, no inline cell edit, no truncation (first column sticky and unwrapped, other cells wrap).
- Numbers right-aligned with tabular figures; payroll uses banded headers, group and grand totals, a capped frame with sticky header/totals, zero as "–".
- The timesheet grid edits cells in place (arrows and Enter move between cells) and saves the whole grid at once.

### Actions

- Page header holds the actions; `DocumentActions` renders lifecycle buttons from `allowed_actions`: Delete and Cancel as outline red, Withdraw and Submit as default (outlined). Extra buttons (Print, Export, Import, Create order, Compute) sit beside them.
- Confirmation (`useConfirm`): deleting a draft, cancelling a document, revoking a role, deleting a calendar entry, a rule, a mail server, an attachment, discarding changes. No confirmation: withdraw, submit, create order from a quotation, payroll compute (recomputes results), removing a row from an unsaved form, retiring an item, clearing a period lock.
- No undo anywhere. Version conflicts open a "Data has changed · Reload" dialog.
- Shortcuts: `/` (list search), Ctrl/Cmd+S (save a form), Esc (close modal, menu, drawer). Arrows/Enter inside the timesheet grid.

### Feedback

- Loading: `PageSkeleton` for a page, `ContentSkeleton` for a region; payroll computation shows an alert with a loader while its job runs.
- Empty: `EmptyState` (dashed frame); filtered-empty offers Clear filters.
- Errors: field errors, form/page banner, `ErrorState` with Retry for failed loads.
- Success: `notifySuccess` toast bottom right, 3 seconds. Job failures: persistent toast.

### Inconsistencies found

| # | Same job, done differently | Evidence |
| --- | --- | --- |
| I1 | Create a document: page (leave, overtime, contract, quotation, order) vs modal then open (timesheet, payroll) | `NewLeavePage.tsx`, `TimesheetsPage.tsx:130`, `PayrollsPage.tsx:136` |
| I2 | Create/edit master data: page + in-place record (customer, employee) vs modal from a row menu (item) | `CustomerPage.tsx`, `ItemsPage.tsx:309` |
| I3 | Configuration edit: own page (approval rules) vs modal (all core settings) | `ApprovalRules.tsx`, `SettingsPage.tsx`, `PeriodLocksPage.tsx` |
| I4 | Success toast after save: employee, catalog modals, create pages toast; leave, overtime, contract, timesheet edits and timesheet/payroll creation do not | `EmployeePage.tsx:60` vs `LeaveForm.tsx:78`, `TimesheetGrid.tsx:44` |
| I5 | Action errors: page banner, inline `ErrorState`, toast, or nothing (unhandled rejection) | `DocPage.tsx:216`, `UserPage.tsx:184`, `NotificationsPage.tsx:34`, `ApprovalRules.tsx:152`, `WorkCalendarPage.tsx:46` |
| I6 | Lists: sort/search present on some, missing on others (timesheets, payrolls, users, items unsortable; contracts, leaves, overtimes without search or employee filter) | `TimesheetsPage.tsx:21`, `LeavesPage.tsx:97` |
| I7 | Row access: whole-row link vs a single action hidden in "…" (items, period locks, settings, print templates) | `CustomersPage.tsx:80` vs `ItemsPage.tsx:306` |
| I8 | Document page has no primary button; destructive buttons are outline red, a variant the skill does not list; other deletes are subtle red | `DocumentActions.tsx:57-72`, `MailServerPage.tsx:187` |
| I9 | Remove-row icon red (lines) vs grey (rule steps) | `DocForm.tsx:545`, `ApprovalRules.tsx:181` |
| I10 | `FormModal` "six fields at most" broken: mail server (7), payroll adjustments (4 per row × N), print template blocks | `MailServerPage.tsx:249`, `AdjustmentsModal.tsx`, `PrintTemplatesPage.tsx:370` |
| I11 | Empty sub-table: dimmed text vs `EmptyState` on the same page; empty-state description gated on create permission on some lists only | `WorkCalendarPage.tsx:118/134`, `ContractsPage.tsx:87` |
| I12 | Section headings written by hand (`Text fw={600}`) instead of a shared section | `PayrollPage.tsx:92`, `WorkCalendarPage.tsx:110` |
| I13 | Same data, different input: months as regex text vs `MonthField`; years as `DecimalField` vs `Select`; port as regex text vs integer field | `DependentsTab.tsx:166`, `BalancesTab.tsx:74`, `MailServerPage.tsx:250` |
| I14 | Read-only display: read-only inputs vs `FieldList` vs table | `CustomerForm`, `UserPage.tsx:242`, `DocForm.tsx:471` |
| I15 | Employee shown as name vs `code · name` | `LeavesPage.tsx:48` vs `ContractsPage.tsx:54` |
| I16 | Filter labels: `aria-label` + placeholder vs visible label; widths 180 vs 200; legal entity clearable vs not | `EmployeesPage.tsx:113/126`, `JobsPage.tsx:100`, `PayrollsPage.tsx:97`, `WorkCalendarPage.tsx:161` |
| I17 | Discount entered as % but shown as an amount when read-only | `DocForm.tsx:537/565` |
| I18 | Side sections: attachments + discussion on leave and contract, discussion only on overtime, timesheet, payroll | `LeavePage.tsx:57`, `OvertimePage.tsx:55` |
| I19 | Contract breadcrumb and Back go to the employee, though a contract list exists | `ContractPage.tsx:26` |
| I20 | Inbox count not refreshed, notification count polled | `AppLayout.tsx:47/53` |

Bugs found on the way (behaviour, not style):

- B1. Timesheet: the invalid-cell message is never shown (the grid ignores the field error) and the footer's "next invalid" has nothing to focus. `TimesheetGrid.tsx:66`.
- B2. Deleting a calendar entry or an approval rule that fails on the server shows nothing (no `try/catch`). `WorkCalendarPage.tsx:46`, `ApprovalRules.tsx:152`.
- B3. Submit/Withdraw in a document header ignore unsaved edits in the form below; the new `version` remounts the form and the edits are lost without warning. `DocPage.tsx:110`, `DocumentActions.tsx`.
- B4. Dependents put "relationship" in the `status` column role, so it shows as a status on small screens. `DependentsTab.tsx:60`.

## 2. Principles taken from the reference system

### Layout

- **Shell:** a top bar for global things (product switch, search, create, notifications, profile) and a collapsible left sidebar for the current product; content takes the rest. The sidebar remembers its state.
- **Page header:** breadcrumb, title, actions right-aligned next to the title, optionally a bar of search and filters under it. One page header per full page, never inside modals or drawers. Avoid truncating the title.
- **Detail view:** two columns, the content (title, description, child items, activity) left and a narrower context panel right with the properties that describe the item (status, assignee, dates) and secondary details folded away.
- **Where details open:** a full page when the item is the task (a deep link, long work); a side panel or modal when the user stays in a list and peeks or changes little ("open in sidebar"); a modal only for a short immediate task.
- **Width:** fluid layouts for lists and boards; reading and form content kept to a comfortable measure.

### Form

- Visible labels above fields; placeholders never replace labels (search with an icon and an accessible label is the exception). Helper text under the field.
- Required marked with an asterisk, with a legend when a form has required fields.
- Validate inline, near the field; errors next to what caused them; never disable the submit button, explain instead.
- Single column, fields sized to their expected content; related fields grouped in titled sections.
- Primary button for the main action, then a secondary or link-style Cancel; show progress on submit.
- **Inline edit** on detail screens for data changed often: the read view and the edit view look identical; click or Enter to edit, Enter saves, Esc cancels, visible confirm/cancel buttons; for long text, blur does not save or close, so work is not lost.
- **Create in a modal** from anywhere: the few fields needed to create, the rest after; a "create another" option for series entry; afterwards a flag with a link to the new item.
- Specialised inputs: searchable select, people picker (avatar + name), date picker, multi-select shown as tags.

### Table / list view

- Search and basic filters as dropdowns above the list, an advanced query mode, and saved filters that can be shared.
- Sortable column headers; no other controls in headers; columns can be chosen and reordered in list views.
- Rows: hover tint (never the only signal of selection or meaning), actions in a "…" menu at the row end, click to open; after a row is removed, focus moves to the next item.
- Multi-select reveals a bulk action bar; bulk changes run as a multi-step flow with a summary.
- Status as lozenges, people as avatar + name.
- Pagination when rows exceed a page; group-by and sticky headers in list views.
- Empty states differ in tone: first use (explain and invite), no results (neutral, suggest changing the filter).

### Action

- One primary action per page or region; others secondary or subtle; the rest in "…".
- Workflow transitions from a status control showing the next possible statuses; a transition that needs data opens a modal.
- Destructive actions: a modal that names the object and repeats the verb; quick reversible changes may instead confirm with a flag offering Undo.
- Feedback for actions: a flag (toast); critical or blocking problems as a banner or modal.
- Keyboard: shortcuts for frequent actions (create, search, next/previous, edit) and a help overlay listing them; a quick search or command palette.
- Drag and drop where order or column carries meaning (boards, ranking).

### Feedback

- Skeletons for content that is loading for the first time; spinners only inside the control that is working.
- Flag (toast) for confirmations and acknowledgements, auto-dismissing only when harmless; never auto-dismiss errors or critical warnings. Banner, one at a time and not dismissible, for page- or site-wide conditions. Inline message for something the user must act on in place. Modal for what must be answered now.
- Errors at three levels: under the field, as a message on the section, as a page-level state with a way out.

## 3. Mapping to the ERP

| Pattern | Apply as is? | ERP adjustment | mmerp today |
| --- | --- | --- | --- |
| Top bar + collapsible sidebar | Yes | Sidebar shows only the current area; areas switch through home | Done |
| Global create button | No | Creating needs an area context and a permission per record type; create stays on each list | — |
| Global quick search / command palette | Partly | Search across record types needs a core search API; useful for document numbers and people | Missing |
| Page header with actions right | Yes | Exactly as `Page` | Done |
| Two-column detail, properties right | Adjusted | ERP documents are long forms edited as a whole in draft; the right column carries approval, attachments, discussion, history instead of properties | Done (`DocumentPage`) |
| Detail in a side panel from a list | Adjusted | Only for reviewing without editing: the approval inbox preview. Records keep a full page (URL, history, approval) | Inbox only |
| Labels above, inline validation, never disable submit | Yes | — | Done |
| Single-column forms | Adjusted | Two columns from 1280px keep long HR forms short on wide screens; label column on the left | Two columns |
| Required legend | Adjusted | `*` is consistent everywhere; the legend adds noise on every form | Not shown |
| Inline edit of single fields | Adjusted | Draft documents are edited as a whole and saved with one `version`; posted documents are immutable; master data saves as a form. Per-field saves would multiply versions, audit rows and conflicts | Not used |
| Create in modal + create another | Adjusted | Only for short catalog/config forms; documents are created on a page because the form *is* the document | Mixed (I1–I3) |
| Search + basic filters + saved filters | Adjusted | Filters live in the URL, so a bookmark is already a saved filter; shared views need permissions-aware storage | Basic, uneven (I6) |
| Column chooser | Later | Columns are fixed per screen until custom fields arrive (then TanStack Table) | No |
| Row "…" menu, row click opens | Yes | Every list row with a detail opens it; rows without a detail open their edit modal on click | Mixed (I7) |
| Multi-select + bulk bar | Adjusted | Bulk operations are jobs (one document per transaction): select → choose action → confirm count → job → notification | Rule written, no component |
| Inline cell edit in tables | Adjusted | Only in grids built for entry (timesheet) and document lines; never in list views of documents | Timesheet only |
| Lozenge status, avatar + name | Yes | Fixed document colours; module progress neutral/info | Done |
| Pagination over infinite scroll | Yes | Pagination for lists (stable totals, URL page); "load more" only for feeds | Done |
| Group by, sticky header | Adjusted | Reconciliation tables: bands, group totals, sticky header and totals, zero as "–" | Payroll |
| Status dropdown for transitions | Adjusted | Few transitions, gated by approval and `allowed_actions`; explicit verb buttons (Submit, Withdraw, Cancel) are clearer than a status picker | Buttons |
| Undo instead of confirm | Adjusted | Posting, cancelling, approving write audit, notifications and posting lines and cannot be undone; undo fits only purely local, reversible changes | No undo |
| Optimistic update | No for documents | Server computes amounts, numbers and status; optimistic UI would show wrong totals. Fine for read flags (mark notification read) | Not used |
| Toast for success, banner for page state, inline for fields | Yes | Job failures stay until closed; period lock and read-only as banners | Done, uneven (I4, I5) |
| Skeleton first load, spinner in control | Yes | Long work (compute, import, print) is a job with progress, not a spinner | Done |
| Drag and drop | Rarely | Line order in quotations, approval steps; no boards in this ERP | No |

ERP patterns the reference system has no equivalent for, designed in the same spirit:

- **Document lines** with item picker, quantities, prices, discount, VAT, line totals and document totals; totals come from the server (single rounding function), so the screen shows them as of the last save or asks the server for a preview.
- **Reconciliation tables**: many number columns, banded headers, group and grand totals, capped frame.
- **Grid entry** (timesheet): cells on two axes, keyboard movement, whole-grid save.
- **Approval** with levels: side panel with the chain, decision bar, inbox with auto-advance.
- **Period locks, product disabled**: banners that explain why something cannot be edited.
- **Version conflicts**: reload dialog instead of silent overwrite.
- **Sensitive fields**: masked, revealed on demand, audited.
- **Jobs**: import, export, print, compute run in the background with progress and a notification.

## 4. Wireframes

### App shell + list page

```
┌──────────────────────────────────────────────────────────────────────────┐
│ ▣ Area name                                      🔔3  ⬚2  ⧗   (QV) Name ▾ │ 48px
├────────────┬─────────────────────────────────────────────────────────────┤
│ Overview   │ Area › …                                                    │
│ ▸ Group    │ Title                                    [Secondary] [Create]│
│   Item ◼   │ One-line description                                         │
│   Item     │ ─────────────────────────────────────────────────────────────│
│ ▸ Group    │ [🔍 Search…] [Status: Posted ✕] [Unit ▾] [Month ▾] Clear      │ filter bar (chips)
│   Item     │ ┌─────────────────────────────────────────────────────────┐ │
│            │ │ Identifier ↕ │ Text      │ Date ↕  │ Amount ↕│ Status │…│ │ 36px header
│            │ ├──────────────┼───────────┼─────────┼─────────┼────────┼─┤ │
│            │ │ DH-0001      │ Customer  │ 01/10   │ 12.500  │[Posted]│…│ │ 40px rows
│            │ │ …            │           │         │         │        │ │ │
│            │ └─────────────────────────────────────────────────────────┘ │
│ ⇤          │ 1.234 records            [50 / page ▾] ‹ 1 2 3 … 25 ›        │
└────────────┴─────────────────────────────────────────────────────────────┘
With rows selected (D-bulk): the filter bar becomes
│ ☑ 12 selected   [Bulk action ▾]   [Clear selection]                       │
```

### Document page (view = edit)

```
│ Quotations › BG-0012                                                      │
│ BG-0012                         [Print] [⋯]   [Delete] [Submit ▣primary]  │
│ Customer · Org unit · [Draft] [Ordered]                                   │
│ ──────────────────────────────────────────────────────────────────────────│
│ ┌ main column ───────────────────────────────┐ ┌ side column 360px ─────┐ │
│ │ [banner: period locked / document error]   │ │ Approval               │ │
│ │ General           field  field             │ │  level 1 ✓ name · time │ │
│ │                   field  field             │ │  level 2 … waiting     │ │
│ │ ───────────────────────────────────────────│ │ Attachments (2)        │ │
│ │ Lines                                      │ │ Discussion (5)         │ │
│ │ ┌ # │ Item ▾ │ Qty │ Price │ Disc% │ VAT │ Net │ ✕ ┐ │ History        │ │
│ │ │ 1 │        │     │       │       │     │     │   │ └────────────────┘ │
│ │ └ + Add line ─────────────── Subtotal / Discount / VAT / Total ┘        │
│ └────────────────────────────────────────────┘                            │
│ ▔▔ sticky footer when dirty: Unsaved changes · Discard        [Save] ▔▔▔▔ │
```

### Record page

```
│ Employees › Nguyễn Văn An                                                 │
│ (NA) Nguyễn Văn An                                    [Print] [Terminate] │
│      NV0001 · Kế toán · [Active]                                          │
│ [Info] [Dependants] [Balances] [Contracts] [Attachments] [Discussion]     │
│ ─────────────────────────────────────────────────────── [xs Add] (tab)    │
│ tab content: form sections, or a DataTable                                │
```

### Wide document (reconciliation)

```
│ Payrolls › BL-2026-09                     [Compute] [Export] [Submit]     │
│ 09/2026 · Legal entity · [Draft]                                          │
│ [Details] [Approval] [Discussion] [History]                               │
│ ┌ frame ≤ 70% of screen height, header and totals stick ──────────────┐   │
│ │ Employee │ Income ─────────────│ Insurance ─────│ Tax │ Net pay     │   │
│ │          │ Base │ Allow │ OT   │ BHXH │ BHYT    │     │             │   │
│ │ ▸ Department A                                                     │   │
│ │ NV0001   │ 12.000 │ 1.000 │ –  │ 960  │ 180     │ 0   │ 11.860      │   │
│ │ Total    │ …                                                       │   │
│ │ Grand total (sticky)                                               │   │
│ └────────────────────────────────────────────────────────────────────┘   │
```

### Create / edit modal (catalog, configuration)

```
┌ Create item ─────────────────────────────── ✕ ┐
│ Code *            Name *                     │
│ [            ]    [                       ]  │
│ Unit *            Unit price                 │
│ [      ▾]         [              ]           │
│ VAT rate                                     │
│ [      ▾]                                    │
│                         [Cancel] [Create item]│
└───────────────────────────────────────────────┘
```

### Approval inbox

```
│ Approval inbox                                                            │
│ ┌ list 360px ────────────┐│┌ preview ──────────────────────────────────┐ │
│ │ ◼ BG-0012 Quotation    │││ BG-0012 · Customer        [Open document] │ │
│ │   Nguyễn A · 2h ago    │││ Amount        125.000.000                 │ │
│ │   DH-0007 Order        │││ Max discount  12%                         │ │
│ │   Trần B · yesterday   │││ Approval: level 1 of 2                    │ │
│ │   NP-0031 Leave        │││ ───────────────────────────────────────── │ │
│ │                        │││ ▔ sticky: [Reject]              [Approve] │ │
│ └────────────────────────┘│└────────────────────────────────────────────┘ │
```

### Settings list + modal

```
│ Leave types                                                [Create]       │
│ ┌ Name │ Paid │ Days/year │ Status   │ … ┐                                │
│ │ Annual │ Yes │ 12 │ [In use]       │ … │  row click = edit modal         │
│ └────────────────────────────────────────┘                                │
```

## 5. Decisions

Each decision records its **Final decision**.

### U1. Where a record or document opens

| Option | Pros | Cons |
| --- | --- | --- |
| A. **Full page for every record and document; a read-only preview panel only in queues** (the inbox today) | Deep links, Back works, room for approval/history; matches how documents are worked on; no new component | Opening many records in a row means back-and-forth |
| B. Side panel ("peek") from lists for every type, full page on demand | Fast scanning of lists | Two views of each record to keep consistent; side panel too narrow for lines and long forms; editing in a panel conflicts with the page footer |
| C. Modal for master data, page for documents | Fast for small catalog records | Modal detail cannot hold tabs, attachments, history |

**Recommended: A**, plus row click on settings lists opening their edit modal (I7).

Final decision: A (full page; settings rows open their edit modal). **Reopened 2026-10-12:** plus a read-only side panel (peek) on lists that declare one: clicking a row outside its link opens the record's preview in a right-hand panel with a link to the full page; the identifier link still opens the page. Editing stays on the page.

### U2. How things are created

| Option | Pros | Cons |
| --- | --- | --- |
| A. Modal for everything | One pattern | Documents and long records do not fit; breaks view = edit |
| B. **Rule by size: `FormModal` when creation needs ≤ 6 fields and the result is then worked on its own page or lives in a list (timesheet, payroll, items, users, configuration); a page when the form is the record (documents, employees, customers)** | Matches most screens already; one sentence to apply | Two patterns to learn (but each predictable) |
| C. Page for everything | One pattern | Heavy for a 2-field catalog entry |

Sub-choice: add a "Create another" checkbox to catalog modals (items, leave types): **recommended no** until bulk catalog entry is a real complaint (import exists).

**Recommended: B**; fix I10 (mail server, adjustments move to a page or a smaller form).

Final decision: B (modal for ≤ 6 fields, page when the form is the record; no "Create another").

### U3. Inline edit

| Option | Pros | Cons |
| --- | --- | --- |
| A. **None for single fields; editing stays "view = edit" with one save** (today). Editable tables only where entry is the job: document lines and the timesheet grid | One save, one `version`, one audit entry per change; no per-field conflicts; works for financial data | Changing one field shows the sticky footer and needs a Save |
| B. Inline edit on detail pages of master data (customer, employee), per field save | Fast small edits | Each field a separate request, version bump and audit row; sensitive fields and permissions per field get complex |
| C. Inline edit in list cells | Fast mass correction | Wrong for documents (approval, period locks, posted immutability); risky for money |
| D. B and C | Most fluid | All the costs above |

Financial data (amounts, prices, rates, posted documents) is never inline-editable under any option. **Recommended: A.**

Final decision: A (no inline edit; view = edit with one save; editable tables only for document lines and the timesheet grid). **Reopened 2026-10-12:** inline edit of single fields is allowed on master-data record pages (customer, employee, item details), one field saved at a time; never on documents, money, rates or sensitive fields.

### U4. Dangerous actions

| Option | Pros | Cons |
| --- | --- | --- |
| A. Confirm every destructive or state-changing action, including removing a row from an unsaved form | Uniform | Friction on harmless edits; confirmations become noise and get clicked through |
| B. **Confirm what the server cannot take back (delete, cancel, reject, revoke, overwrite results such as payroll compute, discard changes); no confirm for edits inside an unsaved form (Discard covers them); no undo** | Matches the ledger nature of documents; small change from today (add compute) | Removing a line by mistake needs a re-type or Discard |
| C. Undo toast for reversible server actions (withdraw, retire item, mark read) instead of confirms | Fluid | Needs reverse endpoints and audit for each; little gain for rare actions |

**Recommended: B.**

Final decision: B (confirm what the server cannot take back; no confirm inside unsaved forms). **Reopened 2026-10-12:** reversible actions that write no ledger, audit-relevant decision or notification (mark read, retire/reactivate a catalog record) apply at once with an optimistic update and a toast offering Undo, instead of a confirmation. Lifecycle actions keep their confirmations and have no undo.

### U5. Paging large tables

| Option | Pros | Cons |
| --- | --- | --- |
| A. **Pagination for lists (today), "load more" only for feeds (notifications); reconciliation tables load whole inside a capped frame, row virtualisation added when a payroll of thousands proves slow** | URL keeps the page; totals are exact; no new component now | Payroll of 3,000 rows renders all rows |
| B. Infinite scroll for lists | Smooth scanning | Loses page in URL, footer unreachable, harder to share a position |
| C. Pagination everywhere including reconciliation | Light pages | Group totals and grand totals split across pages: wrong for reconciliation |

**Recommended: A.**

Final decision: A (pagination; "load more" for feeds; reconciliation tables whole in a capped frame).

### U6. Advanced filtering

| Option | Pros | Cons |
| --- | --- | --- |
| A. Condition builder (field, operator, value, and/or) | Powerful | Big component and query API per list; few users of an SMB ERP need it |
| B. Basic filters + saved views (named, per user) | Covers repeat work | Storage, permission checks when sharing |
| C. **Basic filters made consistent on every list (search + status + the list's key dimension: employee, org unit, period); the URL is the saved filter (bookmark, share a link)** | No new storage; fixes I6 | No named views inside the app |

**Recommended: C now, B when users ask.**

Final decision: C (consistent basic filters; the URL is the saved filter). **Reopened 2026-10-12:** plus named views per user, saved on the server (ADR-0031): a view stores the list's query string.

### U7. Column configuration and saved views: who owns them

| Option | Pros | Cons |
| --- | --- | --- |
| A. **None until custom fields arrive** (fixed columns designed per screen) | Nothing to build; screens stay designed | Users cannot hide columns |
| B. Per user | Personal | Every screen needs defaults + reset; storage |
| C. Per organisation (admin sets) | Consistent for a team | Admin UI to build |
| D. Both: organisation default, user override | Complete | Most work |

When it comes (with custom fields), **D** stored server-side. **Recommended now: A.**

Final decision: A (none until custom fields; then organisation default + user override, server-side). **Reopened 2026-10-12:** per user, on the server (ADR-0031): show/hide columns (the identifier always shows) and drag to resize; no organisation default yet.

### U8. Keyboard shortcuts

| Option | Pros | Cons |
| --- | --- | --- |
| A. **Keep today's set** (`/` search, Ctrl/Cmd+S save, Esc close, arrows/Enter in grids) and add `?` to show them | Nothing to learn by accident; tiny work | Approvers working a long queue click a lot |
| B. Basic set: A + `c` create on a list, `j`/`k` next/previous in lists and the inbox, Enter to open | Faster for heavy users | Must avoid typing conflicts; needs the help overlay and tests |
| C. Full set with a command palette | Power-user speed | Needs global search API; large |
| D. Later | — | — |

Approve/Reject by a single key is excluded under every option (approval is a business decision). **Recommended: A**, B for the inbox only if approvers ask.

Final decision: D (later: keep today's set, add nothing, including `?`).

### U9. Table density

| Option | Pros | Cons |
| --- | --- | --- |
| A. **One density, 40px rows, no user switch (as decided in D5); reconciliation tables keep 40px** | Simple; consistent | Payroll shows fewer rows per screen |
| B. 40px default; a `dense` (32px) variant set by the screen for reconciliation tables only | More numbers visible where it matters | A second row height to test |
| C. User toggle compact/default | Choice | Every table tested twice |

**Recommended: A**, B if payroll review proves cramped.

Final decision: A (one density, 40px, no switch).

### U10. Primary action on a document page

| Option | Pros | Cons |
| --- | --- | --- |
| A. **The next lifecycle step is the primary button** (Submit on a draft; in the inbox Approve); Delete and Cancel leave the outline-red style (see sub-choice); other actions (Print, Export, Create order) stay secondary, moving under "…" when there are more than three | One primary per page, as the skill says; fixes I8 | Changes `DocumentActions` |
| B. Keep all outlined (today) | No change | No primary; red outline variant outside the skill |

Sub-choice for destructive lifecycle buttons: subtle red text button (skill's "Danger text") **recommended**, or keep outline red and add it to the skill.

**Recommended: A + subtle red.**

Final decision: A + subtle red (next lifecycle step is primary; Delete/Cancel as red text buttons; "…" past three actions).

### U11. Document lines editing

| Option | Pros | Cons |
| --- | --- | --- |
| A. Stacked fields per line (today) | Exists | Hard to scan more than 3 lines; no totals while editing |
| B. **Editable lines table**: one row per line (item, description, qty, price, discount %, VAT, net), Tab/Enter move, add row at the end, totals footer showing the totals as of the last save plus a "recalculated on save" note | Scannable, standard for ERP; reuses grid keyboard model | New component (`LinesTable`); narrow screens need the stacked form |
| C. B + live totals from a server preview endpoint (no save, debounced) | Correct live totals with the single rounding function | New endpoint per document type |

Client-side computation of totals is excluded: rounding lives only on the server.

**Recommended: B now, C when users ask for live totals.**

Final decision: B (editable `LinesTable`, totals as of the last save).

### U12. Content width

| Option | Pros | Cons |
| --- | --- | --- |
| A. Full width everywhere (today) | Tables use the whole screen | Forms on a 1920px screen stretch fields far apart |
| B. **Full width for lists and wide documents; form pages and document main columns capped (new token, e.g. 1200px)** | Readable forms on large screens | One more size token |
| C. Cap everything | Calm layout | Wastes width for tables |

**Recommended: B.**

Final decision: B (full width for lists and wide documents; forms and the document main column capped at 1200px, `--mmerp-content-width`).

### U13. Consistency fixes (one yes/no each)

| # | Fix | Recommendation | Final decision |
| --- | --- | --- | --- |
| F1 | Success toast after every successful save, create and lifecycle action (I4) | Yes | Yes |
| F2 | One rule for action errors: form or page banner for the current action, never a toast; always caught (I5, B2) | Yes | Yes |
| F3 | Every business list has search and sort on its identifier and date (I6) | Yes | Yes |
| F4 | Settings rows open their edit modal on click; "…" only when there are two or more actions (I7) | Yes | Yes |
| F5 | Remove-row icon: grey subtle `ActionIcon` everywhere (I9) | Yes | Yes |
| F6 | Empty sub-tables use `EmptyState`; empty-state description never depends on permission, only the button does (I11) | Yes | Yes |
| F7 | A shared section component replaces hand-written headings (I12) | Yes | Yes |
| F8 | Same data, same input: `MonthField`, a year select, an integer field (I13) | Yes | Yes |
| F9 | Read-only values: `FieldList` for record facts; read-only inputs only on a draft the user cannot edit (I14) | Yes | Yes |
| F10 | People in tables shown as `PersonName`; documents list `code · name` only where codes matter (I15) | Yes | Yes |
| F11 | Filters: placeholder + `aria-label`, never visible labels; one width per kind (I16) | Yes | Yes: search 280, select 200, org unit / legal entity 240, month 160, year 120 (`filterWidth` in `theme.ts`). Superseded: filters are now chips sized by their content, only the search box keeps a width (`ui.md`, filter bar) |
| F12 | Discount shown the way it is entered (%), amount in its own column (I17) | Yes | Yes |
| F13 | Attachments on every document type that may carry files (I18) | Decide per type with the product owner | Add attachments to overtime; timesheet and payroll stay without (system-generated data; payroll files would carry everyone's pay) |
| F14 | A document's breadcrumb points to its own list (I19) | Yes | Yes |
| F15 | Inbox count refreshed like the notification count (I20) | Yes | Yes |
| F16 | Fix bugs B1–B4 | Yes | Yes |

### U14. Conflicts with current business flows

| # | Conflict | Recommendation | Final decision |
| --- | --- | --- | --- |
| C1 | Header Submit while the form below has unsaved edits (B3) | Submit saves first, or is disabled with "Save changes first" while dirty; recommend: block with an explanation | Block with an explanation: while the form is dirty, Submit/Withdraw say "Save changes first" and point to the save bar |
| C2 | Undo vs ledger: posted, cancelled and approved states write audit, notifications and posting lines | No undo for lifecycle actions (U4) | Accepted |
| C3 | Inline edit vs document `version` and approval snapshot | No per-field edit on documents (U3) | Accepted |
| C4 | Live totals vs single rounding function | Totals only from the server (U11) | Accepted |
| C5 | Bulk actions vs "one transaction, one document" | Bulk = a job, results in the jobs screen and a notification | Accepted |
| C6 | Global create button vs per-area permissions | No global create (section 3) | Accepted |
| C7 | Status dropdown vs approval gate | Explicit verb buttons from `allowed_actions` (section 3) | Accepted |
| C8 | `FormModal` limit vs payroll adjustments (rows × 4 fields) | Adjustments become a section of the payroll page or a lines table | Accepted |

## 6. Components to add

Only the ones the recommended options need; each goes into `shared/ui` with a `/dev/ui` entry and a row in the skill's `components.md`.

| Component | Needed by | Description |
| --- | --- | --- |
| `LinesTable` | U11 | Editable table of document lines on top of React Hook Form field arrays: one row per line, typed cells (picker, `DecimalField`, `MoneyField`, select), keyboard movement, add/remove row (grey remove icon), totals footer slot; stacked layout below 1024px |
| `Section` | F7 | Titled region inside a page (title h2, optional description and `xs` actions right), unframed, ruled from the previous one; replaces hand-written headings |
| `BulkBar` + `DataTable` selection | Bulk actions (when the first one is built) | Checkbox column, "N selected · action · clear", replaces the filter bar; actions start jobs |
| Header overflow menu | U10 | "…" `Menu` for page actions beyond three, same look as the row menu |
| `IntegerField`, year select | F8 | Integer entry with locale separators off; a year picker bounded by the data |
| Content width token | U12 | `--mmerp-content-width` applied by form pages and the document main column |

| `SidePanel` | U1 (reopened) | Right-hand read-only panel (480px) showing a record's preview, with a link to the full page; its record id lives in the URL (`?peek=`) |
| `InlineEditField` | U3 (reopened) | A field on a record page that reads like text and turns into an input on click or Enter; Enter or ✓ saves, Esc or ✕ cancels, blur keeps it open |
| `ViewsMenu` + `useListView` | U6/U7 (reopened) | Saved views of a list and its column choices (visible, width), stored per user (ADR-0031) |
| `notifyUndo` | U4 (reopened) | Success toast with an Undo button for reversible actions, 5 seconds |

Not needed under these decisions: condition builder, shortcut help, command palette.
