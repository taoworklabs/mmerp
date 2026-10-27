# 0041. Account lifecycle, login security and the root account

Status: accepted · 2026-10-27 · extends [ADR-0009](./0009-iam-setting-audit.md) (sessions, brute-force protection), keeps [ADR-0028](./0028-tenant-admin-holds-business-permissions.md)

## Context

Accounts could sign in but not be managed: no password change or reset, no way to lock out someone who leaves, no way to end a session, and repeated wrong passwords only produced audit rows. ADR-0009 required brute-force protection before the app leaves the internal network. Users are referenced by audit and documents, so the lifecycle choices below are hard to reverse.

## Decision

- **No user deletion.** `login` is immutable; the display name can be renamed.
- **Locked account.** An administrator (`core.user.manage`) can lock a user. A locked user cannot sign in and every session is deleted at once. The user still resolves into approval seats (by role, by name, through delegations), still receives notifications, and jobs they queued still run as them; their name stays on past records. Unlocking restores sign-in with the same password.
- **Forced password change.** After an administrator resets a password (a temporary one the administrator types, no e-mail link) and for the root's first password, the user must change it at the next sign-in and can do nothing else until then. A reset ends every session. Changing one's own password keeps the current session and ends the others.
- **Temporary lock.** After N consecutive wrong passwords for a typed login (default 5), that login is locked for M minutes (default 15). The count is per lowercased login, not per IP (a company often shares one IP), and includes unknown logins, which behave exactly like known ones, so the lock cannot reveal which logins exist. Check order: temporary lock, then the password, then the permanent lock; a locked account is reported only after a correct password. A success resets the counter; unlocking an account clears it. Every temporary lock is audited. It applies to the root too.
- **Password policy.** Tenant-wide: minimum length 8 to 64 (default 8), optional required classes (upper, lower, digit, symbol; all off by default), and always "must not contain the login or name". It is checked only when a password is set (create user, change, reset, root's first password); existing passwords keep working when the policy tightens. A rejection returns one error code listing every failed rule.
- **Root account.** The tenant has exactly one root: login `root` (reserved), tenant-wide `core.admin`. At startup, after migrations, the server creates it from `ROOT_PASSWORD`; without a root and without a valid `ROOT_PASSWORD` it refuses to start. Once the root exists the variable is ignored. Through the UI or API nobody can lock the root, revoke its `core.admin`, or reset its password (stable error codes); it can change its own password, end its own sessions and be renamed. Because root always remains, other administrators may be locked or demoted freely. The root keeps the sensitive exception of ADR-0028: it grants itself the sensitive role to read salaries or identity numbers, and every read is audited.
- **Recovery.** `mmerp create-admin` is replaced by `mmerp reset-password <login>` (password on stdin): it sets the password, clears temporary and permanent locks, and ends every session. Shell access to the server is already trusted.

## Consequences

- Install docs, compose, `make reset-db`, e2e and seed move to `root` plus `ROOT_PASSWORD`. Existing admins stay ordinary admins; a database holding a non-root user with login `root` fails startup.
- A locked user is not filtered out of approvals, notifications, pickers or jobs; reassigning work is done by editing rules and roles.
- The failure table is keyed by typed login and kept bounded by deleting rows whose window has passed.
- Out of scope: password expiry or history, 2FA, SSO, reset by e-mail, per-IP limits.
