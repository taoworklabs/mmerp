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
- **Navigation**: only the open area's menu, from the manifest's `nav` (`label`, `path`, `icon`, `group`, `permission`). Ungrouped items first, then groups; a group emptied by permissions disappears. Items are 32px rows set with no gap, radius 6, the label 14px weight 500 and its 16px icon (8px apart) in `dimmed` until active; a group's heading has room above it; the current item is tinted with primary text (`AppLayout.module.css`). Collapsed and folded states are remembered per user. A screen adds a `nav` entry; it never draws navigation.
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
- `DetailPage` may add an avatar or icon (`leading`) left of the title; its key facts are the description, joined by "·", with the status badge. A `fullWidth` `DocumentPage` and a `ListPage` with view `tabs` (the org tree: tree, table) set their tabs on the header's rule (`Page` `tabs`: the rule is the tab line, never a second one).
- On one record's page (`DetailPage`, and `DocumentPage` built on it, which pass `withBack` to `Page`) the breadcrumb ends with the page's own title (weight 500, not a link) and a round back button (←) leads the title: back in history to the list as the user left it, or to the last crumb when opened directly. Areas pass only the crumbs above.
- Below 768px the actions wrap under the title at full width.

## Widths

| Region | Width |
| --- | --- |
| Lists, wide documents (`fullWidth`), reports | Full width of the content area |
| Form pages (`FormSection`, `Page centered`) | At most 1200px (`--mmerp-content-width`), the header centred with the content so both share a left edge |
| A record's or document's page (`DetailPage`, `DocumentPage`) | At most 1440px (`--mmerp-detail-width`), header included; main and side column about 62/38, one column below 1024px |
| Inbox list | 360px (`--mmerp-side-width`) |
| Side panel (`SidePanel`), side-column content shown as a tab | 480px (`--mmerp-aside-width`, `readingWidth`) |
| Modal (`FormModal`) | 440px (`md`), 620px with `wide` (`lg`), fullscreen below 1024px |

Screens set none of these; the templates and components do.

## Screen types

Pick the type first; the template follows.

| The screen shows | Type | Template |
| --- | --- | --- |
| Many documents or master-data records | List | `ListPage` |
| One document (number, status, lifecycle) | Document detail | `DocumentPage` |
| One master-data record | Record detail | `DetailPage` |
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

### Record detail

```
Employees › Nguyễn Văn An
← (NA) Nguyễn Văn An                              [Secondary] [Primary] [⋯]
       NV0001 · Kế toán · [Active]
┌ main column ~62% ──────────────────────┐ ┌ side column ~38% ──────────────┐
│ ▣ Main details                  ✎ Sửa │ │ [Contracts] [Leave] [Files]    │
│ Label        value                     │ │                 [action]       │
│ Label        value                     │ │ DataTable as cards             │
└────────────────────────────────────────┘ └────────────────────────────────┘
┌ ▣ Personal and contact          ✎ Sửa ┐ ┌ Activity ──────────────────────┐
│ Label        value                     │ │ (comment, timeline)            │
└────────────────────────────────────────┘ └────────────────────────────────┘
          body and header centred at 1440px, one left edge
```

- `DetailPage`: header (back, title, key facts joined by "·", status badge; actions on the right, the rest in `more`, a "…" menu), `banner` over both columns, the main column (`children`) and the side column (`side`), each a stack of cards (`Panel`: title, optional icon and actions). The cards' label–value pairs share one label width (11rem) down the page.
- A group of the record's values is an `EditablePanel`: read as `FieldList` pairs, never inputs, and shows every value its form edits, those also in the header (name, code, unit) included. "Sửa", when the user may edit, turns that card alone into the group's form (`FormFields`, two fields to a row; `FormActions` with `onCancel`: Hủy, Lưu) in place, back once saved or cancelled. No `FormModal` for a group. A single value saved on its own (a user's email) may stay an `InlineEditField`.
- The side column: the record's own lists in one card (`DetailTabs`: the first tab open by default, the tab on the URL, replacing the history entry; a tab's buttons in `TabActions`; tables as `DataTable` `cards`), then "Hoạt động" (`ActivityFeed`) in a `Panel` of its own, never a tab.
- Below 1024px: one column, the side column under the main one, the actions in a bar fixed to the bottom (`narrowActions`).
- People link to their profile (`EmployeeLink` in HRM); a phone is a `tel:` link, an email a `mailto:` link.

### Document detail

```
Leave › NP-0012
← NP-0012                         [Từ chối] [Duyệt]  [Rút lại] [⋯]
  Nguyễn Văn An · 3 days · [Pending approval]
┌ main column ~62% ──────────────────────┐ ┌ side column ~38% ──────────────┐
│ [banner: document error / period lock] │ │ Approval (every step)          │
│ ▣ Request                       ✎ Sửa │ │ Attachments (2)                │
│ Label        value                     │ │ Activity (comment, timeline)   │
└────────────────────────────────────────┘ └────────────────────────────────┘
```

- `DocumentPage` builds on `DetailPage`. Header: number, the person it is for (a link to their profile), key facts, `DocumentStatus`; the actor's "Từ chối" and "Duyệt" (`ApprovalDecision`) when they decide the current step, then `DocumentActions` ("Rút lại" asks first).
- Main column: the document's values as `EditablePanel` cards (leave, overtime, contract); "Sửa" shows while `allowed_actions` has `edit` (a draft) and edits that card in place.
- Side column: `ApprovalPanel` with `decideElsewhere` (the buttons are in the header) and `subject` ("gửi thay cho …" when sent for someone else); it lists every step ("Đã duyệt", "Đang chờ", "Chưa tới", "Từ chối"). Then `sections` (attachments), then the activity. The inbox keeps its own sticky Approve / Reject bar.
- A document with a wide table (payroll, timesheet, quotation, order) sets `fullWidth`: no side column at any size, it shows its form (editable with `edit`, read-only otherwise) and actions stay in the header; from 1024px approval, sections and activity gather in one "Activity" tab next to "Details", two columns at the content width (approval and sections on the left, the activity feed on the right); below 1024px one tab per part.
- Below 1024px (not `fullWidth`): one column, the side cards under the main ones, actions in a bar fixed to the bottom.
- **Activity is one standard everywhere:** documents and records show the same `ActivityFeed`, a card of its own: the comment box (one line until focused), a filter (all, comments, changes, reads), then one timeline newest first under day headings, each line the time of day in a column, a 20px bullet (the person's avatar for plain doings and comments; a coloured icon for a status reached, a decision, a creation or a sensitive read) and one sentence: who in weight 600, the doing in `dimmed`, what to in weight 600. One changed field reads in the sentence (old struck out → new); a comment's text, or more fields, go in a sunken well under it. There is no separate discussion or history panel; old `?tab=discussion|history` links open the activity.

### Create page

```
List › New
Create quotation
───────────────────────────────────────────────────────────────────────
Section title        field            field
Description          field            field
───────────────────────────────────────────────────────────────────────
Lines (wide section: title on top, table full width)
▔ sticky footer from the start:                  [Cancel] [Create draft] ▔
```

A short request of a few fields (new leave, new overtime) uses `FormSection compact`: title on top, four fields to a row.

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
