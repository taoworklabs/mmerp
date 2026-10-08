# 0009. `iam`, `setting`, `audit`: the parts used for sign-in

Status: accepted · 2026-10-05

## Context

Sign-in and the app shell need the first three core modules. These three modules are built just as far as sign-in needs; the org tree, roles, per-legal-entity settings and field change history are added later, in the same modules. This ADR records what is hard to change later.

## Decision

- **`core` tier**, with the dependency DAG `iam → setting, audit`. No module exposes hooks. (Changed: `iam` exposes `TreeChanged`, see ADR-0014; `setting` has an `Authz` hook, see ADR-0021.) No record types or business lines.
- **Ids are auto-increment `bigint`** (`GENERATED ALWAYS AS IDENTITY`) for every table, no UUIDs. With one database per tenant, ids are never merged across databases; ids revealing record counts is acceptable for an internal app.
- `iam` owns:
  - `iam.users (id, login, name, password_hash, locale, authz_version)`. `login` is unique and case-insensitive; it need not be an email, since many employees have no work email. An empty `locale` falls back to the tenant's default language.
  - `iam.sessions (token_hash, user_id, expires_at)`. The cookie carries a random 128-bit token; the database stores only the token's SHA-256, so a leaked backup does not leak sessions. A session expires 14 days after sign-in, with no extension on activity. Sign-out or revocation deletes the row.
  - Passwords are hashed with argon2id, stored as a PHC string (`$argon2id$v=19$m=…,t=…,p=…$salt$hash`) so parameters can change without a migration.
- `authz_version` sent to the frontend is an **opaque string**: the frontend only compares for equality. For now the string holds only the user's number; when a tenant-wide number exists it is appended without the frontend having to change. (Appended: `<user number>.<tenant number>`, see ADR-0010.)
- The **actor** is the user id held in the context by `platform`. `audit` reads the actor from there, so it does not have to import `iam`.
- `setting` owns `setting.values (key, value)`, with string values; a key with no row takes the default declared in code. For now there are only `setting.locale` (default `vi`) and `setting.timezone` (default `Asia/Ho_Chi_Minh`). Per-legal-entity settings add a column when needed. (Replaced: a separate table `setting.legal_entity_values`, see ADR-0021.)
- `audit` owns `audit.log (id, at, actor_id, action, data)`; ADR-0011 adds `doc_type`, `doc_id`. `action` is prefixed with the module (`iam.login`, `iam.login_failed`, `iam.logout`). No foreign key to `iam.users`, because `audit` sits below `iam`.

## Personal data

| Table | Field | Kind |
| --- | --- | --- |
| `iam.users` | `login`, `name` | Personal data |
| `iam.users` | `password_hash` | Secret; never returned by the API or logged |
| `audit.log` | `data.login` of `iam.login_failed` | Personal data (the login that was typed) |

No sensitive fields yet.

## Consequences

- The frontend treats ids as numbers. JSON keeps integers exact up to 2^53, enough for every table.
- No brute-force protection yet (temporary lockout after repeated failures). It must be added before the app is exposed outside the organisation's internal network.
- The cookie has the `Secure` flag, so outside `localhost` the app must run behind HTTPS. The on-premise installation docs must say so.
