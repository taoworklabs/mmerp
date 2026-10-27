# Accounts that work for real: `iam` account lifecycle, sessions, login security

Date: 2026-10-27 · Status: draft, awaiting approval

## Intent

`core/iam` today signs people in, creates users, sets e-mail and locale, and checks roles. A real customer runs into the gaps on the first day: nobody can change a password, a forgotten password cannot be reset, someone who leaves the company cannot be locked out, an open session cannot be ended, and repeated wrong passwords only produce audit rows (ADR-0009 already says brute-force protection must come before the app is exposed beyond the internal network).

This work makes accounts usable for real. It does not depend on which product comes next and stays inside `iam` (plus its screens). Custom roles and permission dimensions beyond the org tree (issue #3) wait for the next product; API keys wait for a real integration.

## Goals

### Self-service (any signed-in user)

- **Change own password**: requires the current password. The new one must pass the password policy. Afterwards every other session of the user is deleted; the current session stays.
- **See own open sessions**: when each was opened, plus a short device/browser label taken from the User-Agent at login. The current session is marked.
- **End sessions**: end one session, or every session except the current one.

### Administration (holders of `core.user.manage`)

- **Reset a password**: the admin types a temporary password (it must pass the policy). The user must change it at the next sign-in, and until then can do nothing else. Every session of that user is deleted. No reset link by e-mail: on-premise must not depend on the Internet, and e-mail is optional.
- **Lock / unlock an account**: a locked user cannot sign in, and all their sessions are deleted at once. Their name keeps showing on past documents, audit and history. Unlocking restores sign-in; the password is unchanged.
- **Rename**: the display name can be edited. `login` stays fixed.
- **No deleting users**: users are referenced by audit and documents.
- **Root account**: the tenant always has exactly one root account, holding tenant-wide `core.admin`. Through the UI or API nobody can lock it, revoke its `core.admin` grant, or reset its password; each attempt is refused with a stable error code. The root user can still change their own password and end their own sessions. The root keeps the sensitive-permission exception of ADR-0028 like any admin: to read salaries or identity numbers it grants itself the sensitive role, and every read is audited. This replaces a "last admin" guard: other admins can be locked or demoted freely, because root always remains. The UI marks the root account and hides the actions it refuses.

### Creating the root

- On startup, after migrations, if no root exists the server creates it: login `root`, tenant-wide `core.admin`, password from `ROOT_PASSWORD` (`platform.LoadConfig`). The root must change that password at its first sign-in *(my call)*, so the value left in the compose file or environment stops working.
- If no root exists and `ROOT_PASSWORD` is unset or fails the password policy, the server refuses to start with a clear message, the way it does without `ENCRYPTION_KEYS`. Once the root exists, `ROOT_PASSWORD` is ignored; changing it resets nothing *(my call)*.
- The login `root` is reserved: creating another user with it is refused *(my call)*. If a database already has a non-root user with login `root`, startup fails with a clear message rather than guessing *(my call)*.
- `mmerp create-admin` is removed; the install docs, compose file, `make reset-db`, e2e and seed scripts move to `root` + `ROOT_PASSWORD` *(my call)*. Existing admins in a database stay ordinary admins. There is no customer yet, so no data migration is needed beyond creating the root.
- The root's display name starts as `root` and can be renamed.

### Password policy (tenant-wide, set by an admin)

- Minimum length: 8 to 64, default 8 (today's `MinPasswordLength`).
- Required character classes, each toggled on or off, all off by default: uppercase, lowercase, digit, symbol.
- Always on, not configurable: the password must not contain the user's login or name (case-insensitive; skip name parts that are too short to matter, the builder picks the cut-off).
- The policy is checked **only when a password is set**: create user, change own password, admin reset, and the root's first password. Existing passwords keep working when the policy gets stricter; nobody is forced to change.
- A rejected password returns one stable error code whose parameters list every rule that failed (for example `min_length` with its value, `digit`, `contains_login`), so the screen shows one line per unmet rule.

### Login lockout (tenant-wide, same settings screen)

- After N consecutive wrong passwords for a login (default 5), that login is temporarily locked for M minutes (default 15). Both are configurable.
- Counted **per login, not per IP**: on-premise, the whole company often shares one IP.
- While temporarily locked, sign-in fails even with the right password, with its own error code so the screen can say "try again later". A successful sign-in resets the counter. An admin unlocking the account (above) also clears a temporary lock.
- An unknown login must behave like a known one (same timing as today's `dummyHash` path, same error), so the lockout cannot be used to discover which logins exist.
- Every temporary lock is audited.

### Cross-cutting

- Every action above is audited as the existing `iam` actions are (`iam.password_changed`, `iam.password_reset`, `iam.user_locked`, …; the builder names them). Audit data never holds a password or hash.
- Bump `authz_version` where the frontend must refresh (lock state, rename, must-change-password).
- Every new text goes through i18n in `vi` and `en`; every error is a stable code with parameters.

## Non-goals

- Password expiry, password history, 2FA, SSO/LDAP, reset by e-mail link.
- Custom roles, new permission dimensions (issue #3), API keys.
- Forcing users whose password no longer meets a stricter policy to change it.
- Per-IP rate limiting.

## Anti-goals

- Anything that needs the Internet or an extra service to sign in or recover an account.
- A lockout that lets an attacker tell existing logins from unknown ones, or lets anyone take away the root account through the UI or API.
- Locking a user and leaving one of their sessions alive.

## Constraints

Everything in `AGENTS.md` applies. In particular: one Go binary + Postgres; queries through the data engine; writes in `platform.InTx`; new migrations via `make migration`; no editing the baseline; the screens follow `docs/ui.md` and the `mmerp-ui` skill. ADR-0009 (sessions, argon2id, `authz_version`) and ADR-0028 (tenant admin) stay in force. The account lifecycle (locked state, no deletion, forced change after reset, the root account) is hard to reverse, so it gets a short ADR before the code, and new terms (locked account, temporary lock, password policy, root account) go into `CONTEXT.md` in the same change.

## Decided with the user

- Scope is the three groups above (account lifecycle, sessions, login security), plus the configurable password policy.
- Admin reset sets a temporary password that must be changed at next sign-in; no e-mail link.
- A stricter policy applies only to the next password set; no forced change.
- One root account with full administration that the UI and API cannot remove; it does not bypass the sensitive-permission rule (ADR-0028 stays as is).
- The root is always created, with no CLI step: its login is always `root`, and its first password comes from an environment variable (`ROOT_PASSWORD`).
- Locking an account only blocks sign-in and ends sessions. A locked user still resolves into approval seats (by role, by name in rules, through delegations), still receives notifications, and jobs they queued still run as them; reassigning their work is done by editing rules and roles. Leaving locked users out of `approval`, `notification`, pickers and jobs is a non-goal for this work. The user list and user page show the locked state.

## My calls (the user can override)

- No user deletion; `login` is immutable.
- The root obeys the temporary lockout like anyone; `mmerp reset-password` is its recovery path. Renaming the root is allowed.
- Lockout per login, defaults 5 tries / 15 minutes, configurable with the policy.
- "Must not contain login or name" is always on.
- Changing the own password keeps the current session; admin reset and lock end every session.
- Session list shows opened-at and a User-Agent label; no IP shown.
- Failure counters are keyed by the typed login (lowercased), not by user id, so unknown logins also count and also get the "try again later" error; the builder keeps that table bounded (e.g. delete rows whose window has passed).
- Sign-in check order: temporary lock first (no password check), then the password, then the permanent lock. A locked account gets its own "account locked" code only after a correct password; with a wrong password it gets the usual invalid-credentials error.
- The temporary lockout applies to admins too. Recovery when the admin is locked out or has forgotten their password is a server-side CLI command, `mmerp reset-password <login>` (password on stdin): it sets the password, clears the temporary and permanent lock, and ends every session. Whoever has shell access to the server is already trusted.
- Settings screen lives in the core admin nav, guarded by `core.user.manage`; "change password" and "my sessions" are reached from the user menu in the app header.

## Left to the builder

- Where the policy and lockout settings are stored (tenant keys in `setting.values` behind a new tenant-setting API, or an `iam`-owned single-row table like `iam.authz`), keeping the dependency DAG `iam → setting, audit`.
- Column and table shapes (locked flag or timestamp, must-change flag, failure counter location, session User-Agent column), audit action names, error code names, endpoint paths.
- How the forced-change state is enforced on the server (every endpoint except change-password, logout and `/me` refuses) and shown in the frontend.
- Ticket breakdown inside the one spec branch.
