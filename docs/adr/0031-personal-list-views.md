# 0031. Each user keeps their own list views on the server

Status: accepted · 2026-10-12

## Context

Lists keep their filters, sort and page in the URL, so a bookmark already reopens a filtered list. Users who return to the same few filters every day still have to rebuild or bookmark them in each browser, and every list shows every column the screen designer chose, with fixed widths. The UX decisions of 2026-10-12 (`docs/design/ux-patterns.md`, U6/U7, reopened) ask for named views and a choice of visible columns and their widths, per user, kept on the server so they follow the user across browsers.

Constraints: on-premise has only the app and Postgres; business code is tenant-unaware; nothing here may reveal data the user cannot already see.

## Decision

- **One row per user per list in `iam.list_settings`**: `user_id`, `list_key` (the list's stable name, e.g. `sales.customers`), and `settings` as `jsonb` holding `views` (name + the list's URL query string) and `columns` (key, hidden, width). `iam` owns it because it is a property of the account and goes away with it (`ON DELETE CASCADE`).
- **Two operations on the signed-in user only**: `GET /me/lists/{key}` (an absent row reads as empty settings) and `PUT /me/lists/{key}` (replaces the whole row). No one reads or writes another user's settings, administrators included.
- **The server stores, the screen interprets.** A view is the query string the list already puts in its URL; applying it is navigating to it, so permissions and filters are checked exactly as for a typed URL. Column keys are the list's own; unknown keys are ignored by the screen. The server checks only shape and bounds (at most 50 views of unique, non-blank names up to 100 characters, queries up to 2,000 characters, at most 100 columns, widths 48–800px).
- **No audit rows.** These are personal display preferences, not business data; the existing rule "every business write is audited" does not reach them.
- **No organisation-wide default yet.** A list with no row shows the screen's designed columns; an administrator-set default is added when a tenant asks for it, as a second source the screen reads before the user's own.

## Rejected alternatives

- **Browser `localStorage`**, like the navigation's collapsed state. Nothing to build on the server, but views would not follow the user to another browser or machine, which is the point of saving them.
- **A table per concern** (`list_views`, `list_columns`). Two tables and four endpoints for data that is always read and written together, by its single owner.
- **Storing filters as structured data** rather than the URL query. It would duplicate each list's filter vocabulary on the server and drift from it; the query string is already the list's public contract.
- **Shared views between users.** Needs a permission model for who may publish and see a view; no one has asked for it.
