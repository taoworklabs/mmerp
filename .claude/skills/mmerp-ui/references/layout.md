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
- `RecordPage` adds an avatar or icon (`leading`) left of the title and sets its tabs on the header's rule (`Page` `tabs`: the rule is the tab line, never a second one); so do a `DocumentPage` with tabs and a `ListPage` with view `tabs` (the org tree: tree, table). A `RecordPage` with a profile column (`aside`) has no rule under the header (`Page` `plain`): its tabs draw their own frame.
- On one record's page (`RecordPage`, `DocumentPage`, which pass `withBack` to `Page`) the breadcrumb ends with the page's own title (weight 500, not a link) and a round back button (←) leads the title: back in history to the list as the user left it, or to the last crumb when opened directly. Areas pass only the crumbs above.
- Below 768px the actions wrap under the title at full width.

## Widths

| Region | Width |
| --- | --- |
| Lists, wide documents (`fullWidth`), reports | Full width of the content area |
| Form sections (`FormSection`) and a document's main column | At most 1200px (`--mmerp-content-width`), applied by the components; on record and form pages centred under the header's rule or the tab line (`RecordPage`, `DocumentPage`, `Page centered`), the header and tabs keeping the full width |
| Document side column, inbox list | 360px; a document's side column grows to 35% of the page, up to 480px |
| Record profile column (`RecordPage` `aside`) | 480px (`--mmerp-aside-width`) |
| Side panel, side-column content shown as a tab | 480px (`readingWidth`) |
| Modal (`FormModal`) | 440px (`md`), 620px with `wide` (`lg`), fullscreen below 1024px |

Screens set none of these; the templates and components do.

## Screen types

Pick the type first; the template follows.

| The screen shows | Type | Template |
| --- | --- | --- |
| Many documents or master-data records | List | `ListPage` |
| One document (number, status, lifecycle) | Document detail | `DocumentPage` |
| One master-data record with tabs | Record detail | `RecordPage` |
| A form that creates a record or document | Create page | `Page` + `Form` |
| A short create/edit form (≤ 6 fields), or one group of a record's values | Create/edit modal | `FormModal` on its list, or opened from the group's `EditableSection` |
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
┌ main column ≤ 1200px ─────────────────┐ ┌ side 360–480px ──────┐
│ [banner: document error / period lock]│ │ Approval             │
│ Section   field   field               │ │ Attachments (2)      │
│ ──────────────────────────────────────│ │ Activity             │
│ Section   field   field               │ │ (comment, timeline)  │
└───────────────────────────────────────┘ └──────────────────────┘
▔ sticky footer while there are unsaved changes: Unsaved · Discard [Save] ▔
```

- Viewing and editing a draft are the same page; fields are read-only without `edit` in `allowed_actions`.
- A document with a wide table (editable lines, timesheet grid, payroll) sets `fullWidth`: actions stay in the header; from 1024px the side column's panels gather in one "Activity" tab next to "Details", two columns at the content width (approval and sections framed on the left, the activity feed unframed on the right), never one short panel stretched over the whole width.
- Below 1024px: one column, the side column as tabs, actions in a bar fixed to the bottom.
- **Activity is one standard everywhere:** a document's side column and a record page show the same `ActivityFeed`: the comment box, a filter (all, comments, changes, reads), then one timeline newest first under day headings, each line the time of day in a column, a 20px bullet (the person's avatar for plain doings and comments; a coloured icon for a status reached, a decision, a creation or a sensitive read) and one sentence: who in weight 600, the doing in `dimmed`, what to in weight 600. One changed field reads in the sentence (old struck out → new); a comment's text, or more fields, go in a sunken well under it. There is no separate discussion or history panel; old `?tab=discussion|history` links open the activity.

### Record detail

```
Employees › Nguyễn Văn An
(NA) Nguyễn Văn An                                    [Secondary] [Primary]
     NV0001 · Kế toán · [Active]
                                  ┌───────────────────────────────────────────
┌ profile 480px ─────────────┐    │ [Activity] [Dependants] [Contracts] [Files]
│ ▣ Main details      ✎ Sửa │    ├───────────────────────────────────────────
│ Label        value         │    │                       [action] (TabActions)
│ Label        value         │    │ tab content: the activity feed or a DataTable
│ ─────────────────────────  │    │
│ ▣ Personal and contact     │    │
│ Label        value         │    │
└─ scrolls on its own ───────┘    └─ scrolls on its own, frame open at the bottom right
```

- The profile column (`aside`) holds the record's values in groups (`EditableSection` around a `FieldList`); the labels share one width down the column. A group the user may edit always shows "Sửa", grey until hovered or focused (which also tints the group); a click opens that group's fields in a `FormModal` ("Sửa …"), which saves the whole record. A single value changed on its own (a user's email) may stay an `InlineEditField`.
- "Hoạt động" (Activity, `activity`) comes first when the record has an audit log; old links to "discussion" or "history" land there. Then the record's own tabs; attachments last, in a 480px reading column. It is first with or without a profile column; below 1024px "Thông tin" comes before it.
- The page fills the window under the app header; each column scrolls on its own in a `ScrollArea` with a quiet overlay scrollbar kept clear of the content. The tabs' frame shows only its top and left edges, tied to the window's bottom right corner; their rule is a hairline across the frame; tabs and content sit 48px (`xxl`) in from its left edge. Tab content is centred at 1200px, except the activity feed, which reads from the left at a comfortable line length.
- Below 1024px the profile column becomes the first tab, "Thông tin", and the page scrolls as a whole.
- The selected tab lives in the URL, replacing the history entry rather than adding one, so Back (and the back button) leaves the record instead of walking back through its tabs. A tab's own buttons sit in `TabActions`, at the default size.

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
