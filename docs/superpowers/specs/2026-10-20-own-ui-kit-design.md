# Own UI kit on Base UI: `@mmerp/ui` replaces Mantine

Date: 2026-10-20. Status: **exploration draft, nothing decided.** The owner is surveying the idea; this document and the proposed [ADR-0037](../../adr/0037-own-ui-kit-on-base-ui.md) exist to be read, not built from.

## Why

The owner wants fewer third-party libraries in the frontend. Today `web/` depends on four Mantine packages (`@mantine/core`, `dates`, `hooks`, `notifications`), imported by 123 files: about 60 in `shared/` and `app/`, about 45 in areas. Mantine also leaks into the design language itself: `theme.ts` is a Mantine `createTheme`, CSS modules read `--mantine-*` variables (31 files), and screens use Mantine style props (`c="dimmed"` 98 times, `fw={…}` 52, `p=` 21, `miw=` 19).

mmerp will provide its own UI, as a package in the same repo, built on the unstyled, accessible primitives of [`@base-ui/react`](https://base-ui.com) (1.9). The model is Twenty's [`twenty-ui`](https://github.com/twentyhq/twenty/tree/main/packages/twenty-ui): a workspace package, with design tokens turned into CSS variables, and `primitives` that wrap Base UI with the project's own styling. Twenty's other layer, presets composed from primitives, already exists in mmerp as `shared/ui`.

Mantine is replaced, not wrapped. A thin wrapper over Mantine would keep the dependency, which defeats the purpose.

## Goals

1. **`@mmerp/ui`**, a standalone package at `packages/ui` (repo root, beside `web/`), holding:
   - **tokens**: the values in today's `theme.ts` and `cssVariablesResolver` (colour ramps, type scale, spacing, radius, shadows, breakpoints, control heights, layout widths), written once in TypeScript and emitted as CSS custom properties `--mmerp-*`;
   - **primitives**: every Mantine component mmerp uses today, rebuilt on Base UI where Base UI has a matching part, and on plain HTML and CSS otherwise (see the inventory below);
   - **hooks**: the five `@mantine/hooks` mmerp uses (`useDisclosure`, `useMediaQuery`, `useDebouncedValue`, `useLocalStorage`, `useElementSize`).
2. **`@mmerp/ui` knows nothing of mmerp.** It has no i18n, router, API client, React Hook Form or domain term, and no user-facing text. Labels, aria labels and messages come in as props. Everything that knows mmerp stays in `shared/ui`. Amended 2026-10-21: see 2026-10-21-ui-kit-layout-blocks-design.md.
3. **`shared/ui` keeps its role and its public API.** `DataTable`, `ListView`, the form fields, page templates, `confirm`, `notify`, `ModalFooter` and the rest are rebuilt on `@mmerp/ui`. Their props do not change unless a prop only passed a Mantine type through. Amended 2026-10-21: see 2026-10-21-ui-kit-layout-blocks-design.md.
4. **Areas move from `@mantine/core` to `@mmerp/ui`**, under the same allowlist as today (`Anchor`, `Button`, `Divider`, `Group`, `List`, `Loader`, `Skeleton`, `Stack`, `Text`, `Tooltip`). Everything else still comes through `shared/ui`.
5. **The package owns its design.** It starts from today's look and may then add, change or remove tokens and styling as its direction evolves; it does not follow `theme.ts`. The starting point is the values in `ui.md` and the `mmerp-ui` skill: 30px controls (40px in modals and form sections), 14/20 text, the 4px spacing scale, the radius per role, overlays as the only things with a shadow, the focus ring, the 44px touch target below 1024px, motion of 200ms at most that honours `prefers-reduced-motion`.
6. **All four `@mantine/*` packages are removed from `package.json`**, together with the Mantine MCP server, the vendored `mantine-*` skills and every Mantine mention in the docs.
7. **Primitives are documented in Storybook; `/dev/ui` keeps mmerp's own layers.** In phase 2, layer 3, today "Mantine, used directly", becomes a short page listing the area allowlist with links to Storybook.

## Non-goals

- No dark mode and no components beyond the parity inventory below, unless the package's direction calls for them.
- No publishing. `@mmerp/ui` is `private: true`, has no build step and no `dist/`, and its `exports` point at TypeScript source.
- No Storybook addons beyond a11y and Vitest, no Chromatic or other hosted service. Storybook is a dev tool of `packages/ui` only and never ships.
- No Tailwind, Sass or CSS-in-JS. Styling is CSS modules plus CSS custom properties, as `shared/ui` already does.
- No generic style props (`m`, `p`, `c`, `fw`, `miw`…) on every component. Primitives take a small, named set of props, listed below.
- No change to the backend, the API, or the area/shared boundaries.

## Anti-goals (a failure even if it technically works)

- `@mantine/*` still in `package.json` or the lockfile at the end.
- `@base-ui/react` imported anywhere but `packages/ui`. Then the next swap would again touch 120 files.
- `@mmerp/ui` importing from `web/src`, using i18n, or holding Vietnamese or English text.
- A colour, size or spacing literal outside the token file or a primitive's own CSS module.
- Visible regressions on real screens: a 36px button, a menu without keyboard support, a modal that does not trap focus, a date field that no longer accepts `dd/mm/yyyy` typed by hand.
- A second, parallel set of components: `shared/ui` re-exporting `@mmerp/ui` under new names, or primitives duplicating a `shared/ui` component. Amended 2026-10-21: see 2026-10-21-ui-kit-layout-blocks-design.md.

## Constraints

- Frontend stack rules from `techstack.md` stay: React 19, Vite, React Hook Form, TanStack Query, Tabler icons, dayjs. Base UI is the one new runtime dependency of the package; class names are joined by a one-line helper, not `clsx`, and date arithmetic uses the native `Date` (UTC), not dayjs.
- `make lint`, `make test`, `make e2e` and `make image` stay green after every ticket, not only at the end.
- **Phase 1 must not touch `web/`, the Dockerfile or the Makefile**, and in CI only adds one workflow (see Phases). Phase 2 wires them: the Docker web stage then also copies `packages/ui` before `pnpm install`.
- A hard-to-reverse decision needs its ADR before code: ADR-0037 is part of this change and supersedes the "Mantine" part of ADR-0008.
- Tests: Vitest covers the hooks and the date parsing. Playwright e2e is the end-to-end proof that real screens still work. Primitives get no snapshot tests.
- **E2E specs keep their steps and their assertions on outcomes.** A ticket may change a selector only when the DOM shape changes but the user does the same thing. Behaviour may not change to fit Base UI: a select you can type into today (searchable `Select`, `OrgUnitField`, `RemoteSelectField`) stays typeable, so it is built on Base UI `combobox`, not `select`.

## Architecture

```
packages/ui/                       @mmerp/ui (repo root)
├── package.json                   private; deps: @base-ui/react (react, react-dom as peers);
│                                  own pnpm-lock.yaml, tsconfig, Vitest, ESLint config
├── .storybook/                    Storybook config (React + Vite), addons: a11y, vitest
├── src/**/*.stories.tsx           one story file per primitive, one story per variant and state
└── src/
│       ├── tokens/
│       │   ├── tokens.ts          the package's own values (ramps, scale, radius…)
│       │   └── tokens.css         generated from tokens.ts; checked in
│       ├── styles/                reset.css, global.css (control heights, touch target, focus ring)
│       ├── primitives/
│       │   ├── layout/            Box, Stack, Group, Grid, SimpleGrid, Center, Divider, ScrollArea, AppShell
│       │   ├── typography/        Text, Title, Anchor, Code, List
│       │   ├── input/             Button, ActionIcon, UnstyledButton, TextInput, PasswordInput,
│       │   │                      Textarea, NumberInput, Checkbox, Select, MultiSelect, Combobox,
│       │   │                      SegmentedControl, FileInput, Field (label, description, error)
│       │   ├── overlay/           Dialog (modal and drawer), Popover, Menu, Tooltip
│       │   ├── navigation/        Tabs, NavLink, Breadcrumbs, Pagination, Burger
│       │   ├── feedback/          Toast, Alert, Badge, Skeleton, Loader, Indicator
│       │   ├── data-display/      Avatar, ThemeIcon, Table, Timeline, Paper
│       │   └── dates/             Calendar, DateInput, MonthPicker, MonthPickerInput
│       ├── hooks/                 useDisclosure, useMediaQuery, useDebouncedValue, useLocalStorage, useElementSize
│       └── index.ts               the public surface; one subpath per folder is optional
web/                               phase 2 only: depends on "@mmerp/ui": "link:../packages/ui"
└── src/
    ├── shared/ui/                 mmerp components, now built on @mmerp/ui
    ├── hrm/, sales/, core/        import @mmerp/ui (allowlist) and shared/ui
    └── app/                       mounts the stylesheets and the toast viewport; no provider beyond that
```

Dependencies only point down: `areas → shared → @mmerp/ui → @base-ui/react`. `app` imports `@mmerp/ui` for the stylesheets and the toast viewport.

### Tokens

- `tokens.ts` exports plain objects: `color` (the seven ramps of today, 0–9, plus the semantic roles: text, dimmed, placeholder, border, input border, body, surface, sunken surface, table header, error, the light tints), `fontSize`, `lineHeight`, `space`, `radius`, `shadow`, `breakpoint`, `controlHeight`, `layoutWidth`, `icon`, `filterWidth`.
- A small script (`scripts/gen-tokens.mjs`, run by `make gen`) writes `tokens.css` with one `:root` block of `--mmerp-*` properties. Names follow roles: `--mmerp-color-primary-6`, `--mmerp-color-text-dimmed`, `--mmerp-space-md`, `--mmerp-radius-md`, `--mmerp-font-size-sm`, `--mmerp-control-height`. The `--mmerp-*` variables that exist today keep their names.
- Breakpoints cannot be CSS variables inside `@media`. The CSS modules write the values (`64em`, `80em`) and `tokens.ts` keeps the same numbers for `useMediaQuery`. A comment ties the two together.
- `scripts/check-colors.sh` keeps its rule. Its exemption moves from `shared/ui` to `packages/ui/src/tokens` and the primitives' CSS modules. `shared/ui` CSS modules then use variables only.

### Primitives

- **Wrapping rule.** A primitive with a Base UI counterpart is a styled composition of the Base UI parts: `className` from its CSS module, states read from Base UI's `data-*` attributes (`[data-open]`, `[data-highlighted]`, `[data-disabled]`, `[data-checked]`). Compound primitives keep Base UI's part names (`Menu.Root`, `Menu.Trigger`, `Menu.Item`…), as Twenty does, unless mmerp's call sites read better with Mantine's shape. That choice is per component and up to the builder.
- **Props.** Each primitive takes the props of the element it renders, plus a small named set:
  - `Text`: `size` (`xs`–`xl`), `tone` (`default`, `dimmed`, `danger`, `success`, …), `weight` (`400`, `500`, `600`, `650`), `align`, `truncate`, `component`/`render`;
  - `Stack`/`Group`: `gap`, `align`, `justify`, `wrap`, `grow`;
  - `Button`/`ActionIcon`: `variant` (the variants mmerp uses today: filled, light, subtle, default), `color`, `size` (`xs`, `sm` default), `leftSection`, `rightSection`, `loading`, `fullWidth`;
  - widths that callers pass today (`miw`, `maw`, `w`) become either a prop with a token value on the few primitives that need one, or a class in the calling `shared/ui` CSS module. Areas pass no widths (they cannot today either).
- **Polymorphism.** Use Base UI's `render` prop (and its `useRender` for primitives without a Base UI part) wherever Mantine's `component` prop is used today (for example `Anchor` rendered as a router `Link`).
- **Accessibility and behaviour come from Base UI**: focus trapping and return, `Escape`, typeahead, roving focus, aria wiring, scroll lock, portal and positioning. Primitives without a Base UI counterpart (`Stack`, `Text`, `Table`, `Timeline`…) are presentational and need none of that.

### Parity inventory: Mantine → `@mmerp/ui`

Usage counts are from `web/src` on 2026-10-20.

| Mantine (uses) | `@mmerp/ui` built on |
| --- | --- |
| Button (65), ActionIcon (8), UnstyledButton (6) | Base UI `button` |
| Text (58), Title (5), Anchor (23), Code (2), List (4) | HTML + CSS; `useRender` for `Anchor` |
| Group (52), Stack (50), Box (9), SimpleGrid (7), Grid (2), Flex (2), Center (2), Divider (2) | HTML + CSS (`Divider` on Base UI `separator`); `Flex` folds into `Group` |
| Paper (11), ThemeIcon (8), Avatar (1), Badge (5), Indicator (1) | HTML + CSS; `Avatar` on Base UI `avatar` |
| Tooltip (13) | Base UI `tooltip` |
| Menu (8), Popover (3) | Base UI `menu`, `popover` |
| Modal (4), Drawer (2) | Base UI `dialog`, `drawer` |
| ScrollArea (6) | Base UI `scroll-area` |
| TextInput (7), PasswordInput (3), Textarea (2), Input, InputWrapper | Base UI `input`, `field` |
| NumberInput (1) | Base UI `number-field` |
| Checkbox (3) | Base UI `checkbox` |
| Select (4), MultiSelect (1), Combobox + useCombobox (3) | Base UI `select`, `combobox` |
| SegmentedControl (4) | Base UI `toggle-group` |
| FileInput (1), FileButton (1) | HTML `<input type="file">` behind a `Button` |
| Tabs (4) | Base UI `tabs` |
| NavLink (4), Breadcrumbs (1), Pagination (1), Burger (2), TableOfContents (1) | HTML + CSS |
| AppShell (2) | CSS grid layout, header and navbar slots, collapsing below `64em` |
| Table (2) | HTML table + CSS (`DataTable`, `LinesTable` already own the behaviour) |
| Skeleton (4), Loader (4), Alert (1), Timeline (1), VisuallyHidden (3) | HTML + CSS |
| Notification, `@mantine/notifications` (2 files) | Base UI `toast` (`Toast.Provider`, viewport, `useToastManager`) behind `shared/ui/notify` |
| `@mantine/dates`: DateInput, MonthPickerInput, MonthPicker, DatesProvider (3 files) | own `Calendar` and month grid on Base UI `popover`, dayjs for dates (see below) |
| `@mantine/hooks` (12 files) | own hooks, a few lines each (`matchMedia`, `ResizeObserver`, `localStorage`, `setTimeout`) |
| `createTheme`, `MantineProvider`, `CSSVariablesResolver`, `rem`, `useMantineTheme` | `tokens.ts`, `tokens.css`; no provider |

Anything in this table that turns out unused once its call site is rebuilt on `shared/ui` is dropped, not built.

### Dates

Base UI has no date picker. `@mmerp/ui` gets:

- `Calendar`: a month grid of days with keyboard navigation (arrows, PageUp/PageDown for months, Home/End), `aria-selected`, today marked, min/max. A month variant shows the 12 months of a year for `MonthPicker`.
- `DateInput`: a text input that accepts `dd/mm/yyyy` typed by hand and opens the calendar in a Base UI `popover`. Parsing stays in `shared/ui/form` (`parseDate`, already tested), passed in as a prop, so the package carries no locale rule.
- `MonthPickerInput`: the same for a month.
- Locale (weekday and month names, first day of week) comes from props that `shared/ui` fills from i18n and dayjs.

The native `<input type="date">` was rejected because it shows the browser's locale format, not always `dd/mm/yyyy`, and cannot be styled to the 30px control.

### Toasts

`Toasts.tsx` mounts the Base UI toast viewport once in `app`. `shared/ui/notify` keeps its current signature and calls the toast manager. Position, durations and the Undo action behave as `ui.md` and the `mmerp-ui` skill describe today.

### Enforcement

| Rule | Enforced by |
| --- | --- |
| Only `packages/ui` imports `@base-ui/react` | ESLint `no-restricted-imports` on `web/src` |
| `@mmerp/ui` does not import `web/src`, i18n, router, RHF, TanStack Query or the API client | dependency-cruiser rule on `packages/ui` (plus ESLint `no-restricted-imports` for the packages) |
| No going back to Mantine for what is already migrated | A ratchet: each ticket adds the names it migrated to an `importNames` ban on `@mantine/core` (and bans `@mantine/hooks`/`notifications`/`dates` once their ticket is done). `packages/ui` bans `@mantine/*` from the start. The rule goes away in ticket 7, once the packages are gone and imports cannot resolve |
| Areas import only the allowlisted primitives from `@mmerp/ui` | today's `allowImportNames` rule, pointed at `@mmerp/ui` |
| No literal text in `@mmerp/ui` | ESLint `i18next/no-literal-string` also runs on `packages/ui` |
| `tokens.css` matches `tokens.ts` | `make gen` regenerates; CI fails on diff, like the other generated files |
| No colour literals outside tokens and primitive CSS | `scripts/check-colors.sh`, with its exemption moved |

The planted fixtures in `web/lint-fixtures` and their `expected.json` get a case for each new rule.

## Phases

### Phase 1: build `@mmerp/ui` in isolation (owner's choice, 2026-10-20)

Phase 1 built on 2026-10-20; see packages/ui.

The package is finished before anything in the app changes. Ongoing work in `web/` is not affected.

- Everything lives under `packages/ui`: its own `package.json` and lockfile, `tsconfig`, Vitest, ESLint config (including `i18next/no-literal-string` and a ban on `@mantine/*`), and a colour-literal check limited to tokens and primitive CSS modules.
- **Nothing outside `packages/ui` changes, except one added CI workflow.** `web/` (its `package.json`, lockfile, `eslint .`, `tsc`, Vitest, dependency-cruiser), the Dockerfile, the Makefile and the existing CI jobs stay as they are. `make lint/test/e2e/image` never see the package.
- **CI workflow `.github/workflows/ui.yml`** (its own file, because GitHub filters paths per workflow, not per job; `ci.yml` stays untouched), run only when a push touches `packages/ui/**` or the workflow itself, working in `packages/ui`: `pnpm install --frozen-lockfile`; `tsc`; ESLint; the colour-literal check; regenerate `tokens.css` and fail on diff; Vitest unit tests; Storybook story tests in Chromium (a11y and `play` functions); `storybook build`. (owner, 2026-10-20)
- **Primitives are viewed and tested in Storybook** (`pnpm storybook` in `packages/ui`), not on `/dev/ui`. (owner, 2026-10-20)
  - One story file per primitive, one story per variant and state, plus a page showing the tokens.
  - `@storybook/addon-a11y` checks every story; a violation fails the story's test.
  - `@storybook/addon-vitest` runs stories as browser tests (Vitest browser mode with Playwright). Interactive primitives (Menu, Select, Combobox, Dialog, Popover, Tabs, Tooltip, Toast, the date inputs) get `play` functions that drive them by keyboard and assert focus, open/close and the chosen value.
  - Stories are not part of the app and have no i18n: sample labels in `*.stories.tsx` are exempt from the literal-string rule.
- **All tokens are seeded once** from `web/src/shared/ui/theme.ts` (and the `--mmerp-*` values of `cssVariablesResolver` and `global.css`) when the package is created: colours, type scale, line heights, spacing, radius, shadows, breakpoints, control heights, layout widths, icon sizes. From then on `tokens.ts` belongs to the package: it adds, changes or removes tokens as its direction changes, and nothing keeps it in step with `theme.ts`. (owner, 2026-10-20)
- Scope is the whole parity inventory below, including dates, toasts and hooks.
- Phase 1 is done when every primitive in the inventory has its stories, the package's lint, unit tests and story tests (a11y included) pass, and the owner has looked through Storybook.
- Known limit: Storybook is not proof on real screens. Issues such as typeable selects or the 40px fields in modals only show in phase 2.

### Phase 2: migrate `web/` (later, separate decision)

One spec branch, one PR, one commit per ticket. The GitHub issues (the spec and one per ticket below) are created when the plan is written. Mantine and `@mmerp/ui` coexist until the last ticket, and every ticket leaves the app working. Primitives built in phase 1 are adjusted where real call sites need it.

1. **Wiring, tokens, ADR.** `web/` depends on `@mmerp/ui` (`link:`), and `make gen`, the Makefile, CI and the Dockerfile are wired. The app adopts the package's tokens: Mantine's theme reads its values from `tokens.ts`, so there is one source of values, and screens take on whatever the package's look has become. `--mantine-*` uses in `shared/ui` CSS modules become `--mmerp-*`. ADR-0037 is accepted.
2. **Layout, typography, buttons and hooks.** `Box`, `Stack`, `Group`, `Grid`, `SimpleGrid`, `Center`, `Divider`, `Text`, `Title`, `Anchor`, `Code`, `List`, `Button`, `ActionIcon`, `UnstyledButton`, `Paper`, `Badge`, `ThemeIcon`, `Loader`, `Skeleton`, `VisuallyHidden`, and the five hooks. Every import of these switches, in areas too, and the area allowlist moves to `@mmerp/ui`. This is the largest ticket by file count and the most mechanical.
3. **Form inputs.** `Field`, `TextInput`, `PasswordInput`, `Textarea`, `NumberInput`, `Checkbox`, `Select`, `MultiSelect`, `Combobox`, `SegmentedControl`, `FileInput`. `shared/ui/form` is rebuilt on them, including `RemoteSelectField` and `SearchInput`.
4. **Overlays.** `Dialog` (modal and drawer), `Popover`, `Menu`, `Tooltip`, `ScrollArea`. `FormModal`, `ModalFooter`, `confirm`, `MoreActions`, `MenuAction`, `SidePanel`, `FilterChip` are rebuilt on them.
5. **Navigation, shell, feedback, data display.** `Tabs`, `NavLink`, `Breadcrumbs`, `Pagination`, `Burger`, `AppShell`, `Table`, `Timeline`, `Avatar`, `Indicator`, `Alert`, `Toast`. `AppLayout`, `notify` and `Toasts` switch over.
6. **Dates.** `Calendar`, `DateInput`, `MonthPicker`, `MonthPickerInput`. `fields.tsx` and `MonthFilter` switch over, and `DatesProvider` goes.
7. **Remove Mantine.** Uninstall the four packages. Delete `theme.ts`'s Mantine half, `MantineProvider` and the Mantine stylesheet imports in `bootstrap.tsx`, the Mantine MCP entry in `.mcp.json`, the `mantine-*` skills in `.claude/skills` and `skills-lock.json`. Update `/dev/ui` layer 3, `techstack.md`, `ui.md`, `frontend.md`, `architecture.md` if it names Mantine, `CLAUDE.md`, and the `mmerp-ui` skill (`SKILL.md`, `references/tokens.md`, `references/components.md`, and every reference that cites a Mantine prop or variable).

Each ticket ends with the `mmerp-ui` checklist on the screens it touched and a manual pass through `/dev/ui`.

## Success

- `grep -r mantine web/ .claude .mcp.json *.md` finds nothing but the ADRs that record history.
- `pnpm why @mantine/core` finds nothing. `@base-ui/react` is the only new runtime dependency.
- `make lint`, `make test`, `make e2e`, `make image` and `make smoke` pass.
- On `/dev/ui` and the seeded demo company (`make seed`), the HRM and Sales screens look like the package's stories, and the owner accepts the differences from today's screens after seeing them. Keyboard use works in menus, selects, comboboxes, dialogs, tabs and the date fields.
- The production JS bundle is no larger than today's. The builder measures before ticket 1 and after ticket 7.

## Direction so far (not decided)

- Base UI is the primitive layer. Mantine is removed entirely, not wrapped. (owner's stated direction)
- A separate package in the same repo, modelled on `twenty-ui`. (owner's stated direction)
- No Tailwind. CSS modules plus CSS custom properties. (owner's stated direction, on the recommendation that Tailwind would add a dependency and erode the rule that areas carry no styling)
- `@mmerp/ui` holds tokens and primitives only and knows nothing of mmerp. `shared/ui` is the presets layer. (my call, played back to the owner) Amended 2026-10-21: see 2026-10-21-ui-kit-layout-blocks-design.md.
- Build the package first, in isolation, without affecting ongoing work; view and test it in Storybook (addons a11y and Vitest only) instead of `/dev/ui`. (owner's stated direction, 2026-10-20)
- In phase 2, `/dev/ui` layer 3 links to the primitives' Storybook instead of repeating them; Storybook covers primitives, `/dev/ui` mmerp components, patterns and templates. (my call)
- The package sits at `packages/ui` in the repo root, named `@mmerp/ui`, consumed from source with no build step; in phase 2 `web/` links to it. (my call: inside `web/`, `eslint .` would already pick it up, which breaks the no-impact rule)
- Only `@mmerp/ui` imports Base UI. Areas keep today's allowlist, now against `@mmerp/ui`. (my call)
- Own date input and calendar instead of the native date input. (my call; reason above)
- Incremental migration with both libraries coexisting until the last ticket. (my call: a big-bang swap of 123 files cannot keep `make e2e` green along the way)
- The Mantine ban is a per-ticket ratchet by component name, not a blanket ban. (my call: a blanket ban would fail lint on every file not yet migrated)
- E2E specs may change selectors, never steps or outcome assertions; typeable selects stay typeable. (my call)
- The 40px fields inside modals and form sections, today set by Mantine theme overrides and `global.css`, are carried across in whichever ticket moves the field or the modal. (my call)

## Left to the builder

- Compound API shape per primitive (Base UI part names or a Mantine-like shape), as long as call sites in `shared/ui` stay readable.
- Exact token variable names beyond the pattern above, and whether `index.ts` also exposes subpaths.
- Splitting a ticket further if it grows too large to review.
- Which Mantine components turn out to be unnecessary once their `shared/ui` user is rebuilt.

## Open questions

- **Visual baseline.** Should before/after screenshots of `/dev/ui` and the seeded HRM and Sales screens be captured before any code changes, so the PR can show them side by side? They cannot be recreated afterwards. The alternative is the owner looking at the finished branch.
- **Phase 2, and when.** The owner chose to build the package first; whether and when to migrate `web/` is not decided.
