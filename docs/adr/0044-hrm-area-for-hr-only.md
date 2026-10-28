# 0044. The HRM area is for HR; everyone else works in the personal space

Status: accepted · 2026-10-28 · amends 0042

## Context

ADR-0042 gave employees a personal space, but the HRM area stayed open to anyone who typed its URL: its overview, org chart, leave and overtime items carry no permission, because a direct manager without an HRM role used `/hrm/leaves` to see their reports' requests, and every record link (notifications, the inbox) pointed into `/hrm`. An ordinary employee landed in a cut-down copy of the HR screens.

## Decision

- An area is entered by the same rule that shows its home tile. A user who may not enter it is sent home; the server still decides what any endpoint returns.
- People without an HRM role, direct managers included, work only in the personal space. A direct manager sees their reports' leave and overtime requests, past ones included, under "Nhóm của tôi" there; the menu shows it only when the server says the user has reports (`has_reports` on the self-service actions).
- The leave and overtime lists take `reports=true`: only requests of employees below the actor on the manager chain, the actor excluded, whatever units the actor's roles cover.
- A record type may be declared by more than one manifest of a product (this amends 0042, which kept them on the main manifest only). A record link goes to the first manifest the user may enter that declares the type, or nowhere; previews and refreshes come from the first that declares it.
- A nav item may declare `useShown`, a hook that hides it, for menu entries that depend on the user's data rather than a permission.

## Consequences

- A user who is both HR and an employee sees both tiles; their record links open in HRM.
- An approver without an HRM role opens the request from the inbox in the personal space and approves there, as `allowed_actions` allows.
