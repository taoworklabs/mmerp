# User interface

Mandatory rules for every screen. **Agents and people must read this document before working on any screen.** How areas are composed and isolated is in [frontend.md](./frontend.md); the technology is in [techstack.md](./techstack.md#frontend).

Sources of truth in code:

- `web/src/shared/ui/theme.ts`: every colour, font size, spacing and radius. Sizes specific to one component (table row height, status dot, 44px touch target) live in its CSS module in `shared/ui`; areas set no other value apart from the width of a filter input.
- `web/src/shared/ui/`: page templates and components. New screens are **assembled from what already exists here**.
- The `/dev/ui` page, dev build only: shows every page template and component in every state. When unsure what to use, open this page first.

## Principles

1. **Consistency over creativity.** Two screens that do the same job must look and behave the same. Every component, wherever it lives, is assembled from `shared/ui` tokens and primitives (see [Where components live](#where-components-live)).
2. **Serve all-day users.** Users are HR staff, accountants and managers sitting at a desktop for many hours a day. Favour speed of operation, information density, keyboard use, and no surprises.
3. **Compact, modern, flat.** A minimalist style: plenty of white space, clear type hierarchy, thin borders instead of heavy frames. No gradients, no illustrations, no decorative shadows, no decorative animation. Colours that carry meaning (status, error) use the right token; decorative accent colours are used only for avatars and object-type icons, and never carry meaning.
4. **The backend decides, the UI displays.** Which buttons appear is decided by `allowed_actions`; error messages come from error codes. The UI never derives business rules on its own ([frontend.md](./frontend.md#permissions-the-frontend-computes-nothing)).

## Devices

| Width | Support |
| --- | --- |
| ≥ 1280px | Full. This is the design size |
| 1024–1279px | Every function usable; the navigation bar collapses automatically |
| < 1024px | Self-service flows only: submitting and viewing one's own leave requests and overtime requests; approving requests in the approval inbox. Admin screens are not optimised for this size |

### Layout on small screens

Below 1024px, page templates switch layout on their own. Areas do not write their own small-screen layout.

| Part | Below 1024px |
| --- | --- |
| Navigation bar | Hidden behind a menu button in the header, opens as a drawer |
| `ListPage` | The table becomes a list of cards. Each `DataTable` column declares its role: `title` (first line of the card), `status`, `meta` (secondary line) or `hidden` (not shown on the card). The filter bar collapses into a "Lọc" (Filter) button that opens a drawer |
| `DocumentPage` | One column. The side column's content (approval, attachments, discussion, history) becomes tabs under the page header: "Chi tiết · Duyệt · Đính kèm · Trao đổi · Lịch sử" (Details · Approval · Attachments · Discussion · History). Action buttons become a bar fixed to the bottom of the screen |
| `InboxPage` | Switches from two columns to list → detail: the list fills the screen; picking an item opens its detail full screen, with a back button. The browser's Back button also returns to the list |
| `FormModal` | Full screen |
| Success notification | Shown top centre, so it does not cover the action bar at the bottom |
| Form | Every field in one column |

No dark mode in the first release. The theme uses semantic tokens (below), so it can be added later without changing screens.

## Tokens

All of them live in `theme.ts`. Area code uses only token names or Mantine props, **never its own colour codes, font sizes in px, or spacing**.

### Colour

| Token | Value | Used for |
| --- | --- | --- |
| `primary` | Blue scale, main shade `#1E40AF` | Primary buttons, links, focus, selected navigation item |
| `text` | `#0F172A` | Main text |
| `text-secondary` | `#475569` | Secondary labels, captions, column headers |
| `placeholder` | `#94A3B8` | Placeholders and disabled state only; never for text that must be read |
| `border` | `#E2E8F0` | Table borders, input borders, dividers |
| `background` | `#FFFFFF` | App background and content area |
| `surface` | `#FFFFFF` | Background of tables, forms, cards |
| `subtle` | `#F8FAFC` | Background of the column header row |
| `success` | Green | Posted status, success notifications |
| `warning` | Amber | Pending approval status, warnings |
| `danger` | Red `#DC2626` | Errors, destructive actions, cancelled status |
| `info` | Blue | Neutral information |

Body text must have a contrast ratio of at least 4.5:1 against its background. Colour is never the only signal: a status always comes with text, an error always comes with text and an icon.

### Document status

Status colours are fixed for every document type. `<DocumentStatus>` is the only way to display a status.

| Status | Colour | Default label (vi / en) |
| --- | --- | --- |
| `draft` | Grey | Nháp / Draft |
| `pending_approval` | `warning` | Chờ duyệt / Pending approval |
| `posted` | `success` | Đã ghi sổ / Posted |
| `cancelled` | `danger` | Đã hủy / Cancelled |

A document type may change the **label**, never the colour, through the translation key `<doc_type>.status.<status>` in the area's `meta` translation file. Examples: a `posted` leave request is "Đã duyệt" (Approved), a `posted` payroll is "Đã chốt" (Finalised), a `posted` contract is "Có hiệu lực" (In effect). A module's own progress status (delivery, payment…) uses a grey or `info` `<Badge>`, not the four colours above.

### Type

- Font: **Inter** (variable font, with full Vietnamese diacritics), self-hosted through `@fontsource-variable/inter`. Fonts are never loaded from a CDN.
- Base font size **14px** for tables, forms and content. Minimum 12px, reserved for captions, column headers and badges. Line height 1.5.
- Page title 22px, weight 600. Section title 14–16px, weight 600. No more than three heading levels on a page.
- Field label 13px, weight 500, colour `text-secondary`: a label is lighter than the value it names.
- Every number and money column uses `font-variant-numeric: tabular-nums`, so digits line up.

### Spacing, radius, shadow

- 4px-based spacing scale: 4, 8, 12, 16, 24, 32. No other values.
- 6px radius for everything except avatars (round).
- **Controls in the same strip have the same height.** A strip is a row of controls placed side by side: filter bar, pagination bar, form footer, page header buttons. The `shared/ui` component that builds the strip picks the size of every control inside it; a button next to an input uses Mantine's `input-*` size so it is exactly as tall as the input (e.g. pagination at `input-sm` next to the rows-per-page select). Controls elsewhere keep the size that fits their place, as long as they meet the minimum touch target. Buttons inside an input (clear value, show password, hide sensitive field) are smaller than the input that holds them.
- No shadows, except on modals, dropdown menus and popovers.
- **A selected item only gets a light blue background**, everywhere (navigation, lists, menus, approval inbox): no accent bar on the left edge and no coloured border.

### Icons

- The [Tabler Icons](https://tabler.io/icons) set (`@tabler/icons-react`), stroke 1.75. Size 16px in text, in tables and in `xs` buttons, 18px in `sm` buttons, 24px (stroke 1.5) in empty states.
- No emoji as icons.
- Decorative icons (form section titles, org unit type in the org tree, empty states) are always `aria-hidden` and always come with text.
- People (employees, users) are shown with `PersonAvatar`: initials on an accent colour derived from the name, so a person always has the same colour.
- Icon-only buttons must have an `aria-label` and a tooltip. Exception: the show/hide password button inside an input needs only an `aria-label`.

## App shell

```
┌─────────────────────────────────────────────────────────────┐
│ Header 48px: logo + area name · [Notifications] [Approval inbox] [Jobs] [avatar] │
├──────────┬──────────────────────────────────────────────────┤
│ Navi-    │ Breadcrumb                                       │
│ gation   │ Page title                        [actions]       │
│ 240px    │──────────────────────────────────────────────────│
│ (collap- │ Content per page template                         │
│ sed 56px)│                                                  │
└──────────┴──────────────────────────────────────────────────┘
```

- The home page is a grid of tiles, one per area (Admin included), shown when the user has a permission in that product; a tile opens the first menu item the user can see. The home page, notifications, approval inbox and jobs have no navigation bar.
- Inside an area, the navigation bar holds only that area's menu, taken from the manifest's `nav`: ungrouped items first, then each group; a group left empty after permission filtering is hidden. Moving to another area goes through the home page (click the logo).
- The header shows the name of the open area ("Quản lý nhân sự", "Quản trị"), and the app name outside any area. **Notifications** (with the unread count, refreshed every minute), **Approval inbox** (with the number of pending items) and **Jobs** are icons in the header, present on every page. They open inside the current area (`/sales/inbox`), keeping its menu and header title, and at the root (`/inbox`) from home; opening an item goes to its record, wherever it lives, and Back returns to the list. Administration holds only their configuration (mail server, approval rules, roles). The notifications page is a `ListPage` holding an `InboxList`: one row per notification (event type, record type and document number · the person who caused it, time in relative form), bold while unread; clicking marks it read and then opens the record, or the jobs screen for job notifications. When the record can no longer be viewed, the row shows only the event type and record type.
- The navigation bar shares the content's background, separated by a right border; the selected item has a light blue background.
- The collapsed state of the navigation bar, and of collapsible groups, is remembered per user per browser.
- A disabled product that is still shown (because it has data) has a fixed banner on every page of its area: "Phân hệ đang tắt: chỉ xem và xuất được dữ liệu." (Product disabled: data can only be viewed and exported.)

## Page templates

Every screen uses one of the following templates. Templates live in `shared/ui/page`. No screen lays out the page level on its own.

| Template | Used for | Structure |
| --- | --- | --- |
| `ListPage` | A list of documents or master data | Title + primary "Tạo mới" (Create) button · filter bar · table · pagination |
| `DocumentPage` | Viewing and editing one document | Header: document number, `DocumentStatus`, `DocumentActions` · main column: the form's sections · 360px side column: `ApprovalPanel`, extra sections (`sections`: attachments if the record type answers `view_files`, discussion; titles carry a count; the comment box suggests people who can see the record when `@` is typed, and mentioned names are bold in the primary colour), `DocumentHistory`. Documents with wide tables (payroll, timesheet) drop the side column at every screen size: approval, extra sections and history become tabs next to "Chi tiết" (Details), and action buttons stay in the header |
| `RecordPage` | One master-data record (e.g. an employee profile) | Header · tabs (e.g. Info, Contracts, Leave, Pay). A tab's buttons go in `TabActions` |
| `ListPage` + `FormModal` | Configuration, small master-data lists (settings, leave types, legal parameters, work calendar) | A list of items; one item is edited in a modal. There is no separate configuration template |
| `InboxPage` | Approval inbox | Two adjacent columns, separated by a vertical line, unframed. Left: list of pending items, one flat row each (document number, type · submitter, submission time in relative form); the selected row has a light blue background. Right: `<RecordPreview>` (quick-view component the area declares in its manifest, [frontend.md](./frontend.md#opening-another-areas-document)) with fields as label left, value right, then `<ApprovalPanel>` with an Approve / Reject bar stuck to the bottom. After a decision it moves on to the next item automatically |
| `FormModal` | Quick creation of a record with at most 6 fields | Modal, buttons in the footer. More than 6 fields uses a page |
| `AuthPage` | Screens outside the app shell (sign-in) | One narrow content frame centred on the screen, no header and no navigation |
| `Page` | Pages that fit none of the templates above (home page, placeholder pages) | Title + primary button · content. The other templates are built on `Page`; the breadcrumb is an array of `Crumb` (`{ label, to }`) |
| `Page` + `TileGrid` | Home page, an area's overview page | Grid of tiles (`shared/ui/Tile`): `LinkTile` (icon + name, opens a page), `StatTile` (label + count, opens the matching list) |

General rules:

- **Page header** (built by `Page`, shared by every template): breadcrumb to the pages above, title, a one-line description (what the list holds, or the record's key facts), action buttons on the right, then a thin divider. `RecordPage` adds an avatar or icon left of the title and uses a tab bar instead of the divider. Below 768px (breakpoint `sm`) the action buttons wrap below the title and take the full width.
- **Viewing and editing a draft document is the same page** (`DocumentPage`); there is no separate "edit" page. Fields are editable when `allowed_actions` contains `edit`, read-only otherwise.
- Filters, sort, page and the selected tab live in the URL. Copying the URL reopens exactly that screen; the browser's Back button behaves as expected.
- Each page has exactly one primary button.

## Data tables

Use `<DataTable>` from `shared/ui`, not Mantine's `Table` directly.

- Rows are 40px tall. The column header row sits on the `subtle` background, 12px text in `text-secondary`. A long table scrolls with the page and the column headers do not stick yet; when needed, give the table frame a fixed height so the headers stick inside the frame.
- The first column is the identifier (document number, full name with avatar), weight 500 in the main text colour, underlined on hover. The whole row is clickable; the link is a real `<a>` tag, so middle-click or Ctrl+click opens a new tab.
- Numbers and money are right-aligned and use `tabular-nums`. Text is left-aligned. Dates are left-aligned.
- The status column uses `<DocumentStatus>` for documents and `<StatusBadge>` (badge with a coloured dot) for master data, e.g. employees who are active or have left.
- Sorting and pagination run on the server. Default 50 rows per page, with choices of 20, 50 and 100. Exception: small admin lists (users, org tree) load in full at once, without pagination.
- A list's search box is `SearchInput`: it applies on Enter or on leaving the box; `/` moves focus to it.
- No action buttons on each row. A row's secondary actions go in a "…" menu at the end of the row.
- A multi-select column exists only when the screen has bulk actions. When rows are selected, the bulk action bar replaces the filter bar.
- A wide table scrolls horizontally inside the table frame, never making the whole page scroll horizontally. The identifier column sticks to the left while scrolling horizontally and does not wrap.
- **Reconciliation tables** (many number columns, e.g. payroll):
  - Columns are grouped by meaning (income, insurance, tax…): two-level headers, each group starting with a vertical line. Result columns (net pay, cost) go last, in bold.
  - The table frame is at most 70% of the screen tall, so the column headers stick to the top of the frame and the totals row sticks to the bottom.
  - Zero is shown as "–" in the secondary colour, so the eye stops on cells that hold numbers.
  - A group's total row says only "Tổng" (Total); the group's name is already on the group's first row.

## Forms

Use the fields of `shared/ui/form`, already wired to React Hook Form.

- **Labels always sit above the input** and are always visible. Placeholders never replace labels. Required fields have a `*`. Help text sits below the input.
- Layout: split into sections (`FormSection`) with a title, a one-sentence description and an optional icon; sections are separated by thin dividers, unframed. From 1280px, the title and description sit in the left column and the fields in the right column, laid out in two columns; long text fields span the whole row.
- **A list's filter bar** is the only exception to the label rule: its inputs line up as a compact toolbar, each input's name shown as the placeholder and attached with `aria-label`, with an icon. Every data-entry form still has visible labels above.
- **Validation:** validate on leaving a field and on submit. The error appears right under the field, linked to the input with `aria-describedby`, and does not repeat the field name because the label is right above ("Chưa nhập" (Not entered), not "Nhập Mã nhân viên" (Enter Employee code)). On failed submit there is no summary box at the top of the form: focus and scroll to the first invalid field; next to the submit button a `role="alert"` line "Còn N trường cần sửa · Đến trường tiếp theo" (N fields to fix · Go to next field) cycles through the invalid fields and disappears when no errors remain; the title of each `FormSection` with errors carries that section's error count. `Form`, `FormActions` and `FormSection` do this already.
- **Server errors** that name a field are attached to that field; document-level errors (e.g. `period_locked`) show as a banner at the top of the page.
- Leaving the page with unsaved changes asks for confirmation.
- **A form footer on a page sticks to the bottom of the screen** (`FormActions` without `onCancel`): a long form always shows the Save button. With no edits there is no footer (except right after a failed submit, to keep the error line). Once there are changes, the left of the footer shows "Có thay đổi chưa lưu" (Unsaved changes), left of the Save button there is a red text button "Bỏ thay đổi" (Discard changes) (which asks before discarding), and the top divider turns `primary`. A form footer in a modal stays as is: not sticky and no state indicator. `FormActions` tells the two places apart by `onCancel`, so a form in a modal always passes `onCancel` and a form on a page never does.
- A control next to an input (checkbox, remove-row button) goes in an `InputRow` so it lines up with the input rather than its label.
- Specialised fields:

| Data | Component | Notes |
| --- | --- | --- |
| Date | `DateField` | Formatted per language (below); weeks start on Monday |
| Money | `MoneyField` | Integer, with thousands separators, no decimals for VND |
| Fractional quantity | `DecimalField` | Number of decimals declared by the field (e.g. workdays: 1) |
| Pick one from a list | `SelectField` | Has a search box when there are more than 7 options. Lists only valid options: what cannot be chosen is not shown. The selected option only gets a background, no check mark. The dropdown opens below the field's error message, never covering it |
| Org unit | `OrgUnitField` | Tree; shows only the units within the user's permission scope |
| Sensitive field | `SensitiveField` | Masked by default (`••••••`), with a "Hiện" (Show) button (every reveal is audited on the backend). Once revealed and not edited, a "Ẩn" (Hide) button masks it again. Without permission it reads "Không có quyền xem" (No permission to view) |

A field that picks a business object (e.g. picking an employee) belongs to the area that owns that object, not to `shared` (see below).

## Where components live

| Kind | Lives in | Examples |
| --- | --- | --- |
| Carries no business concept | `shared/ui` | Page templates, `DataTable`, basic input fields (`DateField`, `MoneyField`, `SelectField`…), loading, empty and error states, `FieldList` (read-only label–value pairs, label on the left) |
| Shared document features | `shared/document` | `DocumentStatus`, `DocumentActions`, `ApprovalPanel`, `DocumentHistory`, `AttachmentPanel`, `DiscussionPanel` |
| Carries a product's concept | That product's area | Employee picker, contract detail section, payroll breakdown table, leave request preview |

What `shared/ui` already has (see also `/dev/ui`), to use before building your own:

- Actions and feedback: `useConfirm` (confirmation dialog), `notifySuccess`/`notifyError` (`notify.ts`).
- States (`states.tsx`): `EmptyState`, `ErrorState`, `PageSkeleton`, `ContentSkeleton`, `ForbiddenPage`, `NotFoundPage`, `ReadOnlyBanner`.
- Display: `DataTable`, `FieldList`, `StatusBadge`, `CellGrid` (grid of inputs along two axes, like a timesheet), `TileGrid`/`StatTile`/`LinkTile`, `OrgUnitName`, `PersonName`/`PersonAvatar`.
- Forms (`shared/ui/form`): `Form`, `FormSection`, `InputRow`, `FormActions`; fields `TextField`, `PasswordField`, `CheckboxField`, `DateField`, `MonthField`, `FileField`, `SelectField`, `DecimalField`, `MoneyField`, `SensitiveField`, `OrgUnitField`/`OrgUnitSelect`; filters `SearchInput`, `MonthFilter`.
- Outside `shared/ui`: `useListParams` (`shared/url`: filters, sort, page in the URL), `ExportButton`/`ImportDialog` (`shared/jobs`), `OrgTree` (`shared/org`).

- Area components are **assembled from `shared/ui` primitives** and follow exactly the rules of this document: no custom colours, no inline `style`, no page-level layout of their own.
- When an area needs a generic display pattern that `shared/ui` lacks (e.g. a new kind of input, a new state), add it to `shared/ui`, with an entry on `/dev/ui`, rather than building it inside the area.
- Never move a business component down into `shared`, even when a second area needs it. In that case build a generic picker in `shared/ui` on top of `core`'s search API by record type, and record it as an ADR.

## Actions and feedback

- **Button hierarchy:** each view has one primary button (filled, `primary` colour). Other buttons are secondary (outlined). Red (`danger`) buttons are only for destructive or data-losing actions: delete, cancel a document, revoke a role, leave a page with unsaved changes.
- **Button size follows the level of where the button sits, not its importance.** Within a strip all buttons have the same height; the primary button stands out through its style (filled versus outlined), not its size. Between levels the size steps down, so a section's buttons do not compete with the page's:

  | Level | Place | Button size | Icon |
  | --- | --- | --- | --- |
  | Page | The page template's `actions`, `FormActions`, modal footer, buttons in empty and error states | `sm` (36px) | 18px |
  | Section | `TabActions`, the header of a section within a page, the `DocumentPage` side column (`ApprovalPanel`) | `xs` (30px) | 16px |
  | In content | Add row in a form, buttons sitting among data; a button right after a line of text (change approver next to the approver's name) uses `compact-xs` | `xs`, `light` or `subtle` style | 16px |

  Page-level buttons keep the default size; section-level and in-content buttons set `size="xs"`.
- **Button placement:** action buttons are always right-aligned: in the page header (the page template's `actions`), at the top of a `RecordPage` tab (`TabActions`) and in a form or modal footer (`FormActions`). Areas do not align buttons themselves.
- **Button names are specific verbs:** "Lưu" (Save), "Gửi duyệt" (Submit for approval), "Rút lại" (Withdraw), "Duyệt" (Approve), "Từ chối" (Reject), "Hủy chứng từ" (Cancel document). Never "OK" or "Đồng ý" (Agree).
- **Ask for confirmation** before deleting a draft, cancelling a posted document, and rejecting an approval. Rejecting requires a reason. The confirm button repeats the exact verb ("Hủy chứng từ"), not "Có" (Yes).
- **In progress:** a submitting button cannot be pressed again, and shows a loading state only if the action takes longer than 300ms, to avoid flicker (`FormActions` does this). The first load uses skeletons: `PageSkeleton` replaces the whole page, with the same margins as `Page`; `ContentSkeleton` replaces a region inside a page (page body, tab, panel), with no margins of its own so it lines up with the header. Skeletons shaped like each layout are added when needed.
- **Success:** a short notification in the bottom-right corner, closing itself after 3 seconds.
- **Errors:** place the error near what caused it (the field, or a banner at the top of the page). Corner notifications are only for job errors. A `version` conflict error always shows a "Dữ liệu đã thay đổi" (Data has changed) dialog with a "Tải lại" (Reload) button.
- **Error messages** say what happened and what the user needs to do: "Kỳ tháng 3 đã khóa sổ. Liên hệ kế toán nếu cần sửa chứng từ này." (The March period is locked. Contact accounting if this document needs changing.) Never show technical error codes, except in a collapsed details section to send to support.

## States of a data region

Every table, page and panel handles all of the following states. The `shared/ui` components already have each state.

| State | Display |
| --- | --- |
| Loading | Skeleton shaped like the layout, `aria-busy` |
| Empty | Dashed-border frame, icon, one explanatory sentence, plus a create button if the user may create and the page does not already have that button. Empty because of filters shows "Không có kết quả" (No results) and a clear-filters button |
| Error | Message by error code, plus a "Thử lại" (Retry) button |
| Forbidden | 403 page with an explanation; never hide silently |
| Not found | 404 page. A record that does not exist (or is out of scope) links back to the list holding it; an unknown path goes back to the home page |
| Read-only | Disabled product: banner (see App shell); write buttons disappear on their own through `allowed_actions` |

## Text and formatting

- **Every string goes through i18n.** No hard-coded text in components. Translation keys have the form `<area>.<screen>.<element>` ([techstack.md](./techstack.md#i18n)).
- **Terminology** comes exactly from [CONTEXT.md](./CONTEXT.md). The same concept uses the same word on every screen.
- **Voice:** short, neutral, no personal pronouns. Instructions use the imperative: "Chọn kỳ lương" (Choose the pay period), not "Bạn hãy chọn kỳ lương" (Please choose the pay period).
- **Capitalisation:** capitalise only the first word of a sentence and proper names, including on buttons and titles ("Gửi duyệt", not "Gửi Duyệt").

| Data | `vi` | `en` |
| --- | --- | --- |
| Number | `1.234.567,5` | `1,234,567.5` |
| VND money in a table | `1.234.567` (unit stated in the column header) | `1,234,567` |
| Standalone VND money | `1.234.567 ₫` | `₫1,234,567` |
| Date | `31/03/2026` | `31/03/2026` (en-GB) |
| Date and time | `31/03/2026 14:05` | `31/03/2026 14:05` |
| Date range | `01/03/2026 – 31/03/2026` | same as `vi` |

Dates and times are shown in the tenant's time zone, not the browser's ([techstack.md](./techstack.md#time)). All formatting goes through `shared/i18n` functions; areas never call `Intl` or `toLocaleString` directly.

## Keyboard and accessibility

- Every action can be done with the keyboard. Keep Mantine's default focus ring, never remove it.
- Tab order follows reading order. Opening a modal moves focus into it; closing it returns focus to where it was.
- Shortcuts, and only these: `/` to go to a list's search box; `Ctrl+S` to save a form; `Esc` to close a modal or menu.
- Minimum touch target 32×32px on desktop, 44×44px on self-service screens on small devices.
- Respect `prefers-reduced-motion`.

## Motion

Only Mantine's default motion (opening modals, menus, notifications), 200ms at most. No page transitions, no scroll effects.

## Enforcement

The rules above are enforced by tools wherever tools can do it:

| Rule | Enforced by |
| --- | --- |
| Areas do not use Mantine components that carry conventions directly: `Table`, `Modal`, `Badge`, `Notification`, `NumberInput`, the `@mantine/dates` package and the `@mantine/notifications` package | ESLint `no-restricted-imports` (`importNames` option), with a message pointing to the matching `shared/ui` component |
| No inline `style` and no CSS files in areas | ESLint `no-restricted-syntax` blocks the `style` attribute in JSX (outside `shared/ui`); dependency-cruiser blocks `*.css` imports from areas. CSS modules exist only in `shared/ui` |
| No colour codes outside `shared/ui` | `make lint` scans for hex codes (3, 6 and 8 digits), `rgb(` and `hsl(` in every file of `web/src` except `shared/ui`, `.css` files included |
| No hard-coded text in JSX | ESLint `i18next/no-literal-string` |
| Translation keys have both `vi` and `en` | Check in `make lint` ([techstack.md](./techstack.md#i18n)) |
| Page-level layout, button hierarchy, button names, voice | Cannot be enforced by tools: use page templates, and check against the list below during review |

## Checklist before finishing a screen

- [ ] Uses a page template from `shared/ui/page`; no page-level layout of its own.
- [ ] No custom colours, font sizes or spacing; no inline `style`.
- [ ] Document status shown with `<DocumentStatus>`; action buttons come from `allowed_actions` through `<DocumentActions>`.
- [ ] All states present: loading, empty, error, forbidden, read-only.
- [ ] Filters, sort, page and tab live in the URL.
- [ ] Each view has exactly one primary button; button names are specific verbs; destructive actions ask for confirmation.
- [ ] Button size by level: `sm` in the page header and form footer, `xs` in section headers, tabs, the side column and in content.
- [ ] Errors appear near their cause; `version` conflict errors use the reload dialog.
- [ ] A form on a page uses `FormActions` without `onCancel`; a form in a modal always has `onCancel`. Controls next to an input (checkbox, remove-row button) sit in an `InputRow`.
- [ ] Every string goes through i18n, with both `vi` and `en`, terminology matching `CONTEXT.md`.
- [ ] Numbers and money are right-aligned, use `tabular-nums`, formatted through `shared/i18n`.
- [ ] A selected item only gets a background, no accent bar on the left edge.
- [ ] Fully usable with the keyboard; icon-only buttons have an `aria-label`.
- [ ] The screen displays correctly at 1280px and 1024px; self-service screens display correctly at 375px.
- [ ] New components are placed per the "Where components live" table; new components in `shared/ui` have an entry on `/dev/ui`.
- [ ] Below 1024px, the screen uses the layout the page template switches to on its own; no separate small-screen layout.
