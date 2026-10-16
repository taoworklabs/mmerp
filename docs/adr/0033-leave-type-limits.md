# 0033. A leave type may cap the days of one request and require an attachment; both always refuse

Status: accepted · 2026-10-16 · extends [ADR-0016](./0016-leave-requests.md) (leave types) and follows the two-place check of [ADR-0030](./0030-leave-day-allocation.md)

## Context

A leave type carries only `deducts_balance`, `paid` and `active`. The kinds the Labour Code grants per event (Article 115: 3 days for one's own marriage, 1 for a child's marriage, 3 for the death of a parent, spouse or child) are entered as ordinary types, and nothing stops a request for 5 days of "marriage" leave: the company pays days the law does not grant, and the approver has to spot it by eye. Sick leave needs a medical certificate, but a request goes through without any file. The leave detail page already has room for these rules and shows nothing.

## Decision

- **Two optional properties on a leave type.** `max_days`, the most days one request of the type may take (`numeric(4,1)`, a multiple of 0.5, `NULL` for no cap), and `requires_attachment` (`boolean`, default `false`). Both are set by whoever manages leave types (`hrm.leave_type.manage`).
- **The cap is per request, not per year.** Article 115 kinds are granted per event; each event is its own request. No running total across requests is kept.
- **Exceeding the cap always refuses** (`leave_days_exceed_type_max`, carrying `max_days`). There is no per-type choice between refusing and warning. The excess goes in a separate request of another type (annual or unpaid leave), which is what the company pays for anyway. The cap applies on top of the working-days cap of ADR-0030; both read the same `days`.
- **A missing attachment refuses when the request leaves draft** (`leave_attachment_required`). A draft saves without files, so the employee can enter the request first and upload the certificate after.
- **Both checks run in two places, as in ADR-0030.** The cap is checked on entry (create and edit, in `leaveHeader`) for immediate feedback; the attachment in `BeforeSubmit`, which runs on every send, with or without an approval rule. Both are checked again in `leaveTransition` on the move to `posted`, since the type may have been tightened after the draft was saved and a file may have been removed while the request waited for a decision. The attachment check reads `attachment.files` by the request's document type and id; HRM may read the lower `core` tier directly.
- **Rules apply forward.** Editing a type's cap or attachment flag does not revisit approved requests.

## Rejected alternatives

- **Warning instead of refusing, or a per-type choice.** For the same reason ADR-0030 refuses: the overrun is paid money and a warning changes nothing the person entering the request can do. A per-type switch adds a setting every tenant must understand, for a case the "separate request" path already covers.
- **A per-year cap.** Needs a running total over approved requests with its own lock and its own refund on cancel, like the leave balance. No kind in scope needs it; a tenant wanting a yearly allowance uses a deducting type with a balance.
- **Requiring the attachment from creation.** It would force the employee to have the file in hand before recording the absence, which is backwards for sick leave.
- **Requiring it only on approval.** The approver would decide without the evidence in front of them.

## Consequences

- The leave type form gains the two fields; the leave form shows the cap of the chosen type and that a file is needed, so neither rule is invisible until the error appears.
- The demo seed adds the Article 115 kinds with their caps, and sick leave with `requires_attachment`.
- Requests captured before this ADR may exceed a cap set later; they are not revisited, but approving one that is still pending is refused until it is shortened.
