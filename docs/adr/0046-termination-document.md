# 0046. Termination is a document that owns the employee's termination date

Status: accepted · 2026-10-28

## Context

An employee's `termination_date` is a profile field anyone with `hrm.employee.edit` can type. The payroll reads it to stop pay and to pay untaken leave. Severance and job-loss allowances (Art 46–47 Labour Code 2019) also hang on it, and on a reason, a decision and a few months figures from the social insurance book. HR needs these recorded, approved and printable, and the payroll must not see an allowance appear or vanish under a closed pay period.

## Decision

- A termination is an `hrm.termination` document (`record`, approval, audit, attachments). Its date is the termination date.
- Only posting a termination sets `hrm.employees.termination_date`; cancelling restores the value it replaced. Editing a profile or importing profiles refuses a changed termination date (`termination_via_document`); creating a profile may still carry one, for former employees loaded without documents.
- At most one non-cancelled termination per employee.
- Its affected range for the payroll lock is [termination date, ∞): posting and cancelling take the payroll lock `FOR SHARE` and are refused over a closed pay period, then lock the employee row.
- The allowance is computed by the payroll of the termination month from inputs it captures: the termination's non-sensitive figures (reason, months, pension flag) in the hashed basis, the six months of contract amounts decrypted and captured encrypted in the payroll line. The termination row never stores an amount.
- Reasons are fixed codes mapped to Art 34 clauses, each with the allowance it pays, in code; they are stable so other reports can read them.

## Consequences

- One source of truth for new terminations; legacy dates stay until a termination replaces them.
- A termination already paid by a closed payroll cannot be cancelled; a correction follows the existing back-pay rules.
- A termination entered after its period's payroll closed cannot be posted until that payroll is cancelled (allowed only before pay-out); after pay-out the allowance does not go through mmerp yet.
- HR can no longer fix a termination date on the profile; they cancel and recreate the termination.
- The payroll basis grows by the leaving employees' termination figures, so a changed termination marks a draft payroll stale like any other source.
