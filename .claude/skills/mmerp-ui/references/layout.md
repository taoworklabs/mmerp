# Layout

Where things sit on a screen. Every screen is one page template from `@/shared/ui/page`; the template owns the frame (padding, header, small-screen layout), the screen fills its slots. Code recipes per screen are in `patterns.md`.

## App shell

```
┌──────────────────────────────────────────────────────────────────────────┐
│ ▣ Area name                                    🔔 3  ⬚ 2  ⧗   (QV) Name ▾  │ header 48px
├────────────┬─────────────────────────────────────────────────────────────┤
│ Overview   │                                                             │
│ GROUP      │   page template                                             │
│   Item ◼   │                                                             │
│   Item     │                                                             │
│ ⇤          │                                                             │
└────────────┴─────────────────────────────────────────────────────────────┘
  navigation 240px (56px collapsed)
```

- **Header** (`app/AppLayout.tsx`): the logo goes home, where areas are switched; then the open area's name; on the right notifications (unread count), approval inbox (pending count), jobs, and the account menu. The icon of the page you are on is `variant="light"`.
- **Navigation**: only the open area's menu, from the manifest's `nav` (`label`, `path`, `icon`, `group`, `permission`). Ungrouped items first, then groups; a group emptied by permissions disappears. Items are at least 30px tall (the control height), radius 6, weight 500, 16px icon; the current item is tinted with primary text. Collapsed and folded states are remembered per user. A screen adds a `nav` entry; it never draws navigation.
- **Home** (`/`) and the queues (inbox, notifications, jobs) have no navigation.
- Below 1024px the navigation is a drawer behind the menu button.

## Page header

Built by `Page` for every template:

```
Area › Parent                                                    breadcrumb (14px, dimmed)
Title                                       [Secondary] [⋯] [Primary]  h1 24px, one line
One-line description: what the list holds, or the record's key facts
───────────────────────────────────────────────────────────────────────  1px rule
```

- Exactly one primary button per page, rightmost. Up to three buttons in total; beyond that the rest go into `MoreActions place="page"` (`actions.md`).
- `RecordPage` adds an avatar or icon (`leading`) left of the title and replaces the rule with its tabs.
- Below 768px the actions wrap under the title at full width.

## Widths

| Region | Width |
| --- | --- |
| Lists, wide documents (`fullWidth`), reports | Full width of the content area |
| Form sections (`FormSection`) and a document's main column | At most 1200px (`--mmerp-content-width`), applied by the components |
| Document side column, inbox list | 360px |
| Side panel, side-column content shown as a tab | 480px (`readingWidth`) |
| Modal (`FormModal`) | `lg` (Mantine), fullscreen below 1024px |

Screens set none of these; the templates and components do.

## Screen types

Pick the type first; the template follows.

| The screen shows | Type | Template |
| --- | --- | --- |
| Many documents or master-data records | List | `ListPage` |
| One document (number, status, lifecycle) | Document detail | `DocumentPage` |
| One master-data record with tabs | Record detail | `RecordPage` |
| A form that creates a record or document | Create page | `Page` + `Form` |
| A short create/edit form (≤ 6 fields) | Create/edit modal | `FormModal` on its list |
| Small configuration lists | Settings | `ListPage` + `FormModal` |
| Counts and entry points | Dashboard | `Page` + `TileGrid` |
| Figures read across many rows, no editing | Report | `ListPage` (filters, no paging) + reconciliation `DataTable` |
| A queue to work through | Inbox | `InboxPage` |

### List

```
Title                                                            [Create]
Description
────────────────────────────────────────────────────────────────────────
[🔍 Search      ] [Status ▾] [Org unit ▾] [Month ▾]        [Views ▾] [Columns]
┌──┬──────────────┬───────────┬──────────┬───────────┬──────────┬───┐
│☐ │ Identifier ↕ │ Text      │ Date ↕   │  Amount ↕ │ Status   │ ⋯ │
├──┼──────────────┼───────────┼──────────┼───────────┼──────────┼───┤
│☐ │ DH-0001      │ Customer  │ 01/10    │    12.500 │ [Posted] │ ⋯ │
└──┴──────────────┴───────────┴──────────┴───────────┴──────────┴───┘
1.234 records                                  [50 / page ▾] ‹ 1 2 3 … ›
```

From 1024px the page is as tall as the window: the header, the filter bar and the paging bar stay put and only the content scrolls; a table that is the whole content scrolls inside itself under its sticky column headers (`ListPage` and `DataTable` do it). Below 1024px the page scrolls as a whole.

With rows selected, the filter bar row becomes `BulkBar`: `☑ 12 selected  [Action] [Action]  ✕ Clear selection`. Clicking a row outside its link may open the side panel on the right (480px, the list stays usable). Details in `tables.md`.

### Document detail

```
Quotations › BG-0012
BG-0012                                   [Print]  Delete   [Submit]
Customer · Org unit · [Draft]
───────────────────────────────────────────────────────────────────────
┌ main column ≤ 1200px ─────────────────┐ ┌ side 360px ──────────┐
│ [banner: document error / period lock]│ │ Approval             │
│ Section   field   field               │ │ Attachments (2)      │
│ ──────────────────────────────────────│ │ Discussion (5)       │
│ Section   field   field               │ │ History              │
└───────────────────────────────────────┘ └──────────────────────┘
▔ sticky footer while there are unsaved changes: Unsaved · Discard [Save] ▔
```

- Viewing and editing a draft are the same page; fields are read-only without `edit` in `allowed_actions`.
- A document with a wide table (editable lines, timesheet grid, payroll) sets `fullWidth`: the side column becomes tabs next to "Details" at every size, actions stay in the header.
- Below 1024px: one column, the side column as tabs, actions in a bar fixed to the bottom.

### Record detail

```
Employees › Nguyễn Văn An
(NA) Nguyễn Văn An                                    [Secondary] [Primary]
     NV0001 · Kế toán · [Active]
[Info] [Dependants] [Balances] [Contracts] [Attachments] [Discussion]
───────────────────────────────────────────────────── [xs action] (TabActions)
tab content: form sections, inline-edit fields, or a DataTable
```

The selected tab lives in the URL. A tab's own buttons sit in `TabActions`, size `xs`.

### Create page

```
List › New
Create quotation
───────────────────────────────────────────────────────────────────────
Section title        field            field
Description          field            field
───────────────────────────────────────────────────────────────────────
Lines (wide section: title on top, table full width)
▔ sticky footer: Unsaved changes · Discard [Create draft] ▔
```

After a successful create: toast, then the new record's page.

### Settings

A `ListPage` whose rows open their edit `FormModal` on click (`onRowClick`); "Create" in the header opens the same modal empty; a row's "…" menu appears only when it has two or more actions. Configuration that needs more than six fields, or tables of its own (approval rules), gets a page.

### Dashboard

`Page` + `TileGrid`: `StatTile` (a count that opens the list it counts), `LinkTile` (a destination). One column on phones, up to three on wide screens. No charts until a product needs one.

### Report

```
Report title                                              [Export]
Description
[Period ▾] [Legal entity ▾] [Org unit ▾]
┌ fills the page below the filters; header and totals stick ────────┐
│ Name │ Income ───────────│ Deductions ─────│ Net                    │
│      │ Base │ Allow │ OT │ Ins. │ Tax      │                        │
│ ▸ Group                                                             │
│ …                                                                   │
│ Group total                                                         │
│ Grand total (sticky)                                                │
└─────────────────────────────────────────────────────────────────────┘
```

Read-only; filters on the URL; a large export is a job. Columns grouped with `band`, totals per group and overall, zero as "–" (`tables.md`).

### Inbox

```
┌ list 360px ─────────────┐│┌ detail ─────────────────────────────────┐
│ ◼ BG-0012 Quotation     │││ BG-0012 · Customer        [Open]         │
│   Nguyễn A · 2h ago     │││ label        value                        │
│   DH-0007 Order         │││ Approval: level 1 of 2                   │
│                         │││ ▔ sticky: [Reject]            [Approve]  │
└─────────────────────────┘│└───────────────────────────────────────────┘
```

Selection in the URL (`?item=`); after a decision the next item opens. Below 1024px: list, then detail full screen with Back.

## Small screens

Below 1024px the templates switch on their own: navigation drawer, tables as cards (each column's `role`), filter chips wrapping under a full-width search box, document tabs and bottom action bar, fullscreen modals, top-centre toasts, lines as cards, 44px touch targets. Only self-service flows (own leave and overtime requests, approving) are designed for this size. A screen writes no small-screen layout of its own.
