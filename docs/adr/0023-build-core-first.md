# 0023. Build out the core first

Status: accepted · 2026-10-07 · replaces the principle "the core follows the first product" ([README.md](../../README.md#principles)) and the sentence "the core is built to exactly what HRM needs" of [ADR-0007](./0007-hrm-product.md)

## Context

The core was built to exactly what HRM needs, and M10 (on-premise operations) together with the production-readiness gate (the checks before real data goes in: accountant review of payroll, configuration and performance targets) were the next steps after M9. The common document features the README promises (attachments, discussion, print templates) and notifications are still missing, and every business-management suite is expected to have them.

## Decision

- **The core is built ahead of production use**: attachments and discussion, notifications, PDF printing (M11 to M13 in the roadmap).
- **HRM still proves it.** Each core capability is done only when at least one HRM screen really uses it, with a test running from the API to the screen. No API is built that no screen calls.
- **M10 and the production-readiness gate are on hold** until a later decision resumes them. No part of M10 is pulled forward.
- The data rule stays: an organisation's real data enters the system only after M10 is done and the gate is passed.
- Every new core module still needs its own ADR before code.

## Consequences

- The core grows before any feedback from real use, so there is a risk of designing by guesswork. Proving each capability with HRM is how that risk is bounded.
- Production use moves back by at least the time of M11 to M13 plus M10.
- Custom fields, accounting and a second product remain outside the roadmap.
