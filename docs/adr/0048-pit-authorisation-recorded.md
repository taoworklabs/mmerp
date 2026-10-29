# 0048. PIT finalisation authorisations are recorded per employee and year

Status: accepted · 2026-10-29

## Context

The 05/QTT-TNCN and 05-1/BK-QTT-TNCN reports finalise the year's tax for each authorising employee. The first release took everyone on the legal entity's posted December payroll to be authorising. That is a guess, not the fact the tax office asks for: an employee signs form 08/UQ-QTT-TNCN to authorise one organisation for one year, may decline, and someone who left in October but gets back pay on the December payroll would be counted.

## Decision

- `hrm.pit_authorisations` holds (employee, year, legal entity), one legal entity per employee per year.
- HR records the set for a legal entity and year on its own screen, choosing among everyone on one of its posted payrolls of the year; who already authorised another legal entity that year is refused (`pit_authorised_elsewhere`). Saving replaces the set and is audited (`hrm.pit_authorisations_changed`).
- Reading needs `hrm.salary.view` at the legal entity, writing `hrm.payroll.edit`. It is not a document: no number, status or approval.
- The signed form is evidence, attached to the employee's profile with the existing attachments; the system does not require it.
- The finalisation reports count as authorising who is recorded and has a line on a posted payroll of the year. December no longer decides anything; the screen offers picking those on December's payroll as a shortcut.

## Consequences

- A year finalised before this change shows nobody authorising until HR records the set.
- The finalisation follows a recorded decision, not an inference from pay.
