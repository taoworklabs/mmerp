# 0005. Enabled products are checked per request; dependencies have no no-op

Status: accepted · 2026-10-05

## Context

The first design mounted only the routes and hooks of the enabled products, once at startup. That does not work under cloud model B, where tenants with different enabled products share one app. That design also attached no-ops to required dependencies when the other product was not enabled, so a required operation could "succeed" while doing nothing.

## Decision

- The architecture relies only on the **list of enabled products**, read from the `PRODUCTS` environment variable.
- Every module is always initialised and mounted. Middleware puts the tenant's list in the context, and every operation goes through a single gate by **action class** (read, export, write), applied the same way to module routes, core routes, `allowed_actions` and jobs. A product that is not enabled or has been disabled stays readable and exportable; writes return 403. System jobs already queued always run to completion; jobs on behalf of a user are re-checked when they run. Disabling a product does not delete data. Hook adapters read the list from the context and become no-ops when the product is not enabled.
- Hooks are optional reactions and may be no-ops. Dependencies in `deps.go` are required and have no no-op. Dependencies between products are declared in code, and the app refuses to start when a depended-on product is missing. A dependency that only affects one feature has its implementation return `ErrUnavailable`, and the frontend hides that feature.

## Consequences

- On-premise and cloud B run the same code path. Another source for the list would only replace where it is read from.
- Whoever runs the installation decides which products are enabled by editing the configuration.
- A module that exposes hooks must test the case where the hook is a no-op.
