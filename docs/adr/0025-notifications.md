# 0025. Notifications

Status: accepted · 2026-10-07 · extends the "Jobs" part of [ADR-0019](./0019-background-jobs-dataio-and-file-storage.md): a notification flag in `JobSpec`, and administrators can see system jobs

## Context

Approvers only learn of a pending request when they open the approval inbox themselves; submitters only learn the outcome when they reopen the request; people importing timesheets or computing payroll have to watch the background jobs screen. The discussion of [ADR-0024](./0024-attachments-and-discussion.md) cannot call anyone. The system does not send email yet, users have no email address yet, and system jobs only live in River's table, where nobody can see them.

Constraints: no business operation depends on the Internet; no broker, no sidecar; notifications must not reveal amounts or sensitive fields.

## Decision

### Module `notification` in `core`

- Table `notification.notifications (id, user_id, kind, doc_type, doc_id, job_id, actor_id, created_at, read_at)`:
  - `kind` has a `CHECK` on exactly this set: `approval_requested`, `approval_approved`, `approval_rejected`, `mentioned`, `job_completed`, `job_failed`;
  - `doc_type`, `doc_id` are set when the notification is about a record; `job_id` when it is about a job; a `CHECK` enforces exactly one of the two;
  - `actor_id` is the person who caused the event, empty for job notifications; an empty `read_at` means unread;
  - an index on `(user_id, id DESC)` and a partial index on `user_id` for unread rows.
- **No content column.** The displayed text is built by the frontend from `kind` through i18n, and email from `kind` through server i18n. There is nowhere for an amount, a rejection reason or comment text to leak.
- **A single send function**, called in the event's transaction: it takes the kind, the list of recipients, and the record or job. The function drops the person who caused the event from the recipients, writes the rows, and enqueues the email job when that kind has email (see below).
- Sending a notification does not change `version`, writes no audit on the record, and does not go through the period lock or the product gate.
- No deletion, no expiry. Cleaning up old notifications is done when numbers show it is needed.

### Who emits

| Kind | Emitter | Recipients |
| --- | --- | --- |
| `approval_requested` | `approval`, when a step starts: submission, the previous step just approved, the step falls back to the fallback role, reassignment | Every approver of the step |
| `approval_approved` | `approval`, when the last step is approved | The submitter of the approval instance |
| `approval_rejected` | `approval`, on rejection | The submitter of the approval instance |
| `mentioned` | `discussion`, when a comment is added | Mentioned people, if they can view the record |
| `job_completed`, `job_failed` | middleware and `platform.CompleteJob`, for job kinds that declare it | The requester |

- `approval` and `discussion` call the `notification` service directly, as they call `audit`. There is no event bus.
- Approval of an intermediate step does not notify the submitter; the submitter only receives the final outcome.
- Every document type with approval gets notifications automatically, with nothing extra to declare.

### Not revealing records

- Every listing re-checks `record.Can(view)` for each notification about a record. If still viewable, it returns the record id, a label (the document number for documents; catalogues have no label) and the actor. If no longer viewable, or the record type is no longer registered, it returns only the event kind, the record type and the time.
- Someone else's notification: marking it read returns not found.
- The list and the unread count are core reads: they stay readable when the product is disabled.

### Mentions

- Syntax `@login` in the plain text of a comment. The comment is stored verbatim, with no extra column or table.
- When a comment is added, `discussion` extracts the `@login`s, looks up the users, keeps those who can view the record (`record.Can(view)` on their behalf), and sends `mentioned` in the comment's transaction.
- A login that does not exist, or a person who cannot view the record: silently ignored, the comment is still saved, no error is returned. Reporting an error would tell the writer who can view the record.
- `GET /records/{type}/{id}/mentionable` returns the login and name of users who can view the record, for the comment box to suggest when typing `@`. Same not-revealing rules as the other `/records/...` routes.

### Job notifications

- `platform.JobSpec` gets a `Notify` flag. `dataio.import`, `dataio.export` and the HRM payroll compute job declare it. System jobs never emit notifications, even if the flag is set by mistake.
- A job kind declaring `Notify` must complete with `platform.CompleteJob`. The `job_completed` notification is written in that same transaction, so job completion and notification commit together.
- `job_failed` is written in the middleware when the job reaches a final state: cancelled by an error with a code, or failing on the last attempt. River records the cancelled state itself after the worker returns, so the notification lives in its own transaction; a crash exactly between the two steps can leave an extra `job_failed` for a job that then runs again. Accepted: rare, and the user still sees the real status on the background jobs screen.
- `platform` does not know `core`: the middleware receives a function that `internal/app` wires to `notification`.

### Email

- `iam.users` gets an `email` column, nullable, not encrypted (not a sensitive field); the service checks basic format. Administrators edit it on the users screen. Addresses are not verified.
- **The mail server** is a single-row configuration owned by `notification`: host, port, security mode (`CHECK`: `starttls`, `tls`, `none`), username, password, sender address, and the mmerp access URL used to build links (an on-premise machine does not know its own public address).
  - The password is column-encrypted with `platform.Encrypt`. The API never returns the password, only a flag that it is set; saving without sending a password keeps the old one, but only if host, port, security mode and username are unchanged. Changing any of these requires entering the password again, so a stored password is never sent to a different server.
  - New permission `core.mail.manage`, in the `core.admin` role. Each save or delete writes audit, without the password.
  - With no configuration, no email is sent; in-app notifications still work.
- Only `approval_*` and `mentioned` send email. Job notifications do not: the requester is waiting in the app.
- The send function enqueues the system job `notification.email` in the event's transaction, when a mail server is configured and the recipient has an email. Mail errors or loss of Internet never break business operations.
- **Args hold only the notification id** (test email: the user id), no address or content, because args live in River's table and show on the admin screen. The current address and configuration are read only when the job runs; if the recipient no longer has an email or the configuration has been deleted, the job finishes without sending.
- The email has a subject and one sentence per event kind, with a link to the notifications page (the server does not know the page path of each record type; from there the recipient clicks the notification to open the record), in the recipient's language, or the tenant's when the user has not chosen one. No document number, person name, amount or comment text. The link goes through sign-in and re-checks permissions like every page.
- Sent with `net/smtp` and `crypto/tls` from the standard library.
- Errors have stable codes: connection failure and temporary server errors (`mail_unreachable`) are retried by River; wrong credentials (`mail_auth_failed`) and rejected addresses (`mail_rejected`) are cancelled immediately. Error messages and logs carry only the code, never the recipient address, the email content or the server's verbatim reply.
- "Gửi thư thử" (send test email) enqueues the same job to the person who clicked; the result is shown on the background jobs screen.
- The email job does not declare `Notify`: email success or failure produces no further notification or email.

### Administrators see system jobs

- New permission `core.job.monitor`, in the `core.admin` role. `GET /jobs` accepts the extra filters `system=true` and `failed=true`; calling `system=true` without the permission is refused.
- System jobs return kind, status, attempt count, time and the error code of the latest attempt. Args are never returned. Errors without a code show `internal_error`.
- Everyone else still sees only their own jobs.

### Core API

| Route | Notes |
| --- | --- |
| `GET /notifications` | The user's own, newest first, `before` cursor by id |
| `GET /notifications/unread-count` | Polled by the header, like the approval inbox count |
| `POST /notifications/{id}/read` | |
| `POST /notifications/read-all` | |
| `GET /records/{type}/{id}/mentionable` | People who can be mentioned on the record |
| `GET`, `PUT`, `DELETE /mail-server` | `core.mail.manage` |
| `POST /mail-server/test` | Test email to the person who clicked |
| `GET /jobs?system=&failed=` | Extends the existing route |

The unread count updates by polling, not real-time push.

## Rejected alternatives

- **An event bus or a common `OnTransition` hook for notifications**: the architecture has no event bus, modules call each other through interfaces; and only `approval` knows who the approvers of a step are, so it is the natural emitter.
- **Storing pre-built notification content**: the text is frozen in the language at send time, and it opens a path for amounts or names to leak into a table that does not go through per-field permissions.
- **Real-time push (SSE, WebSocket, `LISTEN/NOTIFY`)**: adds long-lived connections and state for a number that polling already handles.
- **Recipient address and email content in the email job's args**: args show on the admin screen and stay in River's table for up to 7 days; an address changed after enqueueing would not be used either.
- **Reporting an error when mentioning someone who cannot view the record**: lets the writer probe who has view access.
- **An external mail library**: `net/smtp` is enough for authenticated SMTP, STARTTLS and TLS.
- **Email for every notification kind**: job notifications arrive while the user is in the app; email only adds noise.

## Consequences

- A new product gets approval notifications without writing anything; a new job gets notifications with one flag.
- Every listing calls `Can` once per notification about a record; with 50-row pages that is acceptable.
- Mention suggestions call `Can` for each user of the tenant; with a few hundred users that is acceptable, beyond that it needs filtering by role scope.
- The notifications table grows without bound until there is a cleanup.
- Email goes through the organisation's mail server; mmerp keeps no copy.
- Administrators can point the mail server at any host and port the mmerp server can reach, including the internal network, and read the error code to learn whether a port is open. On-premise, an internal mail server is the correct case and the tenant administrator is trusted; with multi-tenant cloud, connection targets must be restricted.
