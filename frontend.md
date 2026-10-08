# Frontend

A single React app (see [techstack.md](./techstack.md#frontend)). This document describes how the app is composed and isolated; UI rules (page templates, tokens, tables, forms, text) live in [ui.md](./ui.md). Each product is an **area** inside the app, not a separate app. The frontend follows the same shape as the backend ([backend.md](./backend.md)):

| Frontend | Backend counterpart |
| --- | --- |
| `app/` | `internal/app`: composition root |
| `shared/` | `platform` + the shared part of `core`: knows no business logic |
| `core/` | Screens of the core modules |
| `<product>/` (area) | That product's modules in `modules/` |
| `index.ts` (manifest) | `module.go` |

## Structure

```
web/src/
├── main.tsx            # only calls app/bootstrap
├── app/                # composition root: router, providers, layout shell, assembling the areas
│   ├── areas.ts        # list of every area's manifest — the only place that knows which areas exist
│   └── bootstrap.tsx
├── shared/             # knows no business logic, knows no area exists
│   ├── api/            # client generated from OpenAPI, one file per product
│   ├── area/           # AreaManifest type, record type table, useRecordLink, useInvalidateRecord
│   ├── auth/           # /me, useCan, session lifecycle
│   ├── document/       # shared document features
│   ├── jobs/           # background jobs: ExportButton, ImportDialog, JobOutcome, useJobStatus
│   ├── org/            # OrgTree: read-only org tree
│   ├── ui/             # theme.ts, page templates (page/), form/, DataTable… — see ui.md
│   ├── url/            # useListParams: filter, sort, page in the URL
│   └── i18n/           # i18next, formatting functions (date, month, number), text for API error codes
├── core/               # sign-in, users and roles, org tree, settings, approval inbox, period lock
│                       # always present; structured like an area
└── hrm/                # area of the HRM product
    ├── index.ts        # manifest — the only thing imported from outside (core also exports HomePage, InboxPage, JobsPage, LoginPage, nextFromLocation)
    ├── keys.ts         # the area's query key factory
    ├── routes.tsx
    ├── pages/  components/  hooks/
    └── i18n/
        ├── vi/meta.json  vi/main.json
        └── en/meta.json  en/main.json
web/lint-fixtures/      # files that deliberately break the isolation rules, used to verify the configuration (see below)
```

## Isolation

```
app ──► core, hrm, … ──► shared
         area ↮ area: never import each other
```

The rules are checked by **dependency-cruiser** on the resolved dependency graph, not on import strings. So relative paths (`../../hrm/pages/x`), aliases (`@/hrm/pages/x`), re-exports (`export … from`) and dynamic imports (`import(…)`) are all checked the same way. The rules are written as directory patterns, so adding a new area does not require changing the configuration.

| Rule | Notes |
| --- | --- |
| An area does not depend on another area or on `app` | Including through dynamic imports or re-exports |
| Outside an area, only its `index.ts` may be depended on | `app` imports `@/hrm`; nobody imports deep into `hrm/pages/…` |
| `shared` depends on no area, including `core`, and not on `app` | |
| Area `X` uses only `shared/api/core` and `shared/api/X` | Never calls another product's API directly (see [Calling the API](#calling-the-api)) |
| No global store shared between areas | Convention: server state lives in TanStack Query, UI state lives in components |

**Verifying the configuration.** `web/lint-fixtures/` holds a small directory tree with each kind of violation: a deep relative import into another area, an import through an alias, a dynamic import of another area, a re-export of an area through `shared`, and a call to another product's API client. `make lint` runs dependency-cruiser twice: on `web/src` there must be no errors; on `web/lint-fixtures` it must report **every one** of the listed violations. If a configuration change lets a kind of violation slip through, CI fails.

## Manifest: the entry point of an area

Each area exports exactly one manifest. The manifest and what it imports directly (`keys.ts`, the `meta` translation files) must be small, because they are loaded at startup. Screen code is loaded lazily only through `routes`.

```ts
// web/src/hrm/index.ts
import { IconCalendarEvent, IconCash, IconLayoutDashboard, IconUsers, IconUsersGroup } from '@tabler/icons-react'
import type { AreaManifest } from '@/shared/area'
import { hrmKeys, leaveDocType, payrollDocType } from './keys'

export const hrm: AreaManifest = {
  product: 'hrm',
  label: 'hrm.nav.group',
  basePath: '/hrm',
  routes: () => import('./routes'),                       // lazy-loaded
  nav: [
    { label: 'hrm.nav.overview',  path: 'overview',  icon: IconLayoutDashboard },
    { label: 'hrm.nav.employees', path: 'employees', icon: IconUsers, permission: 'hrm.employee.view', group: 'hrm.nav.group_people' },
    // Everyone submits their own leave requests: no permission needed to see this item.
    { label: 'hrm.nav.leaves',    path: 'leaves',    icon: IconCalendarEvent, group: 'hrm.nav.group_time' },
    { label: 'hrm.nav.payrolls',  path: 'payrolls',  icon: IconCash, permission: 'hrm.payroll.view', group: 'hrm.nav.group_pay' },
  ],
  collapsibleGroups: ['hrm.nav.group_config'],
  homeIcon: IconUsersGroup,
  i18n: {
    meta: { vi: () => import('./i18n/vi/meta.json'), en: () => import('./i18n/en/meta.json') },
    main: { vi: () => import('./i18n/vi/main.json'), en: () => import('./i18n/en/main.json') },
  },
  recordTypes: {
    [leaveDocType]: {
      path: (id) => `leaves/${id}`,
      preview: () => import('./components/LeavePreview'),    // lazy-loaded; used in core's approval inbox
      invalidate: (id) => [hrmKeys.leaves.detail(id), hrmKeys.leaves.lists(), hrmKeys.balances.all()],
    },
    [payrollDocType]: {
      path: (id) => `payrolls/${id}`,
      preview: () => import('./components/PayrollPreview'),
      invalidate: (id) => [hrmKeys.payrolls.detail(id), hrmKeys.payrolls.lists()],
    },
  },
}
```

```ts
// web/src/app/areas.ts
import { core } from '@/core'
import { hrm } from '@/hrm'
// Product areas first, core (administration) always last.
export const areas = [hrm, core]
```

| Field | Used for | Backend counterpart |
| --- | --- | --- |
| `product` | Matched against the enabled products in `/me` | The module's product |
| `label` | Translation key of the area name: header title and tile name on the home page | — |
| `homeIcon` | Icon of the tile on the home page | — |
| `routes` | The area's route tree, lazy-loaded | `Routes` |
| `nav` | Menu items: label, path, icon, permission required to show it (one, or one of several), and group | — |
| `collapsibleGroups` | Menu groups the user can collapse | — |
| `i18n.meta` | Every string needed before the area's screens load: menu labels, role names (`<product>.role.<role>`, needed by `core`'s user screen), record type names; for `core` also the sign-in screen and the app shell. Loaded at once for every visible area | — |
| `i18n.main` | All other strings of the area; loaded with the screen code | The module's translation files |
| `recordTypes[…].path` | Path to a record's detail page | Registering the record type with `record` |
| `recordTypes[…].preview` | Read-only quick-view component, lazy-loaded; used in the approval inbox and `core` panels | — |
| `recordTypes[…].invalidate` | Query keys to refresh when a record of this type changes | Hook |

### Opening another area's document

The approval inbox, notifications and history belong to `core`, but must be able to open the detail of a leave request belonging to `hrm` without importing `hrm`. `app` collects the `recordTypes` of every manifest into one table and passes it down through context. `shared/area` provides:

```ts
const link = useRecordLink('hrm.leave_request', id)   // '/hrm/leaves/123', or null if the area is not visible
```

`core` knows only the `doc_type`. Whichever area declares the path, `app` plugs it in, just as hooks are wired in the backend.

Quick-viewing a document (for example the right-hand panel of the approval inbox) uses the same mechanism:

```tsx
<RecordPreview docType="hrm.leave_request" id={id} />   // lazy-loads the preview component the area declared
```

- An area's `preview` is a **read-only** component that receives `id` and calls its own area's API. It holds no action buttons: approval actions live in `<ApprovalPanel>` from `shared/document`, which `core` places next to the preview.
- If an area declares no `preview`, `<RecordPreview>` shows only a link that opens the detail page.

## Calling the API

`make gen` generates a type file `schema.gen.ts` from Huma's OpenAPI. **One client per product**: the same openapi-fetch client, but typed with only that product's paths (`/<product>/…`); the `core` client is typed with all remaining paths. Calling another area's endpoint is a `tsc` error.

```
shared/api/
├── client.ts     # shared openapi-fetch: cookie, session signal, reads X-Authz-Version, handles 401; typed per product
├── error.ts      # ApiError: error code with parameters
├── core.ts       # core client (/me, /users, /org-units, …)
└── hrm.ts        # hrm client (/hrm/…)
```

Language does not travel with the request: the server reads the user's language from their profile. The text for every API error code lives in `shared/i18n/locales` (`shared.error.<code>`), because error codes are part of the API interface and do not belong to any one area.

- An area that needs another product's data gets it through the backend (deps), not by calling the other product's API directly.
- API errors are error codes with parameters ([techstack.md](./techstack.md#i18n)). `client.ts` turns them into typed errors; components never parse error bodies themselves.

## Cache

### Query keys

- Each area has a **query key factory** in `keys.ts`. Never write query keys as literal arrays inside components.
- Keys always start with the product name, so one area never reads or clears another area's cache by mistake.
- Keys contain **every parameter that changes the result**: filters, sort, page or cursor, period, selected org unit. Two screens calling the same endpoint with different parameters have different keys.

```ts
// web/src/hrm/keys.ts
export const hrmKeys = {
  leaves: {
    lists:  ()                    => ['hrm', 'leaves', 'list'] as const,
    list:   (f: LeaveFilter)      => ['hrm', 'leaves', 'list', f] as const,   // f includes filter, sort, page
    detail: (id: number)          => ['hrm', 'leaves', 'detail', id] as const,
  },
  balances: { all: () => ['hrm', 'balances'] as const },
  payrolls: { lists: () => ['hrm', 'payrolls', 'list'] as const,
              detail: (id: number) => ['hrm', 'payrolls', 'detail', id] as const },
}
```

User, permission scope and language are not in the key, because the cache lives only within one session (see below).

### Session lifecycle

The cache belongs to exactly one signed-in session, with one language and one **authorization version**. `shared/auth` holds the app state: `anonymous` (not signed in) or `authenticated` (has `/me`).

**401 handling depends on state**, so there is never a reload loop:

| Situation | Handling |
| --- | --- |
| Bootstrap calls `/me`, gets 401 | State `anonymous`: show the sign-in screen. **No** reload |
| Wrong credentials at sign-in | Show the error on the sign-in form. **No** reload. `client.ts` does not apply session-expiry handling to the sign-in endpoints |
| While `authenticated`, a request gets 401 (session expired or revoked) | Call `endSession()` |
| Sign-out | Call the sign-out API, then `endSession()` |
| Successful sign-in | Reload the page so bootstrap runs again from the start with the new session |

`endSession()` runs only **once** per session (a flag prevents re-runs): it cancels every in-flight request, clears the whole cache, broadcasts `session_ended` over `BroadcastChannel`, then reloads the page to `/login?next=<current path>`. Reloading the page is deliberate: it clears all in-memory state, not just the cache. No loop can occur, because after the reload the app is `anonymous`, and a 401 then only leads to the sign-in screen.

**Other tabs:** a tab that receives `session_ended` or `session_started` reloads itself, and **does not rebroadcast** the message. A tab on the sign-in screen that receives `session_started` also reloads, to go straight into the new session.

**Switching accounts** is signing out and back in, so it takes the same path. **Changing language:** save it to the server, then reload the page.

**Permissions changing mid-session.** `/me` returns `authz_version`, and every API response carries the `X-Authz-Version` header ([platform.md](./platform.md#identity-and-permissions)). `client.ts` compares the header with the value received from `/me`; if they differ, it cancels every in-flight request, clears the whole cache, and reloads the page. So payroll data already in the cache can no longer be displayed after the permission to view payroll is revoked: the next request, to whatever endpoint, detects the change. This only clears stale data in the UI. The backend still checks permissions on every request, so fresh data is never returned to someone who no longer has the permission.

### Refreshing after writes

- When a document changes, **the area that owns the document type decides which caches to refresh**, through `recordTypes[…].invalidate` in the manifest. `shared` does not guess business logic.
- `useDocumentMutation`, and every `core` screen that writes to documents (the approval inbox), call the function returned by `useInvalidateRecord()` after a successful write: `const invalidate = useInvalidateRecord(); await invalidate(docType, id)`. This function refreshes the keys the area declared, plus the related `core` keys (approval inbox, history).
- Example: approving a leave request from `core`'s approval inbox refreshes HRM's request detail, request list and leave balances, even though `core` does not know what a leave balance is.
- Writes not related to a document (for example editing an employee profile) have the area refresh its own keys.
- Changes caused by a job or hook in the backend (for example a posting job) are not refreshed immediately. `staleTime` (default 30 seconds) only marks data as stale; **it does not reload on a timer**. Stale data is reloaded when the screen is opened again, when the tab regains focus, when the network reconnects, or when it is refreshed after a write.
- **Screens that track a background job** (Excel import, data export): use `useJobStatus(jobId)` from `shared`. This hook polls the status while the job has not finished: every 2 seconds for the first 30 seconds, then every 10 seconds. When the job finishes it stops polling, then refreshes the related keys. Apart from this, no screen reloads on a timer.
- Queries that write an audit row on every load (dependants, sensitive values) disable refetching when the tab regains focus (`refetchOnWindowFocus: false`), so they do not produce audit rows the user did not actively view.
- Lists whose data is changed by background jobs (for example posting status) have an explicit "Làm mới" ("Refresh") button.

## Permissions: the frontend computes nothing

The backend is the only place that decides permissions. The frontend only reads the result.

- **Menu and page level:** `/me` returns the enabled products, the areas that already have data, the user's permissions per product, and the language. `useCan('hrm.payroll.view')` is used only to hide or show menus and pages.
- **Record level:** every API that returns a document includes `allowed_actions` ([documents.md](./documents.md#permissions)). The frontend renders buttons from this list and **never derives them from status or role**.
- **Collection level:** actions not tied to a record yet (create, import, export) come from `GET /<product>/<x>/actions`, query key `hrmKeys.<x>.actions()` (for example `useEmployeeActions`). The "Create" button shows only when `create` is present; a create page without that action shows `<ForbiddenPage>`.
- **Route guards:** an area's `routes.tsx` wraps routes that need a permission with `useCan(…) ? <Page /> : <ForbiddenPage />`, matching the menu item's `permission`. Detail pages use `loaded()` (`hrm/components/loaded.tsx`): a skeleton while loading, a 404 (including a record outside the scope) becomes a `NotFoundPage` linking back to the list, other errors become `ErrorState`.
- A disabled product needs no special logic in the frontend. The backend removes every write action from `allowed_actions` and keeps the export action ([platform.md](./platform.md#enabled-products)), so write buttons disappear on their own while the export button stays. The only exception is catalogue screens without `allowed_actions` (leave types, contract types): write buttons are hidden based on the enabled products in `/me`, through `useProductOn`.
- Hiding in the frontend is not security. Every request is still checked by the backend.

## Shared document features

As in the backend: an area declares what it uses and does not rebuild it. All of it lives in `shared/document`.

| Component / hook | Job |
| --- | --- |
| `<DocumentStatus>` | Status badge, translated |
| `<DocumentActions>` | Renders buttons from `allowed_actions`, calls the transition API, handles `version` conflicts with a "data has changed, reload" dialog |
| `<ApprovalPanel>` | Approval steps drawn as a progress path (approved, pending, not yet reached), the approvers, a button to replace the approver at the pending step, approve or reject buttons for exactly the open approval instance |
| `<DocumentHistory>` | Timeline from audit; sensitive fields are masked if the user lacks permission to view them. Each action has a translated label (from the area that owns the document), never a raw code. Consecutive entries by the same person, with the same action and no changed fields (for example viewing a payroll) are merged into one row with a count |
| `<ApprovalRules>` | List and editor for approval rules; takes `product` (shows only that product's document types) and `basePath` (path of the rules page, since it has its own child routes). A product with document types attaches it to its own Configuration group; the administration area has no approval rules page of its own |
| `useDocumentMutation` | Sends `version` automatically, calls `useInvalidateRecord` after writing, turns error codes (`period_locked`, `version_conflict`…) into translated messages |

The `hrm` leave request screen contains only the business form, plus the components above.

The org tree is shared data: a product embeds the read-only `<OrgTree>` (`shared/org`) in its own screens; the tree is edited only in the administration area.

## Startup flow

1. `main.tsx` calls `bootstrap`.
2. **Pick a provisional language:** the language used last time (stored in `localStorage`); if there is none, the browser's; if that is neither `vi` nor `en`, use `vi`. Load the minimal translations: `shared`'s common error codes and `core`'s sign-in screen.
3. Call `GET /me`. If not signed in, show the sign-in page, which now has its translations. After sign-in, reload the page.
4. If the user's language differs from the provisional one, switch and reload the minimal translations; save the language to `localStorage`.
5. Filter `areas`: keep `core`, areas whose `product` is enabled, and areas that already have data (`me.products_with_data`). Load `i18n.meta` of the kept areas. Each area's `i18n.main`, including `core`'s, loads with that area's screens.
6. Build the router (React Router, data mode): `core`'s routes, plus each area's lazy-loaded routes under its `basePath`. Render the layout shell; the navigation bar holds only the `nav` of the open area, filtered by permission (`can(me, item.permission)`). The home page, approval inbox and background jobs have no navigation bar ([ui.md](./ui.md#app-shell)).
7. The first time an area is opened: the `lazy` route loads the screen code **and** the area's `i18n.main`, and renders only after both are done.

Changing language within a session: save it to the server, then reload the page (see [Session lifecycle](#session-lifecycle)). Code of an area that is not visible is never loaded. This is a speed optimisation, not security.

## Adding an area

1. Create `web/src/<product>/` with `index.ts`, `keys.ts`, `routes.tsx`, and `i18n/{vi,en}/{meta,main}.json`.
2. Tag the endpoints with the product in the backend, and run `make gen` to get `shared/api/<product>.ts`.
3. Declare `recordTypes` for every record type that has a detail screen, with `path` and `invalidate`; document types with approval also declare `preview`.
4. Add the manifest to `app/areas.ts`, before `core`.
5. No dependency-cruiser configuration change is needed: the rules are written as directory patterns.
6. Use `shared/document` for every document feature; never build your own status, approval or history buttons. Every screen follows [ui.md](./ui.md) and passes the checklist at the end of that document.
7. Write Playwright tests for the area's flows where a bug has serious consequences ([techstack.md](./techstack.md#testing-and-quality)).
