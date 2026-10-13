# Design direction: moving the visual language towards Jira / ADS

Status: **decided on 2026-10-12** (every recommendation accepted). Phase 2 has started: the token layer (`theme.ts`, `global.css`), `StatusBadge` and the `ui.md` values match these decisions, and the `mmerp-ui` skill describes the result. Still to do: see the end of section 5.

Superseded on 2026-10-13 (`ui.md` and the `mmerp-ui` skill are authoritative): C7 and the `compact-xs` mention, since every button is the one 30px size; C12, since a table's header row sits on its own tone (`--mmerp-table-header`, `#F0F1F2`) at weight 600; and "no shadow outside overlays", since a pinned page-form footer casts an upward shadow (`--mmerp-shadow-footer`).

Reference: the public Atlassian Design System (ADS) docs at atlassian.design (foundations: spacing, radius, typography, elevation; components: button, lozenge) and the values published in the `@atlaskit/tokens` npm package, version 20.4.1 (Apache-2.0), read on 2026-10-12. Only principles and design values (scales, ratios, structure) are taken. No logo, icon asset, illustration or proprietary font (Atlassian Sans, Atlassian Mono, Charlie) is copied. Values marked *(unverified)* come from memory of the product, not from the docs or package, and must be checked before phase 2 relies on them.

## 1. Audit of the current design

### Stack

| Area | Current |
| --- | --- |
| Framework | React 19 + TypeScript (strict), Vite |
| Component library | Mantine 9 (`core`, `dates`, `notifications`), wrapped in `web/src/shared/ui` |
| Styling | Mantine theme + Mantine style props (`c`, `gap`, `fw`…) + CSS modules, **only** in `shared/ui` and `shared/document` (13 files, 388 lines) |
| Forms | React Hook Form, fields in `shared/ui/form` |
| Font | Inter Variable, self-hosted (`@fontsource-variable/inter`) |
| Icons | Tabler Icons (`@tabler/icons-react`), 79 distinct icons in use, stroke 1.75 |

### Where tokens live

- `web/src/shared/ui/theme.ts`: Mantine `createTheme` (5 colour ramps, font sizes, spacing, radius, headings, per-component overrides) and `cssVariablesResolver` (semantic colours as CSS variables: `--mantine-color-text`, `--mantine-color-dimmed`, `--mantine-color-default-border`, `--mmerp-color-subtle`, `--mmerp-color-surface`, …). `dark: {}` is empty.
- `web/src/shared/ui/global.css`: 44px touch targets below 1024px, tint for the chosen dropdown option.
- `ui.md`: the written rules; `theme.ts` is named as the source of truth for values.

### Values actually in use

| Foundation | Values |
| --- | --- |
| Primary | Blue ramp, filled shade `#1E40AF` (Tailwind blue-800) |
| Neutrals | Tailwind Slate: text `#0F172A`, secondary `#475569`, placeholder `#94A3B8`, border `#E2E8F0`, subtle `#F8FAFC`, surface/body `#FFFFFF` |
| Semantic | success Tailwind green (filled `#16A34A`), warning amber (`#D97706`), danger red (`#DC2626`), info sky (`#0284C7`). No *discovery* colour |
| Neutral ramp for UI | Mantine's built-in `gray` (`color="gray"`, 6 uses), which is not Slate |
| Accent | Avatars use Mantine's built-in `blue, teal, violet, orange, pink, cyan, grape, indigo`, outside the theme |
| Font sizes | 12 / 14 / 14 / 16 / 20 (`xs…xl`; `sm` and `md` are both 14), h1 22, h2 16, h3 14, field label 13 |
| Line height | 1.5 everywhere (14px text → 21px line) |
| Weights | 400, 500 (labels, buttons, tabs), 600 (headings, totals) |
| Spacing | 4 / 8 / 12 / 16 / 24 / 32 (`xxs…xl`) |
| Radius | 6px for everything except avatars |
| Shadow | None outside Mantine's own overlays (modal, menu, popover) |
| Heights | Header 48, navigation 240 / 56 collapsed, table row 40, table header 36, CellGrid cell 32, buttons `sm` 36 / `xs` 30, inputs `sm` 36 |
| Motion | Mantine defaults, ≤ 200ms, `respectReducedMotion` |

### Consistency

Consistency is **high**, because tools enforce it: `make lint` rejects hex/`rgb(`/`hsl(` outside `shared/ui`, ESLint bans inline `style` and convention-bearing Mantine components in areas, dependency-cruiser bans CSS in areas. The only colour literals are in `theme.ts`.

What slips through (all inside `shared/ui`, `shared/document` or `app`, plus two in areas):

- Off-scale sizes: field label 13px; `gap={2}` (×4) and `gap={4}` (`core/pages/InboxPage.tsx`); `31px`, `34px`, `36px`, `2.25rem` in CSS modules; `Tile` skeleton 28×64; `Indicator size={16}`, `ThemeIcon size={28}`/`{48}`.
- Two neutral systems: Slate tokens for text and borders, Mantine `gray` for neutral badges and header icons.
- Avatar accent colours from Mantine defaults, not theme tokens.
- `fontSizes.sm === fontSizes.md`, so components that default to `md` look the same as `sm`.
- Usage of style props is otherwise disciplined: `c="dimmed"` (40), `gap="xs"` (54), `size="sm"` (54), `variant="default"` (36), `color="danger"` (23).

### Shared components and screen patterns

- Page templates (`shared/ui/page`): `Page`, `ListPage`, `DocumentPage`, `RecordPage`, `InboxPage`, `FormModal`, `AuthPage`, `TileGrid`.
- Display: `DataTable` (40px rows, subtle header, sticky id column, card list below 1024px, reconciliation tables with grouped headers and sticky totals), `FieldList`, `StatusBadge` (badge + dot), `DocumentStatus`, `CellGrid` (timesheet), `StatTile`/`LinkTile`, `OrgUnitName`, `PersonName`/`PersonAvatar`.
- Forms: `Form`, `FormSection` (unframed, ruled), `InputRow`, `FormActions` (sticky footer with dirty state), fields (`TextField`, `DateField`, `MoneyField`, `SelectField`, `SensitiveField`, `OrgUnitField`, …), filters (`SearchInput`, `MonthFilter`).
- Feedback: `useConfirm`, `notifySuccess`/`notifyError` (bottom-right), states (`EmptyState` dashed frame + icon, `ErrorState`, skeletons, 403/404, `ReadOnlyBanner`).
- Documents (`shared/document`): `DocumentActions`, `ApprovalPanel`, `DocumentHistory`, `AttachmentPanel`, `DiscussionPanel`.
- App shell (`app/AppLayout.tsx`): 48px header (home logo, area title, notifications/inbox/jobs icons, avatar menu), 240px left navigation per area, home page of tiles.
- 69 screen files across `core`, `hrm`, `sales`; every one built from the above.

## 2. ADS principles that matter here

### Colour

- Tokens are named by role, not hue: `color.text`, `color.text.subtle`, `color.text.subtlest`, `color.background.neutral`, `color.background.selected`, `color.border`, `color.border.focused`, `color.icon.subtle`…, each with `hovered`/`pressed` states.
- Semantic roles: **success, warning, danger, information, discovery** (discovery = new/beta/onboarding, purple), plus **brand**. Each has `subtlest → subtler → subtle → (default) → bold` background steps and a text colour.
- Accent colours (lime, red, orange, yellow, green, teal, blue, purple, magenta, gray) are for user-chosen colour without meaning: avatars, labels, tags.
- Borders and neutral backgrounds are translucent (alpha over the surface), so they work on any surface and in dark mode.

Selected values, light theme (refresh / legacy):

| Token | Refresh | Legacy | Current mmerp |
| --- | --- | --- | --- |
| `text` | `#292A2E` | `#172B4D` | `#0F172A` |
| `text.subtle` | `#505258` | `#42526E` | `#475569` |
| `text.subtlest` | `#6B6E76` | `#7A869A` | (none) |
| `text.disabled` | `#080F214A` | `#A5ADBA` | `#94A3B8` |
| `background.brand.bold` / `link` | `#1868DB` | `#0052CC` | `#1E40AF` |
| `background.selected` | `#E9F2FE` | `#DEEBFF` | primary-light (`#EFF6FF`-ish) |
| `border` | `#0B120E24` | `#091E4221` | `#E2E8F0` |
| `border.input` | `#8C8F97` | — | `#E2E8F0` |
| `border.focused` | `#4688EC` | `#2684FF` | primary |
| `background.neutral` | `#0515240F` | `#DFE1E6` | Mantine gray |
| `surface.sunken` | `#F8F8F8` | `#F4F5F7` | `#F8FAFC` (`subtle`) |
| `background.success.bold` / `text.success` | `#5B7F24` / `#4C6B1F` | `#00875A` / `#006644` | `#16A34A` |
| `background.warning.bold` / `text.warning` | `#FBC828` / `#9E4C00` | `#FFAB00` / `#974F0C` | `#D97706` |
| `background.danger.bold` / `text.danger` | `#C9372C` / `#AE2E24` | `#DE350B` | `#DC2626` |
| `background.information.bold` | `#1868DB` | `#0052CC` | `#0284C7` |
| `background.discovery.bold` | `#964AC0` | `#5243AA` | (none) |

### Typography

Body `font.body` 14/20, `font.body.small` 12/16, `font.body.large` 16/24. Headings, weight bold (653 in the refresh): `xxlarge` 32/36, `xlarge` 28/32, `large` 24/28, `medium` 20/24, `small` 16/20, `xsmall` 14/20, `xxsmall` 12/16. Weights: 400, 500, 600, 653. Sizes are in rem. The refresh uses the proprietary Atlassian Sans with a system-font fallback (`ui-sans-serif, -apple-system, "Segoe UI", Ubuntu, "Helvetica Neue", sans-serif`). Line heights are fixed 4px multiples (20 for 14px), tighter than our 1.5.

### Spacing

8px base, tokens named as a percentage of it: `space.025` 2, `.050` 4, `.075` 6, `.100` 8, `.150` 12, `.200` 16, `.250` 20, `.300` 24, `.400` 32, `.500` 40, `.600` 48, `.800` 64, `.1000` 80.

### Elevation

Four levels, surface and shadow always paired: **sunken** (`#F8F8F8`, a well for grouped content, e.g. board columns), **default** (flat, white, borders instead of shadows), **raised** (`0 1px 1px #1E1F2140, 0 0 1px #1E1F214F`, reserved for movable cards), **overlay** (`0 8px 12px #1E1F2126, 0 0 1px #1E1F214F`, modals, dropdowns, popovers). Plus overflow shadows for scroll edges.

### Radius, border, focus

Graded by role: `xsmall` 2 (badges, checkboxes), `small` 4 (lozenges, tags, tooltips), `medium` 6 (buttons, inputs, selects, nav items), `large` 8 (cards, dropdown menus, floating UI), `xlarge` 12 (modals, large containers, tables), `full` (people: avatars). Focus ring: 2px `border.focused`, radius = element radius + 2.

### Components

- **Button**: appearances `default`, `primary`, `subtle`, `warning`, `danger`, `discovery`, link. Refresh `default` is transparent with a 1px `border` outline (same idea as our outlined secondary). Height 32px, compact 24px, radius 6, padding 12px inline. One primary per page or area; sentence case; verbs; avoid disabled buttons in forms.
- **Lozenge**: status/state only (workflow status, system state, priority), not categorisation (use a tag). Sentence case in the current docs, max width 200px, semantic colours (success, warning, danger, information, discovery, neutral) or accent colours. Tinted background, no dot.
- **Navigation**: top bar (product switcher, search, create, notifications, help, profile) + collapsible left sidebar for the current product/space. Selected item: `background.selected` + `text.selected`. *(Bar heights unverified.)*
- **Table** (dynamic table): small subtle column headers, row hover tint, inline actions in a "…" menu.
- **Inline edit**: read view that turns into a field on click, with confirm/cancel; used on Jira's issue view.
- **Flag**: toast with icon, title, text and actions, stacked in the bottom-left corner (slides in from the left).
- **Modal**: overlay surface, radius 12, widths in steps *(exact widths unverified)*, header / body / footer with right-aligned actions.
- **Empty state**: illustration, heading, one sentence, primary + secondary action.
- **Avatar**: round for people, square (radius) for containers (projects/spaces).

### Density, motion, states

- One default density: 14/20 text, 32px controls, 8px rhythm. Compact variants exist per component (button 24px, table), not as a global switch.
- Motion durations 50–600ms (`short` 150, `medium` 200, `long` 250); hover/pressed colour transitions 150ms with ease-out `cubic-bezier(0.4, 1, 0.6, 1)`. Flag enter 250ms, exit 200ms.
- Interaction states are tokens: neutral subtle hovered `#0515240F`, pressed `#0B120E24`; selected `#E9F2FE` + blue text; disabled background `#0515240F` + `text.disabled`.

## 3. Current vs target

Target = the recommended options in section 4. "Change" = size of the change in phase 2.

| Aspect | Current | Target | Change |
| --- | --- | --- | --- |
| UI kit | Mantine, wrapped in `shared/ui` | Same | None |
| Primary | `#1E40AF` | `#1868DB` ramp | Ramp in `theme.ts` |
| Neutrals | Slate (cool blue-grey) + Mantine gray | One warm-neutral scale, alpha borders | `theme.ts`, replace `color="gray"` |
| Semantic | success/warning/danger/info | + discovery; ADS-like hues | `theme.ts` |
| Status display | Badge + dot, radius 6 | Lozenge: tint, no dot, radius 4, sentence case | `StatusBadge` |
| Font | Inter | Inter | None |
| Body line height | 1.5 (21px) | 20px (≈1.43) | `theme.ts` |
| Headings | 22 / 16 / 14, weight 600 | 24 or 20 / 16 / 14, weight ≈650 | `theme.ts` |
| Field label | 13px / 500 / secondary | 12px or 14px / 600 / subtle | `theme.ts` |
| Spacing | 4…32, t-shirt keys | Same scale plus 2 and 40/48 | Add keys |
| Radius | 6 everywhere | 4 / 6 / 8 / 12 by role | `theme.ts` + overrides |
| Controls | Buttons 36 / 30, inputs 36 | 32 / 24 (compact), inputs 32 | Theme sizes, `.inputRow` |
| Table | Row 40, header 36 on `subtle` | Row 40, header 36, row hover tint | Small CSS |
| Elevation | None but overlays | Same + ADS overlay shadow | `theme.ts` shadows |
| Icons | Tabler, stroke 1.75 | Tabler, stroke 1.5, subtle icon colour | `icon` constant |
| Toast | Bottom-right | See conflict C8 | — |
| Dark mode | None, empty resolver | Semantic tokens only, no dark values | None |

## 4. Decisions

Each decision records its **Final decision**.

### D1. Which Jira style: legacy or refresh

| Option | Pros | Cons |
| --- | --- | --- |
| A. **Refresh** (current ADS: graded radius 4–12, 32px buttons, outlined default button, warm neutrals, 653 heading weight, sentence-case lozenges) | It is what ADS documents and ships today; our 6px radius and outlined secondary button already match; sentence case matches `ui.md` capitalisation | Warm-grey neutrals and olive success green look less "classic Jira" |
| B. Legacy (3px radius, `#0052CC`, `#172B4D` text, uppercase bold lozenges, grey-filled default button) | The look most people picture as "Jira" | Being phased out by ADS; uppercase lozenges conflict with our capitalisation rule; values live only in legacy theme files |

**Recommended: A.**

Final decision: A (refresh).

### D2. Primary colour

| Option | Pros | Cons |
| --- | --- | --- |
| A. **ADS-like blue** (`#1868DB`, ramp derived from it) | Reads as Jira at a glance; 5.2:1 with white text (AA); lighter and livelier than the current navy | All primary-coloured UI shifts hue slightly |
| B. Keep `#1E40AF`, only adopt neutrals and semantics | Smallest visual jump | Dark navy on a light warm-neutral UI looks heavier than Jira; "Jira style" mostly lost |
| C. Own blue between the two (e.g. `#1D5FD1`) | Avoids looking like a clone | Extra design work with no user benefit |

The project is open-source and neutral, so there is no brand colour to protect. **Recommended: A.**

Final decision: A (`#1868DB`).

### D3. Font

| Option | Pros | Cons |
| --- | --- | --- |
| A. **Keep Inter Variable** | Installed, self-hosted, full Vietnamese, tabular figures; variable weight axis allows ADS's 653 heading weight; metrics close to Atlassian Sans | None worth noting |
| B. System font stack (what ADS falls back to) | Zero bytes, native feel | Different rendering per OS (on-prem Linux desktops may get DejaVu); Vietnamese diacritics and tabular figures vary; breaks "same look everywhere" |
| C. Another open font (Be Vietnam Pro, Noto Sans) | Noto already used for PDF printing | New dependency for no gain; Be Vietnam Pro is wider, worse for dense tables |

**Recommended: A.**

Final decision: A (keep Inter Variable).

### D4. Icon library

| Option | Pros | Cons |
| --- | --- | --- |
| A. **Keep Tabler**, stroke 1.5 at 16px, `icon.subtle` colour | 79 icons already in use, 5,000+ available, outline style close to ADS refresh icons; only the `icon` constant in `theme.ts` changes | Slightly rounder than ADS icons |
| B. Lucide | Very close to ADS refresh line icons; popular | Replace 79 imports for a marginal gain; new dependency |
| C. Phosphor | Several weights | Heavier package, larger migration, style further from ADS |

**Recommended: A.**

Final decision: A (keep Tabler, stroke 1.5).

### D5. Density

| Option | Pros | Cons |
| --- | --- | --- |
| A. **One density, ADS default**: 14/20 text, 32px controls, 40px table rows, 8px rhythm | Matches Jira; denser than now (36→32px controls, 21→20px lines); one set of values to test | No "comfortable" mode for users who want more air |
| B. One density, compact (24px controls, 32px rows everywhere) | Maximum rows on screen | Below the 32px desktop touch target in `ui.md`; tiring for all-day users |
| C. User toggle (compact / default) | Flexible | Two sizes for every component and screen to test; nobody has asked for it |

Compact stays per component where it already exists (`CellGrid` 32px cells, `compact-xs` buttons). **Recommended: A.**

Final decision: A (one density, ADS default).

### D6. Dark mode

| Option | Pros | Cons |
| --- | --- | --- |
| A. Build it now | Done once | Every screen to test twice; not asked for by any user; `ui.md` excludes it from the first release |
| B. Prepare dark token values without shipping a switch | Values ready | Untested values rot; dark palettes need real screens to tune |
| C. **Semantic token names only, no dark values** (what `theme.ts` already does: `dark: {}`) | No extra work; adding dark later only fills the resolver; use alpha borders/neutrals (D1) which already suit dark | Dark mode later still needs a tuning pass |

**Recommended: C.**

Final decision: C (semantic names only, no dark values).

### D7. Technical strategy

| Option | Pros | Cons |
| --- | --- | --- |
| A. **Retheme Mantine**: `theme.ts` (ramps, sizes, radius, shadows, component `extend`s), `cssVariablesResolver`, and the CSS modules in `shared/ui` | Areas already contain no colours, sizes or CSS, so screens change without being touched; no new dependency; lint keeps working | Some ADS details need Mantine Styles API overrides (lozenge, button outline, table hover) |
| B. Wrap every Mantine component in own components | Full control | Most convention-bearing ones are already wrapped; wrapping the rest (`Button`, `TextInput`, `Group`…) is churn |
| C. Replace Mantine with `@atlaskit/*` components | Exact ADS look | Brings Atlassian icons/fonts expectations and a different styling engine; rewrites every screen; licensing of assets to check; needs an ADR (UI kit is ADR-0008) |
| D. Write an own component library | Total control | Months of work for a solo project |

**Recommended: A** (keep Mantine).

Final decision: A (retheme Mantine).

### D8. Migration

| Option | Pros | Cons |
| --- | --- | --- |
| A. **All at once, layered**: one branch, one PR; commits in order tokens (`theme.ts`) → `shared/ui` components → app shell → spot fixes in areas → `ui.md` and `/dev/ui` | Values are central, so a theme change already flips every screen at once; no period where two looks coexist; e2e and `/dev/ui` check everything | One large visual PR to review |
| B. Module by module (HRM, then Sales…) | Smaller steps | Not possible cleanly: one global theme; would need two themes or scoped overrides, then removal |

Hard-coded values: replace the ~15 off-scale values listed in section 1 by tokens in the same PR; no new lint rule, since they are few and confined to `shared/ui`. **Recommended: A.**

Final decision: A (all at once, layered).

### D9. Naming

The system needs no product name: it is `theme.ts` plus `shared/ui`, and only this app uses it.

| Option | Pros | Cons |
| --- | --- | --- |
| A. **No separate name; keep the `--mmerp-` prefix**; name tokens by role with the ADS-style grammar `property-role-emphasis-state` (`--mmerp-color-text-subtle`, `--mmerp-color-background-selected`, `--mmerp-elevation-surface-sunken`); keep Mantine's t-shirt keys for spacing and radius because Mantine props need them | Already in use; role grammar is generic, not a trademark | Two naming styles side by side (Mantine keys and `--mmerp-` roles) |
| B. Give it a name (e.g. a short neutral word) and prefix tokens with it | Feels like a "system" | Rename work, no user value |
| C. Numeric ADS-style spacing keys (`space-100`) instead of t-shirt sizes | Closer to ADS | Every `gap="xs"` (100+ uses) changes; Mantine props expect its own keys |

**Recommended: A.**

Final decision: A (no separate name, `--mmerp-` prefix).

### D10. Conflicts between the current rules and the new style

Each row needs a yes/no. "Keep" means `ui.md` wins over ADS.

| # | Conflict | Recommendation | Final decision |
| --- | --- | --- | --- |
| C1 | Status shown as badge + dot (`StatusBadge`) vs ADS lozenge (tint, no dot, radius 4) | Adopt the lozenge look; keep text always present (colour is never the only signal) | Accepted |
| C2 | Status colours: ADS often shows "in progress" in blue (information); ours shows `pending_approval` in `warning` | Keep ours (fixed colours, `ui.md`); module progress keeps grey / information | Accepted |
| C3 | Line height 1.5 vs ADS fixed 20px for 14px text | Adopt 20px (denser tables and forms) | Accepted |
| C4 | Page title 22/600 vs ADS heading 20 or 24, weight 653 | Page title 24 (`heading.large`), section 16, sub-section 14, weight 650 | Accepted |
| C5 | Field label 13px/500 vs ADS label 12px semibold *(label size unverified)* | 12px / 600 / `text.subtle`; 13px is off-scale | Accepted |
| C6 | Radius 6 everywhere vs graded 4/6/8/12 | Adopt: 4 lozenge/badge/tooltip, 6 controls, 8 menus/popovers/tiles/table frame, 12 modals; avatars full | Accepted |
| C7 | Button 36/30px and input 36px vs ADS 32/24 *(ADS input height unverified)* | 32px for page-level controls and inputs, 24px for section/in-content buttons; the "same height in a strip" rule stays | Accepted |
| C8 | Success toast bottom-right vs ADS flag bottom-left | Keep bottom-right: the bottom-left is where the navigation bar ends and the user's eye is not | Accepted |
| C9 | ADS left-aligns buttons on full-page forms; ours are always right-aligned | Keep ours (sticky footer, consistent placement) | Accepted |
| C10 | ADS empty states use illustrations; `ui.md` forbids illustrations | Keep icon + sentence; ADS illustrations may not be copied anyway | Accepted |
| C11 | Selected item: ADS adds blue text to the light blue background | Adopt blue text + tint; still no accent bar or coloured border | Accepted |
| C12 | Neutral table header on `subtle` background vs ADS plain header with a bottom rule | Keep the tinted header (helps wide reconciliation tables), switch tint to `surface.sunken` value | Accepted |
| C13 | Avatar colours from Mantine defaults; ADS: accent palette, round for people, square for containers | Move avatar colours to accent tokens in `theme.ts`; org-unit and product icons stay square (radius 4/6) | Accepted |
| C14 | Two neutral systems (Slate tokens + Mantine `gray`) | One neutral ramp replaces both; `color="gray"` maps to it | Accepted |
| C15 | ADS advises against disabled buttons in forms | Already compatible: only a submitting button is blocked; no change | Accepted |
| C16 | Motion: ours ≤ 200ms with Mantine curves; ADS 150ms hover transitions, 250ms flag enter | Keep ≤ 200ms; set hover transitions to 150ms ease-out; reduced motion still respected | Accepted |
| C17 | No *discovery* (purple) colour today | Add the token, use only for "new / beta" markers; do not invent uses | Accepted |

## 5. What phase 2 will touch

Only after the decisions above are final:

1. `ui.md`: tokens, type, spacing, radius, status, buttons, checklist, rewritten to the decided values (it stays the source of truth).
2. `web/src/shared/ui/theme.ts`: ramps, resolver variables, sizes, radius, shadows, component overrides, `icon` constant.
3. `shared/ui` and `shared/document` CSS modules and components (`StatusBadge`, `DataTable`, `Form`, `avatar`, `states`, `Tile`), the app shell, and the few off-scale values.
4. `/dev/ui` review at 1280px, 1024px and 375px; `make lint`, `make test`, `make e2e`.

Not done yet in phase 2: avatar colours from theme accent tokens (C13), 150ms hover transitions (C16), and a visual pass of every screen on `/dev/ui` and in the areas.
