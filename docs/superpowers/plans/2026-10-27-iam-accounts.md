# IAM accounts that work for real: implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Password policy, login lockout, account lock/rename, password change and admin reset with forced change, session list and revocation, and an always-present `root` account, in `core/iam` and its screens.

**Architecture:** Everything lives in `internal/core/iam` (new columns on `iam.users` and `iam.sessions`, two new `iam` tables), wired in `internal/app` and `cmd/server`. Pure password rules sit in `password.go`. The frontend adds two personal pages under the root and one admin page under `/admin`, and extends the user pages.

**Tech Stack:** Go, Huma, sqlc, Postgres 18; React, TanStack Query, `@mmerp/ui`, Playwright.

**Spec:** `docs/superpowers/specs/2026-10-27-iam-accounts-design.md`

## Global Constraints

- Branch `feat/iam-accounts`, one commit per task, Conventional Commits with scope `iam`, `web` or `app`; every commit ends with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Stage explicit paths, never `git add -A`.
- New migrations only via `make migration name=iam_<description>`; never edit the baseline.
- Queries through `store.<Query>.Run(ctx, …)`; writes inside `platform.InTx`; run `make gen` after touching `queries/*.sql` or handlers.
- Errors are `*platform.Error` with stable codes and params, never sentences. Every new user-facing string goes into both `vi` and `en`.
- Audit data never holds a password or hash; passwords never reach logs.
- Password policy: min length 8 to 64, default 8; character classes upper, lower, digit, symbol, all off by default; "must not contain login or name" always on.
- Lockout: per typed login (lowercased), default 5 consecutive failures, default 15 minutes; applies to unknown logins and to root.
- Root: login always `root`, first password from `ROOT_PASSWORD`, must change at first sign-in; cannot be locked, have its `core.admin` revoked, or have its password reset through UI/API; keeps the ADR-0028 sensitive exception.
- Before finishing each task: `make lint` and the task's tests pass. Before the PR: `make gen` (no diff), `make lint`, `make test`, `make e2e`.

## Review Focus

1. Two wrong passwords racing at the threshold: the counter must be one atomic upsert, so the lock still lands at exactly N. Pinned in Task 3 (`TestLockoutCountsAtomically`).
2. A locked user whose session row somehow survives (e.g. a lock committed while a login was in flight): `Authenticate` must still refuse. Pinned in Task 4 (`TestLockedUserSessionIsDead`).
3. A must-change user calling any other API directly (not through the UI): refused with `password_change_required`. Pinned in Task 5 (`TestForcedChangeBlocksAPI`).
4. Login case: `ROOT`, `Root`, `root` share one failure counter and one reserved name. Pinned in Task 3 and Task 7.
5. Vietnamese letters in passwords: `Đ` counts as upper, `ư` as lower; a space is not a symbol. Pinned in Task 2 (`TestPasswordProblems`).

---

### Task 1: ADR and glossary

**Files:**
- Create: `docs/adr/0041-account-lifecycle-and-root.md`
- Modify: `CONTEXT.md`, `docs/adr/0009-iam-setting-audit.md` (append a "Changed" note pointing at ADR-0041 where it says no brute-force protection yet)

- [ ] **Step 1: Write ADR-0041** in the style of ADR-0028 (Context / Decision / Consequences). It records: no user deletion; `login` immutable; locked account (sign-in blocked, sessions ended, still resolves into approval seats, notifications and jobs); forced change after admin reset and for the root's first password; per-login temporary lock counting unknown logins; password policy checked only when a password is set; root account (login `root`, created at startup from `ROOT_PASSWORD`, protected operations, keeps the sensitive exception); `mmerp create-admin` replaced by `mmerp reset-password`.
- [ ] **Step 2: Add the terms** "locked account", "temporary lock", "password policy", "root account" to `CONTEXT.md`, matching its existing entry format.
- [ ] **Step 3: Commit** `docs(iam): ADR-0041 account lifecycle and root account`.

### Task 2: Password policy

**Files:**
- Create: migration `iam_login_policy`, `internal/core/iam/policy.go`, `internal/core/iam/password_test.go`
- Modify: `internal/core/iam/password.go`, `queries/user.sql` (or new `queries/policy.sql`), `types.go`, `service.go` (`createUser`), `handler.go`, `service_test.go`, `web/src/shared/i18n/locales/{vi,en}.json`

**Interfaces:**
- Produces:
  - `type LoginPolicy struct { MinLength int \`json:"min_length"\`; Upper, Lower, Digit, Symbol bool (json upper/lower/digit/symbol); MaxFailures int \`json:"max_failures"\`; LockMinutes int \`json:"lock_minutes"\` }`
  - `func (s *Service) LoginPolicy(ctx) (LoginPolicy, error)` (`core.user.manage`), `func (s *Service) SetLoginPolicy(ctx, LoginPolicy) error` (`core.user.manage`, audited `iam.login_policy_set`, bounds: MinLength 8–64, MaxFailures 3–20, LockMinutes 1–1440, else `invalid_request` with `fields`).
  - unexported `func (s *Service) policy(ctx) (LoginPolicy, error)` (no permission check; used by later tasks).
  - `func passwordProblems(p LoginPolicy, login, name, password string) []string` returning, in this order, any of `min_length`, `upper`, `lower`, `digit`, `symbol`, `contains_login`, `contains_name`.
  - `func errPasswordWeak(p LoginPolicy, problems []string) *platform.Error` → status 422, code `password_weak`, params `{"rules": problems, "min": p.MinLength}`. Replaces `ErrPasswordTooShort` and `MinPasswordLength` (delete both).
  - unexported `func (s *Service) checkPassword(ctx, login, name, password string) error` = load policy, return `errPasswordWeak` or nil. Every password set in later tasks calls it.
  - Routes: `GET /login-policy` (`get-login-policy`), `PUT /login-policy` (`set-login-policy`, 204).
- Storage: single-row `iam.login_policy` (one row inserted by the migration with the defaults, like `iam.authz`), columns matching the struct, with CHECKs for the bounds.

- [ ] **Step 1: Write `TestPasswordProblems`** (table test in `password_test.go`, package `iam`, no DB):
  - default policy, `"short"` → `[min_length]`
  - `{MinLength: 8, Upper, Lower, Digit, Symbol}`, `"abcdefgh"` → `[upper digit symbol]`; `"Đường-1234"` → `[]`; `"Abc 12345"` → `[symbol]` (space is not a symbol)
  - login `an`, `"xxANxxxx"` → `[]` (logins shorter than 3 runes are skipped, same cut-off as name parts); login `binh`, `"myBINH2026"` → `[contains_login]`
  - name `"Trần Văn Bình"`, `"binhbinh"` → `[]` (no diacritics folding), `"bìnhtran!"` → `[contains_name]` (part `bình`, case-insensitive; parts under 3 runes skipped)
- [ ] **Step 2: Run** `go test ./internal/core/iam -run TestPasswordProblems` → FAIL (undefined).
- [ ] **Step 3: Implement** `passwordProblems` with `unicode.IsUpper/IsLower/IsDigit`; symbol = not letter, not digit, not space; length in runes.
- [ ] **Step 4: Write `TestLoginPolicy`** (`service_test.go`): defaults read back as `{8,false×4,5,15}`; set `{MinLength: 10, Digit: true}` then `CreateUser(..., "longenough")` fails with code `password_weak`, params rules `[digit]` (assert via `errors.As` to `*platform.Error`); `SetLoginPolicy` with MinLength 7 → `invalid_request`; non-admin → `platform.ErrForbidden`. Update `TestCreateUserRules` to the new code.
- [ ] **Step 5: Run** → FAIL; **implement** migration, queries, service, handlers; `make gen`; **run** `go test ./internal/core/iam` → PASS.
- [ ] **Step 6: i18n**: `shared.error.password_weak` (a lead sentence) plus one key per rule, `shared.error.password_rule.<rule>` (`min_length` uses `{{min}}`). Remove `shared.error.password_too_short`; update `UsersPage.tsx` `errorFields` to `password_weak: 'password'`. Add `web/src/shared/ui/form` helper `passwordErrorText(t, err): string[]` that returns one line per rule (or `[errorText(t, err)]` for other errors), with a Vitest test.
- [ ] **Step 7: Commit** `feat(iam): configurable password policy`.

### Task 3: Login lockout

**Files:**
- Create: migration `iam_login_failures`
- Modify: `queries/session.sql` (or new `queries/login.sql`), `service.go` (`Login`, `loginFailed`), `types.go`, `internal/app/auth_test.go`, `web/src/core/pages/LoginPage.tsx`, `web/src/core/i18n/{vi,en}/meta.json`

**Interfaces:**
- Consumes: `s.policy(ctx)` (Task 2).
- Produces: `ErrLoginLocked = &platform.Error{Status: 429, Code: "login_temporarily_locked"}`; unexported `func (s *Service) clearFailures(ctx, login string) error` (Task 4 unlock and Task 7 recovery use it).
- Storage: `iam.login_failures (login text PRIMARY KEY /* lower(login) */, failures int NOT NULL, last_failed_at timestamptz NOT NULL, locked_until timestamptz)`.

`Login` order: (1) if `locked_until > now()` for `lower(login)` → `ErrLoginLocked` with no hash run (also audit nothing new); (2) password check as today (unknown login keeps the `dummyHash` path); on failure, in the same tx as `iam.login_failed`, one upsert: increment `failures`, set `last_failed_at = now()`, and when the new count reaches `MaxFailures` set `locked_until = now() + LockMinutes`, reset `failures` to 0, and audit `iam.login_locked` `{login}`; also delete rows whose `last_failed_at` is older than one day and are not locked (keeps the table bounded); (3) on success delete the login's row in the login tx.

- [ ] **Step 1: Write tests** in `service_test.go`:
  - `TestLockout`: 4 wrong passwords then the right one succeeds and resets; 5 wrong → the 6th attempt with the right password returns `ErrLoginLocked`; same for `"AN"` vs `"an"`; an unknown login `"ghost"` also locks after 5; after setting `locked_until` into the past through the system pool, the right password succeeds.
  - `TestLockoutCountsAtomically`: 5 concurrent wrong attempts (goroutines) → exactly one `iam.login_locked` audit row and the next right attempt is `ErrLoginLocked`.
- [ ] **Step 2: Run** → FAIL; **implement**; **run** `go test ./internal/core/iam` → PASS. Update the expected audit string in `internal/app/auth_test.go` if it changes.
- [ ] **Step 3: LoginPage** shows `core.login.temporarily_locked` for `login_temporarily_locked` (same pattern as `invalid_credentials`, line 27).
- [ ] **Step 4: Commit** `feat(iam): temporary lock after repeated wrong passwords`.

### Task 4: Lock, unlock, rename

**Files:**
- Create: migration `iam_user_lock`
- Modify: `queries/user.sql`, `queries/session.sql`, `service.go`, `types.go`, `handler.go`, `service_test.go`

**Interfaces:**
- Consumes: `clearFailures` (Task 3).
- Produces:
  - column `iam.users.locked_at timestamptz` (NULL = active).
  - `func (s *Service) LockUser(ctx, id int64) error`, `UnlockUser(ctx, id int64) error`, `RenameUser(ctx, id int64, name string) error`; all `core.user.manage`, audited `iam.user_locked` / `iam.user_unlocked` / `iam.user_renamed {from,to}`, bump the user's `authz_version`. Lock deletes every session of the user; unlock calls `clearFailures(login)`. Name: trimmed, 1–200 chars, else `invalid_request` `fields: [name]`.
  - `User` gains `Locked bool \`json:"locked"\``.
  - `ErrAccountLocked = &platform.Error{Status: 403, Code: "account_locked"}`: returned by `Login` only after a correct password.
  - `GetSession` joins `u.locked_at IS NULL`.
  - Routes: `POST /users/{id}/lock` (`lock-user`, 204), `POST /users/{id}/unlock` (`unlock-user`, 204), `PUT /users/{id}/name` (`rename-user`, body `{name}`, 204).

- [ ] **Step 1: Write tests**: `TestLockUser` (locked user: wrong password → `ErrInvalidCredentials`, right → `ErrAccountLocked`; existing token → `Authenticate` ok=false; unlock → right password works and a prior temporary lock is cleared); `TestLockedUserSessionIsDead` (insert a session row for a locked user through the system pool, `Authenticate` → ok=false); `TestRenameUser` (blank → `invalid_request`; audited).
- [ ] **Step 2: Run** → FAIL; **implement**; `make gen`; **run** → PASS.
- [ ] **Step 3: Commit** `feat(iam): lock, unlock and rename users`.

### Task 5: Change password, admin reset, forced change

**Files:**
- Create: migration `iam_must_change_password`
- Modify: `queries/user.sql`, `queries/session.sql`, `service.go`, `types.go`, `handler.go`, `service_test.go`, `internal/app/auth_test.go`

**Interfaces:**
- Consumes: `checkPassword` (Task 2).
- Produces:
  - column `iam.users.must_change_password boolean NOT NULL DEFAULT false`.
  - `func (s *Service) ChangePassword(ctx, token, current, next string) error`: actor's own; wrong `current` → `ErrWrongPassword = {422, "wrong_password"}` (not 401, so the frontend does not end the session); `checkPassword`; clears `must_change_password`; deletes every session of the user except `tokenHash(token)`; audit `iam.password_changed`; bump authz.
  - `func (s *Service) ResetPassword(ctx, id int64, temporary string) error`: `core.user.manage`; `checkPassword`; sets `must_change_password = true`; deletes every session; audit `iam.password_reset`.
  - `Session` gains `MustChangePassword bool`; `Me` gains `MustChangePassword bool \`json:"must_change_password"\``.
  - In `authenticate`: when `MustChangePassword`, every `/api` request except `GET /api/me`, `PATCH /api/me`, `POST /api/auth/logout`, `POST /api/me/password` is answered 403 `password_change_required`.
  - Routes: `POST /me/password` (`change-my-password`, body `{current, next}`, reads the session cookie like logout, 204), `POST /users/{id}/password` (`reset-user-password`, body `{password}`, 204).

- [ ] **Step 1: Write tests**: `TestChangePassword` (wrong current → `wrong_password`; weak next → `password_weak`; success keeps the given token, drops another token, old password fails, new works); `TestResetPassword` (non-admin forbidden; sessions dropped; next login's `Me` has `MustChangePassword`); `TestForcedChangeBlocksAPI` in `internal/app/auth_test.go` (after reset, `GET /api/users` → 403 `password_change_required`; `GET /api/me` → 200; after `POST /api/me/password`, `GET /api/users` works for an admin).
- [ ] **Step 2: Run** → FAIL; **implement**; `make gen`; **run** `go test ./internal/core/iam ./internal/app -run 'Password|Forced'` → PASS.
- [ ] **Step 3: Commit** `feat(iam): change and reset passwords, forced change`.

### Task 6: Sessions

**Files:**
- Create: migration `iam_session_id_user_agent`
- Modify: `queries/session.sql`, `service.go` (`Login`), `types.go`, `handler.go`, `service_test.go`, every `Login(` caller (`grep -rn '\.Login(' internal`)

**Interfaces:**
- Produces:
  - `iam.sessions` gains `id bigint GENERATED ALWAYS AS IDENTITY UNIQUE` and `user_agent text` (truncated to 300 chars at write).
  - `Login(ctx, login, password, userAgent string) (string, error)`; the handler passes the `User-Agent` header.
  - `type SessionInfo struct { ID int64; CreatedAt, ExpiresAt time.Time; UserAgent string; Current bool }` (json snake_case).
  - `func (s *Service) MySessions(ctx, token string) ([]SessionInfo, error)` (unexpired, newest first), `EndMySession(ctx, id int64) error` (only the actor's; unknown id → `platform.ErrNotFound`; audit `iam.session_ended`), `EndMyOtherSessions(ctx, token string) error` (audit `iam.sessions_ended {count}`).
  - Routes: `GET /me/sessions` (`list-my-sessions`), `DELETE /me/sessions/{id}` (`end-my-session`, 204), `DELETE /me/sessions` (`end-my-other-sessions`, 204). The two that need the current token read the session cookie.

- [ ] **Step 1: Write `TestMySessions`**: two logins with UAs `"A"` and `"B"`; listing with token B shows two, B `Current`; another user's session id → not found; `EndMyOtherSessions(B)` leaves only B; `EndMySession(B)` logs B out.
- [ ] **Step 2: Run** → FAIL; **implement**; `make gen`; **run** → PASS.
- [ ] **Step 3: Commit** `feat(iam): list and end own sessions`.

### Task 7: Root account

**Files:**
- Create: migration `iam_root`
- Modify: `internal/core/iam/{service.go,types.go,queries/user.sql,service_test.go}`, `internal/platform/config.go`, `internal/app/app.go`, `cmd/server/main.go`, `Makefile`, `scripts/e2e-server.sh`, `compose.yml`, `web/e2e/helpers.ts`, `web/e2e/session.spec.ts`, `AGENTS.md` (env var list and the `create-admin` row of the commands table), install docs mentioning `create-admin` (`grep -rln create-admin docs AGENTS.md`, excluding `docs/archive` and ADRs)

**Interfaces:**
- Consumes: `checkPassword`, `clearFailures`, `locked_at`, `must_change_password`.
- Produces:
  - column `iam.users.is_root boolean NOT NULL DEFAULT false` with a unique partial index `WHERE is_root`; `User` gains `Root bool \`json:"root"\``.
  - `const RootLogin = "root"`.
  - `func (s *Service) EnsureRoot(ctx, password string) error`: if a root exists, nil. Else if any user has `lower(login) = 'root'`, error `"iam: a user named root exists but is not the root account"`. Else if `password == ""`, error `"ROOT_PASSWORD must be set to create the root account"`. Else `checkPassword` (as a plain Go error with the failed rules), create user `root`/`root` with `is_root`, tenant-wide `core.admin`, `must_change_password = true`; audit `iam.root_created`.
  - `func (s *Service) RecoverPassword(ctx, login, password string) error`: no permission check (CLI only); `checkPassword`; sets password, `locked_at = NULL`, `must_change_password = false`, `clearFailures`, deletes every session; audit `iam.password_recovered`.
  - `ErrRootProtected = &platform.Error{Status: 409, Code: "root_protected"}` from `LockUser`, `ResetPassword` on the root, and `RevokeRole` of the root's `core.admin` grant (the root revoking it from itself included).
  - `CreateUser` with login `root` (any case) → `ErrLoginTaken`.
  - `platform.Config.RootPassword string` from `ROOT_PASSWORD` (optional at load).
  - `app.EnsureRoot(ctx, pool, password string) error`, `app.RecoverPassword(ctx, pool, login, password string) error`; delete `app.CreateAdmin`. Keep `iam.(*Service).CreateAdmin` (tests and `coretest` use it).
  - `cmd/server`: after migrations and before command dispatch, `app.EnsureRoot(ctx, pool, cfg.RootPassword)`; the only command is `reset-password <login>` (password on stdin, prints `reset-password: <login> done`).
  - Dev/e2e: `Makefile` exports a dev `ROOT_PASSWORD` (`e2e password`); `reset-db` and `scripts/e2e-server.sh` run `echo 'e2e password' | ROOT_PASSWORD='e2e password' mmerp reset-password root` (creates root, clears forced change). `compose.yml` passes `ROOT_PASSWORD: ${ROOT_PASSWORD:-}`. `web/e2e/helpers.ts` default login `root`, name `root`.

- [ ] **Step 1: Write `TestRoot`** (`service_test.go`): `EnsureRoot("")` on an empty db errors; `EnsureRoot("root1234")` errors (contains login); `EnsureRoot("long enough")` creates it, a second call is a no-op even with another password; `Login("ROOT", …)` works and `Me.MustChangePassword`; the admin from `setup` gets `ErrRootProtected` for `LockUser(root)`, `ResetPassword(root)`, `RevokeRole(root, core.admin grant)`, and root itself for revoking its own `core.admin`; `CreateUser("Root", …)` → `ErrLoginTaken`; `RecoverPassword("root", "another long one")` clears the forced change, a temporary lock and sessions. A separate case: a pre-existing plain user `root` makes `EnsureRoot` fail.
- [ ] **Step 2: Run** → FAIL; **implement** the iam part; **run** → PASS.
- [ ] **Step 3: Wire** config, `app`, `cmd/server`, Makefile, script, compose, e2e helpers and docs; `make reset-db` then `go run ./cmd/server` starts with no error and `curl -s -c /tmp/c -XPOST localhost:8080/api/auth/login -d '{"login":"root","password":"e2e password"}' -H 'content-type: application/json'` → 204.
- [ ] **Step 4: Commit** `feat(iam): root account created at startup; reset-password replaces create-admin`.

### Task 8: Frontend, personal account

Use the `mmerp-ui` and `new-screen` skills; finish with the `docs/ui.md` checklist.

**Files:**
- Create: `web/src/core/pages/PasswordPage.tsx`, `web/src/core/pages/SessionsPage.tsx`, `web/src/core/deviceLabel.ts` (+ `deviceLabel.test.ts`)
- Modify: `web/src/core/index.ts` (export the two lazy pages, strings in `meta` like `InboxPage`), `web/src/app/router.tsx`, `web/src/app/AppLayout.tsx` (account menu: "Đổi mật khẩu", "Phiên đăng nhập" above logout), `web/src/core/pages/LoginPage.tsx`, `web/src/core/i18n/{vi,en}/meta.json`

**Interfaces:**
- Consumes: `/me` `must_change_password`; `POST /me/password`; `/me/sessions` routes; `passwordErrorText` (Task 2).
- Produces: routes `/account/password` and `/account/sessions`; `deviceLabel(ua: string): string` returning e.g. `"Chrome · Windows"` from the common browser/OS tokens, the raw UA truncated to 40 chars when nothing matches, `""` for empty.

- [ ] **Step 1: `deviceLabel` test then code** (Vitest: Chrome/Windows, Firefox/Linux, Safari/iOS, Edge/Windows (not Chrome), unknown → truncated).
- [ ] **Step 2: PasswordPage**: current, new, confirm (mismatch is a client-side field error); rule failures shown one per line under the new-password field; on success `notifySuccess` and navigate home. When `me.must_change_password`, `AppLayout` renders only this page (no nav), with a line explaining an administrator set a temporary password, and the logout action.
- [ ] **Step 3: SessionsPage**: list (device label, opened at, expires at, "this device" badge), "end" per row except current, "end all other sessions" with a confirm.
- [ ] **Step 4: LoginPage**: `account_locked` → `core.login.account_locked`.
- [ ] **Step 5: Run** `make lint` and `pnpm -C web test` → pass. **Commit** `feat(web): change password and manage own sessions`.

### Task 9: Frontend, administration

Same skills as Task 8.

**Files:**
- Create: `web/src/core/pages/LoginPolicyPage.tsx`
- Modify: `web/src/core/index.ts` (nav item `core.nav.login_policy`, path `login-policy`, permission `core.user.manage`), `web/src/core/routes.tsx`, `web/src/core/pages/UserPage.tsx`, `web/src/core/pages/UsersPage.tsx`, `web/src/core/i18n/{vi,en}/main.json`

**Interfaces:**
- Consumes: `/login-policy`, lock/unlock/name/password user routes, `User.locked`, `User.root`.

- [ ] **Step 1: LoginPolicyPage**: one form, two sections (password rules: min length, four switches; lockout: max failures, lock minutes), field bounds as Task 2.
- [ ] **Step 2: UserPage**: badges "Tài khoản gốc" / "Đã khoá"; name editable (`InlineEditField`, as email); actions menu: reset password (modal with the temporary password, rule errors per line), lock (confirm) / unlock. For `root`, hide lock and reset, and hide revoke on its `core.admin` grant.
- [ ] **Step 3: UsersPage**: the same two badges in the list.
- [ ] **Step 4: Run** `make lint`, `pnpm -C web test` → pass. **Commit** `feat(web): login policy, lock and reset users`.

### Task 10: End-to-end

**Files:**
- Create: `web/e2e/accounts.spec.ts`

- [ ] **Step 1: Write specs** (Vietnamese UI, as the other specs): (a) admin creates user, resets their password; the user signs in, lands on the forced change page, other menus unreachable, changes it, reaches home; (b) admin locks the user; the user's open page gets signed out on the next request and sign-in shows the locked message; unlock restores it; (c) the root's user page shows no lock/reset actions; (d) user sees two sessions after signing in from two contexts and ends the other one, which is then signed out.
- [ ] **Step 2: Run** `make e2e` → all pass (including the existing specs on `root`).
- [ ] **Step 3: Commit** `test(web): e2e for account lifecycle`.

### Finish

- [ ] `make gen` (no diff), `make lint`, `make test`, `make e2e` all pass.
- [ ] Run the `code-review` skill on the branch; fix findings.
- [ ] Ask the user before pushing and opening the PR (house style: title `feat(iam): accounts that work for real`, plain-text body listing tasks, behaviour changes (`create-admin` removed, `ROOT_PASSWORD`, `password_too_short` → `password_weak`, `Login` signature) and migrations).
