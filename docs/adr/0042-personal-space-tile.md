# 0042. Personal space: an area may export several manifests, one home tile each

Status: accepted · 2026-10-27 · extends the area rules of `docs/frontend.md`

## Context

An employee who is not HR needs a home for their own data (attendance, payslips, leave, profile). It belongs to HRM (same API, same product gate), but it is a different audience and menu from the HR screens. Today an area is one manifest, so one tile, and `HomePage` shows a tile when the user holds a permission in the area (`web/src/shared/area/index.ts`, `web/src/core/pages/HomePage.tsx`). An ordinary employee holds none, and the right to a personal space is not a permission: it is being linked to an employee profile.

## Decision

- A product area may export more than one `AreaManifest`. Each is one home tile with its own `basePath`, nav and header title.
- A manifest may declare `useEntitled: () => boolean | undefined`. It decides the tile from the product's own API (`undefined` while loading) and replaces the permission rule for that manifest. The personal space uses it to ask whether the user is linked to an employee.
- `recordTypes` stay on the product's main manifest only, so record links and previews keep one owner.
- Areas still never import each other; both manifests call only the product's own API client.

## Consequences

- `HomePage` and the router treat each manifest alike; a hook-based check is why `useEntitled` is a hook, not a plain function.
- The tile is only a convenience: the server decides what the personal endpoints return (own data only, from `hrm.employees.user_id`).
