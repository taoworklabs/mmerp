# `@mmerp/ui` Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the standalone UI kit `@mmerp/ui` at `packages/ui`: tokens seeded once from today's theme, every primitive of the parity inventory on Base UI or plain HTML/CSS, five hooks, Storybook with a11y and story tests, and a CI workflow. Nothing in the app changes.

**Architecture:** A private package outside `web/`, with its own lockfile and tooling. `tokens.ts` is the package's own source of values. A script renders it into `tokens.css` (`--mmerp-*` custom properties). Primitives are CSS modules reading those variables, wrapping Base UI parts where Base UI has one. Storybook stories are the catalogue, and through `@storybook/addon-vitest` they are also browser tests: a11y checks plus `play` functions for keyboard behaviour.

**Tech Stack:** React 19, TypeScript 5.9, `@base-ui/react` 1.9, Vite 8, Vitest 5 (browser mode, Playwright Chromium), Storybook 10.6 (`@storybook/react-vite`, `addon-a11y`, `addon-vitest`), `vitest-browser-react` 2.3, ESLint 10 with `typescript-eslint` and `eslint-plugin-i18next`, pnpm 11, Node 24.

**Spec:** `docs/superpowers/specs/2026-10-20-own-ui-kit-design.md`. This plan covers its "Phase 1" section, plus the parts of Goals, Architecture, Tokens, Primitives, Parity inventory, Dates and Toasts that phase 1 builds. Read the spec first. Proposed decision record: `docs/adr/0037-own-ui-kit-on-base-ui.md`.

## Global Constraints

- Everything is created under `packages/ui/`. The only file created outside it is `.github/workflows/ui.yml`. Do not modify `web/`, `Makefile`, `Dockerfile`, `.github/workflows/ci.yml`, or any doc except this plan and the spec.
- Work on branch `feat/own-ui-kit`. One commit per task, Conventional Commits with scope `ui` (e.g. `feat(ui): …`). Every commit message ends with `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`. Stage explicit paths, never `git add -A` or `git add .`.
- Runtime dependency: `@base-ui/react` only (`^1.9.0`). `react` and `react-dom` (`^19`) are peer deps. No `clsx`, no dayjs, no date-fns, no Mantine.
- `packages/ui/src` must not import `web/`, `i18next`, `react-i18next`, `react-router`, `react-hook-form`, `@tanstack/*`, `openapi-fetch`, `dayjs` or `@mantine/*`. ESLint enforces this (Task 1).
- No user-facing text in primitives. Every label, aria label, placeholder and message is a prop. `i18next/no-literal-string` runs on `src/**/*.tsx` except `*.stories.tsx`. Punctuation-only strings such as `*` and `/` are fine.
- Colour literals (`#…`, `rgb(`, `rgba(`, `hsl(`, `hsla(`) appear only in `src/tokens/tokens.ts` and the generated `src/tokens/tokens.css`. Everything else reads `var(--mmerp-…)`.
- Every primitive:
  - spreads the remaining props onto its root element;
  - accepts `className`, merged with `cx` from `src/internal/cx.ts`;
  - accepts `ref` as a normal prop (React 19).
- Every control root (button, input, select trigger, combobox input, segmented control item, pagination button, burger) carries the attribute `data-mmerp-control`. `styles/global.css` uses it for height and the touch target.
- Stories:
  - one file per primitive, next to it, `<Name>.stories.tsx`, `title: '<Folder>/<Name>'` (folders: `Foundations`, `Layout`, `Typography`, `Input`, `Overlay`, `Navigation`, `Feedback`, `Data display`, `Dates`);
  - one story per variant and per state listed in the task;
  - `play` functions import from `storybook/test`.
- Everything public is exported from `src/index.ts`, and each task adds its exports there.
- Commands run from `packages/ui`:
  - `pnpm typecheck` (`tsc --noEmit`);
  - `pnpm lint` (ESLint, then `sh scripts/check-colors.sh`);
  - `pnpm test` (`vitest run`: projects `unit` and `storybook`);
  - `pnpm gen` (`node scripts/gen-tokens.ts`);
  - `pnpm storybook` (`storybook dev -p 6006`);
  - `pnpm build-storybook` (`storybook build`).
- Motion: 150ms (`--mmerp-motion-duration`), never above 200ms. Everything honours `prefers-reduced-motion` through `global.css`.
- Primitive APIs below are decided. When Base UI forces a different shape, keep the decided prop names and adapt inside the primitive.

## Review Focus

1. **Typing in a typeable select.** A user types a partial, accent-free name ("nguyen") and expects "Nguyễn Văn A" to match. Typing nonsense shows the "nothing found" text. Escape or blur without choosing puts back the label of the current value. Covered in Task 6 (`matchesQuery` unit test, Combobox `play`).
2. **A hand-typed invalid date.** `31/02/2026` typed and blurred must not change the value. The text reverts to the formatted current value, or to empty. Covered in Task 10 (DateInput `play`).
3. **Focus after overlays.** Closing a dialog by Escape returns focus to its trigger. Escape in a menu inside a dialog closes only the menu. Covered in Task 7 (Dialog `play`).
4. **Small screens.** Below 1024px every control is at least 44×44px. At 1280px a default button is exactly 30px tall. Covered in Task 4 (`touchTarget.test.tsx`).
5. **Long text.** A long label in `Text truncate`, a `NavLink`, a `Select` trigger or a `Badge` ellipsises inside its box instead of widening the layout. Covered in Task 3 (Text `play`), Task 6 (Select `play`) and Task 8 (NavLink `play`).

---

### Task 1: Package scaffold, tokens, base styles, tooling, CI

**Files:**
- Create in `packages/ui/`:
  - `package.json`, `tsconfig.json`, `.gitignore` (`storybook-static/`), `eslint.config.js`, `vitest.config.ts`;
  - `.storybook/main.ts`, `.storybook/preview.ts`;
  - `scripts/gen-tokens.ts`, `scripts/check-colors.sh`;
  - `src/tokens/tokens.ts`, `src/tokens/render.ts`, `src/tokens/tokens.css` (generated), `src/tokens/render.test.ts`, `src/tokens/Tokens.stories.tsx`;
  - `src/styles/reset.css`, `src/styles/global.css`, `src/styles/index.css`;
  - `src/internal/cx.ts`, `src/index.ts`.
- Create: `.github/workflows/ui.yml`

**Interfaces:**
- Produces:
  - `tokens` (the `const` object below), plus types `Space = keyof typeof tokens.space`, `Radius`, `FontSize`, `ColorName = 'primary'|'success'|'warning'|'danger'|'info'|'discovery'|'gray'`;
  - `breakpoint` (em strings) and `icon` (Tabler sizes), both exported from `src/index.ts`;
  - `renderTokensCss(): string`;
  - `cx(...parts: (string | false | null | undefined)[]): string`;
  - the stylesheet entry `@mmerp/ui/styles.css` (`src/styles/index.css`, which imports `tokens.css`, `reset.css` and `global.css`);
  - CSS variables, named exactly as in step 3.

- [ ] **Step 1: Scaffold the package and tooling**

  - **`package.json`**:
    - `"name": "@mmerp/ui"`, `"private": true`, `"type": "module"`;
    - `"exports": { ".": "./src/index.ts", "./styles.css": "./src/styles/index.css" }`;
    - the scripts listed in Global Constraints;
    - `dependencies: { "@base-ui/react": "^1.9.0" }`;
    - `peerDependencies: react/react-dom ^19`;
    - devDependencies, pinned to the versions in Tech Stack:
      - `react`, `react-dom` `^19.3.0`, `@types/react`, `@types/react-dom`;
      - `typescript ^5.9.3`, `vite ^8.3.2`, `@vitejs/plugin-react ^6.1.1`;
      - `vitest`, `@vitest/browser`, `@vitest/browser-playwright` `^5.0.3`, `playwright ^1.63.0`, `vitest-browser-react ^2.3.0`;
      - `storybook`, `@storybook/react-vite`, `@storybook/addon-a11y`, `@storybook/addon-vitest` `^10.6.2`;
      - `eslint ^10.12.0`, `@eslint/js ^10.0.1`, `typescript-eslint ^8.71.0`, `eslint-plugin-i18next ^6.1.5`, `globals ^17.13.0`, `@tabler/icons-react ^3.48.0` (stories only).
  - **`tsconfig.json`**: copy the compiler flags of `web/tsconfig.json` (strict, `noUncheckedIndexedAccess`, `verbatimModuleSyntax`, `moduleResolution: bundler`, `jsx: react-jsx`, `noEmit`), plus `allowImportingTsExtensions: true`. Include `src`, `scripts`, `.storybook`, `vitest.config.ts`.
  - **`eslint.config.js`**: mirror the shape of `web/eslint.config.js`:
    - `js.configs.recommended` and `tseslint.configs.recommended`;
    - `i18next.configs['flat/recommended']` on `src/**/*.tsx`, ignoring `src/**/*.stories.tsx`;
    - `no-restricted-imports` with patterns for every package listed in Global Constraints, plus `**/web/**`, with the message `@mmerp/ui knows nothing of the app; take it as a prop.`;
    - ignore `storybook-static`.
  - **`scripts/check-colors.sh`**: same regex as `web/scripts/check-colors.sh`, run over `src`, excluding `./tokens/tokens.ts` and `./tokens/tokens.css`. Message: `colour literals found outside src/tokens`.
  - **`vitest.config.ts`**: `test.projects`:
    - `unit`: include `src/**/*.test.{ts,tsx}`, `browser: { enabled: true, provider: playwright(), headless: true, instances: [{ browser: 'chromium' }] }`;
    - `storybook`: `plugins: [storybookTest({ configDir: '.storybook' })]`, same browser block.
  - **`.storybook/main.ts`**: framework `@storybook/react-vite`, `stories: ['../src/**/*.stories.tsx']`, addons `@storybook/addon-a11y` and `@storybook/addon-vitest`.
  - **`.storybook/preview.ts`**: imports `../src/styles/index.css`, sets `parameters: { a11y: { test: 'error' } }`.
  - **`src/internal/cx.ts`**: the `cx` helper, in one line.

Run: `cd packages/ui && pnpm install`
Expected: lockfile `packages/ui/pnpm-lock.yaml` created. Then `pnpm exec playwright install chromium`.

- [ ] **Step 2: Write the failing test `src/tokens/render.test.ts`**

```ts
import { describe, expect, it } from 'vitest'
import generated from './tokens.css?raw'
import { renderTokensCss } from './render'

describe('renderTokensCss', () => {
  const css = renderTokensCss()
  it('emits ramps, semantic colours and tints', () => {
    expect(css).toContain('--mmerp-color-primary-6: #1868DB;')
    expect(css).toContain('--mmerp-color-gray-0: #F8F8F8;')
    expect(css).toContain('--mmerp-color-text: #292A2E;')
    expect(css).toContain('--mmerp-color-dimmed: #505258;')
    expect(css).toContain('--mmerp-color-border: #0B120E24;')
    expect(css).toContain('--mmerp-color-error: #AE2E24;')
    expect(css).toContain('--mmerp-color-danger-light: #FFECEB;')
    expect(css).toContain('--mmerp-color-danger-light-color: #AE2E24;')
    expect(css).toContain('--mmerp-color-gray-light: #F0F1F2;') // neutral tint starts one step darker
  })
  it('emits sizes in rem from px', () => {
    expect(css).toContain('--mmerp-control-height: 1.875rem;')
    expect(css).toContain('--mmerp-control-height-lg: 2.5rem;')
    expect(css).toContain('--mmerp-space-md: 1rem;')
    expect(css).toContain('--mmerp-radius-lg: 0.5rem;')
    expect(css).toContain('--mmerp-font-size-xl: 1.25rem;')
    expect(css).toContain('--mmerp-content-width: 75rem;')
  })
  it('keeps breakpoints and icon sizes out of CSS', () => {
    expect(css).not.toContain('breakpoint')
    expect(css).not.toContain('--mmerp-icon')
  })
  it('matches the committed tokens.css', () => {
    expect(generated).toBe(css)
  })
})
```

- [ ] **Step 3: Write `src/tokens/tokens.ts`, `src/tokens/render.ts` and `scripts/gen-tokens.ts`**

  **`tokens.ts`** is seeded once from `web/src/shared/ui/theme.ts`, its `cssVariablesResolver`, and `global.css`. Copy the values verbatim. Sizes are px numbers.
  - **Ramps.** `color.ramp`: the seven ramps (`primary`, `success`, `warning`, `danger`, `info` = primary, `discovery`, `gray`).
  - **Semantic colours.** `color.semantic`, mapping name → value:
    - `text` gray 9, `dimmed` gray 7, `placeholder` gray 6, `disabled` gray 1;
    - `border` `#0B120E24`, `body` `#FFFFFF`, `error` danger 7;
    - `elevation-surface` `#FFFFFF`, `elevation-surface-sunken` gray 0;
    - `table-header` `#F4F5F6`, `input-border` `#0B120E33`;
    - `overlay-backdrop` `#1E1F2166` (new: dialog backdrop).
  - **Tints.** The tint offsets: `0` for every ramp, `1` for `gray`.
  - **Type.** `fontFamily` `'Inter Variable', system-ui, sans-serif`, `fontWeight: { regular: 400, medium: 500, semibold: 600, heading: 650 }`.
    - `fontSize`: xs 12, sm 14, md 14, lg 16, xl 20.
    - `lineHeight`: xs `1.34`, sm `1.43`, md `1.43`, lg `1.5`, xl `1.2`.
    - `heading`: h1 24/`1.17`, h2 16/`1.25`, h3 14/`1.43`.
  - **Space.** xxxs 2, xxs 4, xs 8, sm 12, md 16, lg 24, xl 32, xxl 48.
  - **Radius.** xs 2, sm 4, md 6, lg 8, xl 12.
  - **Shadows.** `shadow` xs to xl and `footer`, verbatim.
  - **Breakpoints.** `breakpoint`: xs `36em`, sm `48em`, md `64em`, lg `80em`, xl `96em`.
  - **Control heights.** `controlHeight: { sm: 30, lg: 40 }`.
  - **Layout widths.** `layoutWidth: { content: 1200, side: 360, aside: 480, detail: 1440 }`.
  - **Filter widths.** `filterWidth: { search: 280, searchMin: 160, chip: 320, menu: 240 }`.
  - **Icons.** `icon` as in `theme.ts`.
  - **Motion.** `motionDuration: 150`.
  - **Touch target.** `touchTarget: 44`.

  **`render.ts`** returns one `:root { … }` block with one declaration per line, two-space indent, and a trailing newline. It starts with the comment `/* Generated from tokens.ts by scripts/gen-tokens.ts. Do not edit. */`.
  - **Colour names.** `--mmerp-color-<ramp>-<0..9>`, `--mmerp-color-<semantic>`, and per ramp `--mmerp-color-<ramp>-light`, `-light-hover`, `-light-color`:
    - `-light` and `-light-hover` are steps offset and offset+1;
    - `-light-color` is step 7.
  - **Size names.**
    - `--mmerp-font-family`, `--mmerp-font-size-<k>`, `--mmerp-line-height-<k>`, `--mmerp-font-weight-<k>`;
    - `--mmerp-heading-<h>-size`, `--mmerp-heading-<h>-line-height`;
    - `--mmerp-space-<k>`, `--mmerp-radius-<k>`, `--mmerp-shadow-<k>`;
    - `--mmerp-control-height` (sm), `--mmerp-control-height-lg`;
    - `--mmerp-content-width`, `--mmerp-side-width`, `--mmerp-aside-width`, `--mmerp-detail-width`;
    - `--mmerp-filter-width-<kebab key>`;
    - `--mmerp-motion-duration` (ms), `--mmerp-touch-target` (px).
  - **Unit rule.** Every px size becomes `n/16` rem (no trailing zeros), except `--mmerp-touch-target`, which stays px.

  **`scripts/gen-tokens.ts`** writes `renderTokensCss()` to `src/tokens/tokens.css`. Node 24 runs it directly through type stripping.

- [ ] **Step 4: Generate and run the test**

Run: `pnpm gen && pnpm vitest run --project unit src/tokens`
Expected: PASS (4 tests).

- [ ] **Step 5: Write the base styles**

  **`reset.css`**:
  - `*, ::before, ::after { box-sizing: border-box }`, `body { margin: 0; position: relative }`;
  - body font from `--mmerp-font-family`, size `--mmerp-font-size-sm`, line height `--mmerp-line-height-sm`, colour `--mmerp-color-text`, background `--mmerp-color-body`;
  - `#root { isolation: isolate }`, which Base UI needs for portals.

  **`global.css`**:
  - `[data-mmerp-control] { height: var(--mmerp-control-height) }` for single-line controls. Multi-line ones opt out with `[data-mmerp-control='multiline']`.
  - `:focus-visible { outline: 2px solid var(--mmerp-color-primary-6); outline-offset: 2px }`.
  - `@media (max-width: 63.99em) { [data-mmerp-control] { min-height: var(--mmerp-touch-target); min-width: var(--mmerp-touch-target) } }`.
  - `@media (prefers-reduced-motion: reduce) { *, ::before, ::after { animation-duration: 0.01ms !important; transition-duration: 0.01ms !important } }`.

- [ ] **Step 6: Write `Tokens.stories.tsx`** (`title: 'Foundations/Tokens'`)

Write one story each: Colours (every ramp step and semantic colour as a swatch with its variable name), Type, Space, Radius, Shadow. It reads `tokens` and renders inline-styled swatches. Inline styles are fine in stories; no colour literal appears because values come from `tokens`.

- [ ] **Step 7: Write `.github/workflows/ui.yml`**

```yaml
name: UI kit
on:
  push:
    paths: ['packages/ui/**', '.github/workflows/ui.yml']
  workflow_dispatch:
concurrency:
  group: ui-${{ github.ref }}
  cancel-in-progress: true
jobs:
  ui:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: packages/ui
    steps:
      - uses: actions/checkout@v7
      - uses: pnpm/action-setup@v6
        with:
          version: 11
      - uses: actions/setup-node@v7
        with:
          node-version: 24
          cache: pnpm
          cache-dependency-path: packages/ui/pnpm-lock.yaml
      - run: pnpm install --frozen-lockfile
      - run: pnpm gen && git diff --exit-code -- src/tokens/tokens.css
      - run: pnpm typecheck
      - run: pnpm lint
      - run: pnpm exec playwright install --with-deps chromium
      - run: pnpm test
      - run: pnpm build-storybook
```

- [ ] **Step 8: Verify everything**

Run: `pnpm typecheck && pnpm lint && pnpm test && pnpm build-storybook`
Expected:
- every command exits 0;
- the storybook project runs the 5 token stories with no a11y violations;
- `git status` shows changes only under `packages/ui/` and `.github/workflows/ui.yml`.

Check that the lint guards bite: add `import 'i18next'` to `src/index.ts`, run `pnpm lint`, expect a `no-restricted-imports` error, then remove the line.

- [ ] **Step 9: Commit**

```bash
git add packages/ui .github/workflows/ui.yml
git commit -m "feat(ui): @mmerp/ui package with tokens, base styles, Storybook and CI"
```

---

### Task 2: Hooks

**Files:**
- Create: `packages/ui/src/hooks/{useDisclosure,useMediaQuery,useDebouncedValue,useLocalStorage,useElementSize}.ts`, `packages/ui/src/hooks/hooks.test.tsx`
- Modify: `packages/ui/src/index.ts`

**Interfaces:**
- Produces:
  - `useDisclosure(initial?: boolean): [opened: boolean, handlers: { open(): void; close(): void; toggle(): void }]`
  - `useMediaQuery(query: string, initial?: boolean): boolean`: subscribes with `matchMedia`, so it is correct on the first client render.
  - `useDebouncedValue<T>(value: T, ms: number): [debounced: T, cancel: () => void]`
  - `useLocalStorage<T>(key: string, defaultValue: T): [value: T, setValue: (v: T) => void]`: JSON-serialised. An unreadable or unparsable entry yields `defaultValue`.
  - `useElementSize<T extends HTMLElement>(): { ref: (el: T | null) => void; width: number; height: number }`: uses `ResizeObserver`.

- [ ] **Step 1: Write failing tests in `hooks.test.tsx`** (`renderHook` from `vitest-browser-react`, `page` from `vitest/browser`)
  - **`useDisclosure`:** `toggle()` flips `false → true → false`.
  - **`useMediaQuery`:** after `await page.viewport(800, 600)`, `(max-width: 63.99em)` returns `true`. After `await page.viewport(1280, 800)` and a rerender it returns `false`.
  - **`useDebouncedValue`:**
    - with `vi.useFakeTimers()`, rerendering with `'b'` keeps `'a'` until `vi.advanceTimersByTime(300)`, then gives `'b'`;
    - `cancel()` before the timeout keeps `'a'`.
  - **`useLocalStorage`:**
    - `setValue({ a: 1 })` writes `localStorage['k'] === '{"a":1}'`;
    - with `localStorage['k'] = 'not json'`, the hook returns `defaultValue`.
  - **`useElementSize`:** a rendered `div` of `width: 120px; height: 40px` reports `{ width: 120, height: 40 }` (`expect.poll`).

- [ ] **Step 2: Run and see them fail**

Run: `pnpm vitest run --project unit src/hooks`
Expected: FAIL (modules not found).

- [ ] **Step 3: Implement the five hooks** with the signatures above.

- [ ] **Step 4: Run tests**

Run: `pnpm vitest run --project unit src/hooks`
Expected: PASS.

- [ ] **Step 5: Export and commit**

```bash
git add packages/ui/src/hooks packages/ui/src/index.ts
git commit -m "feat(ui): hooks for disclosure, media query, debounce, local storage and element size"
```

---

### Task 3: Layout and typography primitives

**Files:**
- Create under `packages/ui/src/primitives/layout/`: `Box`, `Stack`, `Group`, `Grid`, `SimpleGrid`, `Center`, `Divider`, `Paper`, `VisuallyHidden`. Each gets `.tsx`, plus `.module.css` where it needs one, and `.stories.tsx`.
- Create under `packages/ui/src/primitives/typography/`: `Text`, `Title`, `Anchor`, `Code`, `List`. Each gets `.tsx`, `.module.css` and `.stories.tsx`.
- Modify: `packages/ui/src/index.ts`

**Interfaces:**
- Consumes: `Space`, `cx`, the token variables (Task 1).
- Produces, where every component also takes its element's own props:
  - `RenderProp`: re-export of Base UI's `useRender.RenderProp`. Polymorphic primitives take `render?: RenderProp`, e.g. `render={<span />}` or `render={<Link to="/" />}`, implemented with `useRender`.
  - `Box`: a `div`, plus `render`.
  - `Stack`:
    - `gap?: Space | 0` (default `'md'`);
    - `align?: 'start'|'center'|'end'|'stretch'` (default `'stretch'`);
    - `justify?: 'start'|'center'|'end'|'space-between'`;
    - `render`.
  - `Group`:
    - `gap?: Space | 0` (default `'md'`);
    - `align?` (default `'center'`), `justify?`;
    - `wrap?: boolean` (default `true`), `grow?: boolean`;
    - `render`.
  - `Grid`, `Grid.Col`:
    - `Grid`: `columns?: number` (default 12), `gap?: Space` (default `'md'`);
    - `Grid.Col`: `span?: number | 'auto'`.
  - `SimpleGrid`:
    - `cols: number | { base: number; sm?: number; md?: number; lg?: number }` (breakpoints `48em`, `64em`, `80em`);
    - `spacing?: Space` (default `'md'`).
  - `Center`: `inline?: boolean`.
  - `Divider`: Base UI `Separator`, `orientation?: 'horizontal'|'vertical'`, colour `--mmerp-color-border`.
  - `Paper`:
    - `withBorder?: boolean`, `padding?: Space | 0` (default 0);
    - `shadow?: 'xs'|'sm'`;
    - background `--mmerp-color-elevation-surface`, radius `md`.
  - `VisuallyHidden`.
  - `Text`:
    - `size?: 'xs'|'sm'|'md'|'lg'|'xl'` (default `'sm'`), with font size and line height from tokens;
    - `tone?: 'default'|'dimmed'|'danger'|'success'|'warning'|'primary'`: `dimmed` uses `--mmerp-color-dimmed`; the others use `-light-color`;
    - `weight?: 400|500|600|650`, `align?: 'start'|'center'|'end'`;
    - `truncate?: boolean`, `lineClamp?: number`;
    - `render` (default `<p>`, margin 0).
  - `Title`: `order?: 1|2|3` (default 1), heading tokens, weight 650, margin 0.
  - `Anchor`:
    - an `<a>`, with `tone?` as `Text`;
    - `underline?: 'hover'|'always'|'never'` (default `'hover'`);
    - `render`; colour `--mmerp-color-primary-6`.
  - `Code`: `block?: boolean` (`<pre><code>`), background sunken surface, radius sm.
  - `List`, `List.Item`: `type?: 'unordered'|'ordered'`, `spacing?: Space`.

- [ ] **Step 1: Write the stories, with `play` assertions on computed style**
  - **Stack:** `gap="md"` → `getComputedStyle(el).rowGap === '16px'`.
  - **Group:** `wrap={false}` → `flexWrap === 'nowrap'`.
  - **Text:**
    - `size="xs"` → `fontSize === '12px'`;
    - `tone="dimmed"` → `color === 'rgb(80, 82, 88)'`;
    - **Truncate** story (Review Focus 5): text 600 characters long inside a 200px box, with `truncate` → `textOverflow === 'ellipsis'` and `el.scrollWidth > el.clientWidth`, and the box stays 200px wide.
  - **Anchor:** `render={<button type="button" />}` renders a `button`.
  - **Title:** `order={2}` renders `h2` at 16px.
  - Every other primitive gets a default story plus one story per prop value listed above.

- [ ] **Step 2: Run and see them fail**

Run: `pnpm vitest run --project storybook src/primitives/layout src/primitives/typography`
Expected: FAIL (imports missing).

- [ ] **Step 3: Implement the primitives** to the interfaces above. Gaps and spacing map to `var(--mmerp-space-<k>)` through a `data-gap` attribute or a CSS custom property set inline on the root. No generic `m`/`p`/`c` props.

- [ ] **Step 4: Run the stories, typecheck and lint**

Run: `pnpm vitest run --project storybook src/primitives/layout src/primitives/typography && pnpm typecheck && pnpm lint`
Expected: PASS, no a11y violations.

- [ ] **Step 5: Export and commit**

```bash
git add packages/ui/src/primitives/layout packages/ui/src/primitives/typography packages/ui/src/index.ts
git commit -m "feat(ui): layout and typography primitives"
```

---

### Task 4: Buttons and display primitives

**Files:**
- Create under `packages/ui/src/primitives/input/`: `Button`, `ActionIcon`, `UnstyledButton` (`.tsx`, `.module.css`, `.stories.tsx`).
- Create under `packages/ui/src/primitives/feedback/`: `Badge`, `Loader`, `Skeleton`, `Alert`, `Indicator` (same).
- Create under `packages/ui/src/primitives/data-display/`: `ThemeIcon`, `Avatar` (same).
- Create: `packages/ui/src/styles/touchTarget.test.tsx`
- Modify: `packages/ui/src/index.ts`

**Interfaces:**
- Consumes: `cx`, `RenderProp` (Task 3), `ColorName` (Task 1).
- Produces:
  - `ButtonVariant = 'filled'|'light'|'subtle'|'default'|'outline'`.
  - `Button` (Base UI `Button`):
    - `variant?: ButtonVariant` (default `'filled'`);
    - `color?: 'primary'|'gray'|'danger'|'success'|'warning'` (default `'primary'`);
    - `size?: 'xs'|'sm'` (default `'sm'`, height `--mmerp-control-height`; `xs` is 24px);
    - `leftSection?`, `rightSection?: ReactNode`;
    - `loading?: boolean`: disables, sets `aria-busy`, shows `Loader` in place of `leftSection`;
    - `fullWidth?`, `render?`;
    - weight 500, radius md, `data-mmerp-control`.
    - Variant looks:
      - `default`: surface background with a `--mmerp-color-border` edge, text colour;
      - `light`: `-light` background with `-light-color` text;
      - `subtle`: transparent, `-light` on hover;
      - `filled`: step 6 background, white text through `--mmerp-color-body`;
      - `outline`: transparent with a step 6 edge.
  - `ActionIcon`: the same `variant`, `color`, `size` and `loading` as `Button`, square, plus a required `'aria-label': string`.
  - `UnstyledButton`: Base UI `Button` with no styles beyond a reset.
  - `Badge`:
    - `color?: ColorName` (default `'primary'`), `variant?: 'light'|'filled'|'outline'` (default `'light'`);
    - `light`: background step 1 (`gray`: step 2), text step 8;
    - radius sm, font xs, weight 500, no uppercase, single line with ellipsis.
  - `Loader`: `size?: 'xs'|'sm'|'md'`, `label: string` (rendered as `role="status"` and `aria-label`).
  - `Skeleton`: `width?`, `height?: number | string`, `circle?: boolean`, `visible?: boolean` (default `true`; when `false` it renders its children).
  - `Alert`:
    - `color?: 'info'|'success'|'warning'|'danger'`, `title?: ReactNode`, `icon?: ReactNode`, `children`;
    - `role="alert"` for `danger`, otherwise `role="status"`.
  - `Indicator`: `label?: ReactNode` (a dot when absent), `color?: ColorName`, `disabled?: boolean`, `children` (the element it decorates).
  - `ThemeIcon`: `color?: ColorName`, `variant?: 'light'|'filled'`, `size?: number` (default 30), radius md.
  - `Avatar` (Base UI `Avatar`): `src?`, `alt: string`, `size?: number` (default 30), `color?: ColorName`, `children` (initials fallback).

- [ ] **Step 1: Write `touchTarget.test.tsx`** (Review Focus 4). It renders `Button`, `ActionIcon` (with `aria-label="x"`) and `Button size="xs"` with the stylesheet imported.
  - After `await page.viewport(800, 600)`: every `getBoundingClientRect()` has height ≥ 44 and width ≥ 44.
  - After `await page.viewport(1280, 800)`: the default `Button` is exactly 30px tall and the `xs` button is 24px.

- [ ] **Step 2: Write the stories**
  - **Button:**
    - one story per variant × `primary`/`gray`/`danger`; then `xs`, `loading`, `disabled`, with sections, `fullWidth`;
    - `play` on `loading`: the button has `aria-busy="true"` and is disabled;
    - `play` on default: Space key fires `onClick` (`fn()` from `storybook/test`).
  - **Badge:** every colour, with a **Long** story: a 300-character label in a 120px box ellipsises (Review Focus 5).
  - **Loader:** each size. **Skeleton:** visible, hidden, circle. **Alert:** each colour with title and icon. **Indicator:** dot, label, disabled. **ThemeIcon:** light, filled. **Avatar:** image, initials.

- [ ] **Step 3: Run and see them fail**

Run: `pnpm vitest run src/styles src/primitives/input src/primitives/feedback src/primitives/data-display`
Expected: FAIL.

- [ ] **Step 4: Implement** to the interfaces above.

- [ ] **Step 5: Run tests, typecheck, lint**

Run: `pnpm vitest run src/styles src/primitives && pnpm typecheck && pnpm lint`
Expected: PASS.

- [ ] **Step 6: Export and commit**

```bash
git add packages/ui/src/primitives/input packages/ui/src/primitives/feedback packages/ui/src/primitives/data-display packages/ui/src/styles/touchTarget.test.tsx packages/ui/src/index.ts
git commit -m "feat(ui): buttons, badges, loaders, alerts, avatars and the touch target"
```

---

### Task 5: Field and basic inputs

**Files:**
- Create under `packages/ui/src/primitives/input/`: `Field`, `TextInput`, `PasswordInput`, `Textarea`, `NumberInput`, `Checkbox`, `FileInput`, `SegmentedControl` (`.tsx`, `.module.css`, `.stories.tsx`). `Field` gets its own CSS module, shared by the inputs through composition.
- Modify: `packages/ui/src/index.ts`

**Interfaces:**
- Consumes: `Button`, `ActionIcon` (Task 4), `cx`.
- Produces:
  - `FieldProps = { label?: ReactNode; description?: ReactNode; error?: ReactNode; required?: boolean; disabled?: boolean; readOnly?: boolean }`.
  - `Field` (Base UI `Field`):
    - `FieldProps & { children: ReactNode }`;
    - order: label, control, description, error;
    - label: font xs, weight 600, colour dimmed, `--mmerp-space-xxs` below;
    - required mark `*` in `--mmerp-color-error`, `aria-hidden`;
    - the error is linked to the control through `aria-describedby` and sets `aria-invalid`.
  - `TextInput`:
    - `FieldProps & Omit<ComponentProps<'input'>, 'size'> & { leftSection?: ReactNode; rightSection?: ReactNode }`;
    - Base UI `Input`;
    - edge `--mmerp-color-input-border`, error edge `--mmerp-color-error`, focus edge primary 6;
    - placeholder colour `--mmerp-color-placeholder`.
  - `PasswordInput`: `TextInput` props plus a required `visibilityLabels: { show: string; hide: string }`. Its `ActionIcon` toggles `type`.
  - `Textarea`: `FieldProps & ComponentProps<'textarea'> & { autosize?: boolean; minRows?: number; maxRows?: number }`, with `data-mmerp-control="multiline"`.
  - `NumberInput` (Base UI `NumberField`):
    - `FieldProps & { value: number | null; onChange(v: number | null): void; min?; max?; step?; locale?: string; format?: Intl.NumberFormatOptions; placeholder? }`;
    - no stepper buttons; ArrowUp and ArrowDown step.
  - `Checkbox` (Base UI `Checkbox`): `{ checked?; defaultChecked?; indeterminate?; onChange?(checked: boolean): void; label?: ReactNode; description?; error?; disabled? }`, with the edge from gray step 4.
  - `FileInput`:
    - `FieldProps & { value: File | null; onChange(f: File | null): void; accept?: string; placeholder?: string; clearLabel: string }`;
    - a trigger showing the file name, opening a hidden `<input type="file">`;
    - an `ActionIcon` clear button labelled `clearLabel` when a file is set.
  - `SegmentedControl` (Base UI `ToggleGroup`, single):
    - `{ data: { value: string; label: ReactNode }[]; value: string; onChange(v: string): void; fullWidth?: boolean; disabled?: boolean; 'aria-label': string }`;
    - the chosen item sits on the surface with a shadow xs, on a sunken track.

- [ ] **Step 1: Write the stories**
  - **States for every input:** default, filled, placeholder, with description, required, error, disabled, read-only.
  - **InModal story:** a wrapper sets `--mmerp-control-height: var(--mmerp-control-height-lg)`, and `play` asserts the `TextInput` is 40px tall.
  - **`play` assertions:**
    - **TextInput** typing `abc` shows `abc`. **Field** with error: the input has `aria-invalid="true"` and an `aria-describedby` pointing at the error text.
    - **PasswordInput:** clicking `show` flips the input to `type="text"`, and the button label becomes `hide`.
    - **NumberInput** starting at 5: ArrowUp gives 6. Clearing the text calls `onChange(null)`.
    - **Checkbox:** Space toggles `aria-checked`. Indeterminate gives `aria-checked="mixed"`.
    - **FileInput:** `userEvent.upload` with a `File` shows its name. Clearing calls `onChange(null)`.
    - **SegmentedControl:** ArrowRight then Space moves the choice (`onChange` called with the second value).

- [ ] **Step 2: Run and see them fail**

Run: `pnpm vitest run --project storybook src/primitives/input`
Expected: the new stories FAIL.

- [ ] **Step 3: Implement** to the interfaces above.

- [ ] **Step 4: Run tests, typecheck, lint**

Run: `pnpm vitest run --project storybook src/primitives/input && pnpm typecheck && pnpm lint`
Expected: PASS.

- [ ] **Step 5: Export and commit**

```bash
git add packages/ui/src/primitives/input packages/ui/src/index.ts
git commit -m "feat(ui): field and basic inputs"
```

---

### Task 6: Select, Combobox, MultiSelect

**Files:**
- Create under `packages/ui/src/primitives/input/`: `Select`, `Combobox`, `MultiSelect` (`.tsx`, `.stories.tsx`), `options.module.css` (the shared dropdown and option look), `matchesQuery.ts`, `matchesQuery.test.ts`.
- Modify: `packages/ui/src/index.ts`

**Interfaces:**
- Consumes: `FieldProps`, `Field` (Task 5), `ActionIcon`, `Loader` (Task 4).
- Produces:
  - `SelectItem = { value: string; label: string; disabled?: boolean }`.
  - `matchesQuery(label: string, query: string): boolean`:
    - case-insensitive, ignores Vietnamese diacritics (NFD, strip combining marks, `đ`/`Đ` → `d`);
    - every whitespace-separated word of the query must appear in the label;
    - an empty query matches everything.
  - `Select` (Base UI `Select`, not typeable):
    - `FieldProps & { data: SelectItem[]; value: string | null; onChange(v: string | null): void; placeholder?: string; clearable?: boolean; clearLabel?: string }`;
    - `clearLabel` is required when `clearable` (a discriminated union).
  - `Combobox` (Base UI `Combobox`, typeable, single):
    - `FieldProps & { data: SelectItem[]; value: string | null; onChange(v: string | null): void; nothingFound: ReactNode; placeholder?: string; filter?: (item: SelectItem, query: string) => boolean; onSearchChange?(q: string): void; loading?: boolean; loadingLabel?: string; clearable?: boolean; clearLabel?: string }`;
    - `filter` defaults to `matchesQuery` on `label`;
    - `onSearchChange` serves remote search: the caller replaces `data`;
    - Escape or blur without choosing restores the current value's label (or empty);
    - clearing the text and blurring sets `null` only when `clearable`.
  - `MultiSelect` (Base UI `Combobox`, multiple):
    - `FieldProps & { data: SelectItem[]; value: string[]; onChange(v: string[]): void; nothingFound: ReactNode; placeholder?: string; removeLabel: (itemLabel: string) => string }`;
    - chosen items are chips with a remove button labelled `removeLabel(label)`.
  - Dropdown look:
    - radius lg, shadow md, surface background;
    - the chosen option tinted primary `-light` / `-light-color`, with no check mark;
    - the highlighted option on gray `-light`.

- [ ] **Step 1: Write the failing `matchesQuery.test.ts`**

```ts
import { describe, expect, it } from 'vitest'
import { matchesQuery } from './matchesQuery'

describe('matchesQuery', () => {
  it('ignores case and Vietnamese diacritics', () => {
    expect(matchesQuery('Nguyễn Văn A', 'nguyen van')).toBe(true)
    expect(matchesQuery('Đặng Thị Bích', 'dang bich')).toBe(true)
  })
  it('needs every word', () => {
    expect(matchesQuery('Trần Văn C', 'tran b')).toBe(false)
  })
  it('matches everything on an empty query', () => {
    expect(matchesQuery('Any', '  ')).toBe(true)
  })
})
```

Run: `pnpm vitest run --project unit src/primitives/input/matchesQuery.test.ts`. Expected: FAIL. Then implement and run again; expected PASS.

- [ ] **Step 2: Write the stories** (sample data: `Nguyễn Văn A`, `Đặng Thị Bích`, `Trần Văn C`, and one 200-character label)
  - **States for each component:** default, with value, disabled, read-only, error, clearable.
  - **`play` on Combobox** (Review Focus 1):
    - typing `nguyen` leaves exactly one option, `Nguyễn Văn A`;
    - typing `zzz` shows the `nothingFound` text;
    - with value `Trần Văn C`, typing `x` then Escape puts `Trần Văn C` back in the input, and `onChange` was not called;
    - ArrowDown then Enter on `Đặng Thị Bích` calls `onChange` with its value.
  - **`play` on Select:**
    - focus the trigger, then ArrowDown, ArrowDown, Enter: `onChange` is called with the second value;
    - **Long** story (Review Focus 5): the trigger is 240px wide with the 200-character label chosen; its value text has `textOverflow === 'ellipsis'` and the trigger stays 240px.
  - **`play` on MultiSelect:**
    - choosing two items shows two chips;
    - clicking the remove button of the first (found by its `removeLabel`) calls `onChange` with the remaining one.

- [ ] **Step 3: Run and see the stories fail**

Run: `pnpm vitest run --project storybook src/primitives/input`
Expected: the new stories FAIL.

- [ ] **Step 4: Implement** to the interfaces above.

- [ ] **Step 5: Run tests, typecheck, lint**

Run: `pnpm vitest run src/primitives/input && pnpm typecheck && pnpm lint`
Expected: PASS.

- [ ] **Step 6: Export and commit**

```bash
git add packages/ui/src/primitives/input packages/ui/src/index.ts
git commit -m "feat(ui): select, combobox and multiselect"
```

---

### Task 7: Overlays

**Files:**
- Create under `packages/ui/src/primitives/overlay/`: `Dialog`, `Drawer`, `Popover`, `Menu`, `Tooltip` (`.tsx`, `.module.css`, `.stories.tsx`).
- Create under `packages/ui/src/primitives/layout/`: `ScrollArea` (`.tsx`, `.module.css`, `.stories.tsx`).
- Modify: `packages/ui/src/index.ts`

**Interfaces:**
- Consumes: `ActionIcon`, `Button` (Task 4), `TextInput` (Task 5).
- Produces:
  - `Dialog` (Base UI `Dialog`):
    - `{ opened: boolean; onClose(): void; title: ReactNode; size?: 'sm'|'md'|'lg'|'xl' | number; closeLabel: string; withCloseButton?: boolean; children }`;
    - widths sm 380px, md 440px, lg 620px, xl 780px; `size` defaults to `'md'`;
    - header band: title font xl, weight 650, padding `--mmerp-space-md` vertical and `--mmerp-space-lg` horizontal, border-bottom `--mmerp-color-border`;
    - the body sets `--mmerp-control-height: var(--mmerp-control-height-lg)`;
    - backdrop `--mmerp-color-overlay-backdrop`, radius md, shadow md;
    - fade and scale over `--mmerp-motion-duration`.
  - `Drawer` (Base UI `Drawer`): `{ opened; onClose(); title: ReactNode; position?: 'left'|'right'; size?: number | string; closeLabel: string; children }`, title font lg, weight 650.
  - `Popover`: compound `Popover.Root`, `Popover.Trigger`, `Popover.Popup` (`{ side?: 'top'|'bottom'|'left'|'right'; align?: 'start'|'center'|'end'; width?: number | 'target' }`), radius lg, shadow md. The parts follow Base UI naming; `Popup` wraps Portal, Positioner and Popup.
  - `Menu`:
    - compound `Menu.Root`, `Menu.Trigger`, `Menu.Popup` (same `side`/`align`);
    - `Menu.Item` (`{ leftSection?; color?: 'danger'; disabled?; onClick? }`), `Menu.Separator`, `Menu.Label`;
    - radius lg; the highlighted item on gray `-light`; the danger item in `--mmerp-color-danger-light-color`.
  - `Tooltip` (Base UI `Tooltip`): `{ label: ReactNode; side?: 'top'|'bottom'|'left'|'right'; disabled?: boolean; children: ReactElement }`, radius sm, background gray 9, text white, font xs.
  - `ScrollArea` (Base UI `ScrollArea`): `{ children; className?; viewportRef?: Ref<HTMLDivElement> }`, scrollbar 8px, thumb gray step 4.

- [ ] **Step 1: Write the stories**
  - **Dialog:** sizes sm to xl, without close button, with a long body that scrolls, with a form inside.
  - **`play` on Dialog** (Review Focus 3):
    - click the trigger: the dialog opens with focus inside;
    - Tab cycles inside;
    - Escape calls `onClose` and focus is back on the trigger;
    - a `TextInput` inside is 40px tall.
  - **`play` on DialogWithMenu:** open the dialog, open a `Menu` inside it, press Escape. The menu closes and the dialog stays open (`onClose` not called).
  - **`play` on Menu:**
    - Enter on the trigger opens it with the first item highlighted;
    - ArrowDown moves; Enter fires the item's `onClick` and closes;
    - typeahead of `d` highlights `Delete`.
  - **`play` on Tooltip:** focusing the trigger shows a `role="tooltip"` element with the label.
  - **Drawer** left and right, **Popover** each side, **ScrollArea** overflow.

- [ ] **Step 2: Run and see them fail**

Run: `pnpm vitest run --project storybook src/primitives/overlay src/primitives/layout/ScrollArea.stories.tsx`
Expected: FAIL.

- [ ] **Step 3: Implement** to the interfaces above.

- [ ] **Step 4: Run tests, typecheck, lint**

Run: `pnpm vitest run --project storybook src/primitives && pnpm typecheck && pnpm lint`
Expected: PASS.

- [ ] **Step 5: Export and commit**

```bash
git add packages/ui/src/primitives/overlay packages/ui/src/primitives/layout/ScrollArea.* packages/ui/src/index.ts
git commit -m "feat(ui): dialog, drawer, popover, menu, tooltip and scroll area"
```

---

### Task 8: Navigation, shell and data display

**Files:**
- Create under `packages/ui/src/primitives/navigation/`: `Tabs`, `NavLink`, `Breadcrumbs`, `Pagination`, `Burger`, `TableOfContents` (`.tsx`, `.module.css`, `.stories.tsx`).
- Create under `packages/ui/src/primitives/layout/`: `AppShell` (same).
- Create under `packages/ui/src/primitives/data-display/`: `Table`, `Timeline` (same).
- Modify: `packages/ui/src/index.ts`

**Interfaces:**
- Consumes: `useMediaQuery` (Task 2), `UnstyledButton`, `ActionIcon` (Task 4), `RenderProp` (Task 3).
- Produces:
  - `Tabs`: Base UI `Tabs`. `Tabs.Root` takes `{ value?; defaultValue?; onValueChange?(v: string): void }`. Then `Tabs.List`, `Tabs.Tab` (`{ value: string }`), `Tabs.Panel` (`{ value: string }`). Tab weight 500, active indicator primary 6.
  - `NavLink`:
    - `{ label: ReactNode; leftSection?; rightSection?; description?; active?: boolean; render?: RenderProp; children?: ReactNode; defaultOpened?: boolean }`;
    - an `<a>` by default; with `children` it is a disclosure that shows nested links;
    - radius md, min-height `--mmerp-control-height`, label weight 500, single line with ellipsis;
    - the active link on primary `-light`.
  - `Breadcrumbs`: `{ 'aria-label': string; separator?: ReactNode; children }`. It renders `<nav><ol>`, the last item gets `aria-current="page"`, and the separator defaults to `/`.
  - `Pagination`:
    - `{ total: number; value: number; onChange(page: number): void; labels: { previous: string; next: string; page: (n: number) => string } }`;
    - pages collapse to `1 … 4 5 6 … 20` (siblings 1, boundaries 1);
    - round buttons with `data-mmerp-control`; previous and next are disabled at the ends.
  - `Burger`: `{ opened: boolean; onClick(): void; 'aria-label': string }`, with `aria-expanded`.
  - `TableOfContents`: `{ items: { id: string; label: ReactNode; depth?: 1|2|3 }[]; activeId?: string; 'aria-label': string }`. Anchor links `#id`; the active link gets `aria-current="location"`.
  - `AppShell`:
    - `{ header?: ReactNode; headerHeight?: number; navbar?: ReactNode; navbarWidth?: number; navbarOpened?: boolean; children }`;
    - defaults: `headerHeight` 48, `navbarWidth` 240;
    - CSS grid. Below `64em` the navbar is off-canvas and slides in when `navbarOpened`.
  - `Table`:
    - compound `Table` (`{ highlightOnHover?: boolean; stickyHeader?: boolean }`), `Table.Thead`, `Table.Tbody`, `Table.Tfoot`, `Table.Tr`, `Table.Th`, `Table.Td`;
    - rows 40px, header background `--mmerp-color-table-header`, header text xs weight 600 dimmed, rules `--mmerp-color-border`.
  - `Timeline`: `Timeline` and `Timeline.Item` (`{ bullet?: ReactNode; title?: ReactNode; active?: boolean; children? }`). The line is gray step 2; the active bullet and line are primary 6.

- [ ] **Step 1: Write the stories**
  - **`play` on Tabs:** ArrowRight moves focus and selects the next tab. The matching panel is visible and the others are hidden.
  - **`play` on NavLink:**
    - **Long** story (Review Focus 5): in a 200px navbar the label ellipsises and the link stays 200px;
    - **Nested**: clicking the parent shows its children.
  - **`play` on Pagination:**
    - `total=20 value=5` renders `1`, `4`, `5`, `6`, `20` and two ellipses;
    - at `value=20`, `next` is disabled;
    - clicking `page(6)` calls `onChange(6)`.
  - **`play` on AppShell:**
    - at viewport 800×600 the navbar is not visible; with `navbarOpened` it is;
    - at 1280×800 it is visible without the flag.
  - **Breadcrumbs, Burger, TableOfContents, Table, Timeline:** default stories, plus Table with `highlightOnHover` and `stickyHeader`.

- [ ] **Step 2: Run and see them fail**

Run: `pnpm vitest run --project storybook src/primitives/navigation src/primitives/layout/AppShell.stories.tsx src/primitives/data-display`
Expected: the new stories FAIL.

- [ ] **Step 3: Implement** to the interfaces above.

- [ ] **Step 4: Run tests, typecheck, lint**

Run: `pnpm vitest run --project storybook src/primitives && pnpm typecheck && pnpm lint`
Expected: PASS.

- [ ] **Step 5: Export and commit**

```bash
git add packages/ui/src/primitives/navigation packages/ui/src/primitives/layout/AppShell.* packages/ui/src/primitives/data-display packages/ui/src/index.ts
git commit -m "feat(ui): tabs, navigation, pagination, app shell, table and timeline"
```

---

### Task 9: Toast

**Files:**
- Create: `packages/ui/src/primitives/feedback/Toast.tsx`, `Toast.module.css`, `Toast.stories.tsx`
- Modify: `packages/ui/src/index.ts`

**Interfaces:**
- Consumes: `Button`, `ActionIcon` (Task 4).
- Produces:
  - `ToastOptions = { message: ReactNode; color?: 'success'|'danger'|'info'|'warning'; timeout?: number | null; action?: { label: string; onClick(): void } }`:
    - `timeout` defaults to 3000; `null` keeps the toast until closed;
    - clicking `action` runs `onClick` and closes the toast.
  - `toast`: `{ show(o: ToastOptions): string; close(id: string): void }`, a module-level manager built on Base UI `Toast.createToastManager()`, callable outside React.
  - `Toaster`: `{ closeLabel: string }`, mounted once at the app root:
    - renders Base UI `Toast.Provider` with the module manager and a viewport at bottom right;
    - each toast has `role="status"`, a colour bar from the ramp, and a close `ActionIcon` labelled `closeLabel`.

- [ ] **Step 1: Write the stories with `play`**, with `Toaster` in a decorator:
  - **Success:** clicking the trigger calls `toast.show({ message: 'Saved', timeout: 300 })`. A `role="status"` with `Saved` appears, then disappears within 1s (`expect.poll`).
  - **Sticky:** with `timeout: null` it is still present after 1s, and the close button removes it.
  - **Undo:** clicking the action button calls `onClick` (`fn()`) and removes the toast.
  - **Colours:** one toast per colour, kept open.

- [ ] **Step 2: Run and see them fail**

Run: `pnpm vitest run --project storybook src/primitives/feedback/Toast.stories.tsx`
Expected: FAIL.

- [ ] **Step 3: Implement** to the interfaces above.

- [ ] **Step 4: Run tests, typecheck, lint**

Run: `pnpm vitest run --project storybook src/primitives/feedback && pnpm typecheck && pnpm lint`
Expected: PASS.

- [ ] **Step 5: Export and commit**

```bash
git add packages/ui/src/primitives/feedback/Toast.* packages/ui/src/index.ts
git commit -m "feat(ui): toasts with a module-level manager"
```

---

### Task 10: Dates

**Files:**
- Create under `packages/ui/src/primitives/dates/`: `calendarGrid.ts`, `calendarGrid.test.ts`, `Calendar`, `MonthPicker`, `DateInput`, `MonthPickerInput` (`.tsx`, `.module.css`, `.stories.tsx`).
- Modify: `packages/ui/src/index.ts`

**Interfaces:**
- Consumes: `TextInput`, `FieldProps` (Task 5), `Popover` (Task 7), `ActionIcon` (Task 4).
- Produces. Dates are ISO strings: a day is `YYYY-MM-DD`, a month is `YYYY-MM`. All arithmetic uses UTC `Date`.
  - `monthGrid(year: number, month: number, firstDayOfWeek: 0|1|2|3|4|5|6): string[]`: always 42 ISO days, starting on `firstDayOfWeek` on or before the 1st (`month` is 1–12).
  - `addDays(iso: string, n: number): string`, `addMonths(iso: string, n: number): string`. `addMonths` clamps to the month's last day.
  - `CalendarLocale = { weekdays: string[]; months: string[]; firstDayOfWeek?: 0|…|6 }`:
    - 7 short names, Sunday first; 12 names;
    - `firstDayOfWeek` defaults to 1, which the caller passes.
  - `Calendar`:
    - `{ value: string | null; onChange(iso: string): void; defaultMonth?: string; minDate?: string; maxDate?: string; locale: CalendarLocale; labels: { previousMonth: string; nextMonth: string }; today?: string }`;
    - `today` defaults to the browser's local date;
    - a `role="grid"` with the days as `gridcell` buttons, roving tabindex, `aria-selected` on the value, `aria-current="date"` on today, disabled outside min/max;
    - keys: arrows move ±1/±7 days, PageUp/PageDown ±1 month, Home/End to the week's start or end, Enter or Space chooses.
  - `MonthPicker`: `{ value: string | null; onChange(month: string): void; minDate?; maxDate?: string; locale: CalendarLocale; labels: { previousYear: string; nextYear: string } }`, a 3×4 grid of months with the same roving keyboard model (arrows ±1/±3).
  - `DateInput`:
    - `FieldProps & { value: string | null; onChange(iso: string | null): void; parse(text: string): string | null; format(iso: string): string; locale: CalendarLocale; labels: { previousMonth: string; nextMonth: string }; minDate?; maxDate?; placeholder?: string; leftSection?: ReactNode; clearable?: boolean; clearLabel?: string }`;
    - the text is free; on blur or Enter it is parsed:
      - a valid result inside min/max calls `onChange(iso)` and the text becomes `format(iso)`;
      - empty text with `clearable` calls `onChange(null)`;
      - anything else reverts the text to `format(value)` (or empty) without calling `onChange`;
    - clicking the input or pressing ArrowDown opens the calendar in a `Popover`;
    - choosing a day calls `onChange`, closes, and returns focus to the input.
  - `MonthPickerInput`: `FieldProps & { value: string | null; onChange(month: string | null): void; format(month: string): string; locale; labels: { previousYear; nextYear }; minDate?; maxDate?; placeholder?; clearable?; clearLabel? }`. A read-only trigger showing `format(value)` opens `MonthPicker` in a `Popover`.

- [ ] **Step 1: Write the failing `calendarGrid.test.ts`**

```ts
import { describe, expect, it } from 'vitest'
import { addDays, addMonths, monthGrid } from './calendarGrid'

describe('monthGrid', () => {
  it('starts on the chosen weekday and spans 42 days', () => {
    const g = monthGrid(2026, 10, 1) // 1 Oct 2026 is a Thursday
    expect(g).toHaveLength(42)
    expect(g[0]).toBe('2026-09-28')
    expect(g[3]).toBe('2026-10-01')
    expect(g[41]).toBe('2026-11-08')
  })
  it('starts on the 1st when it falls on the first weekday', () => {
    expect(monthGrid(2026, 2, 0)[0]).toBe('2026-02-01') // a Sunday
  })
})

describe('date arithmetic', () => {
  it('crosses month and year ends', () => {
    expect(addDays('2026-12-31', 1)).toBe('2027-01-01')
    expect(addMonths('2026-01-31', 1)).toBe('2026-02-28')
  })
})
```

Run: `pnpm vitest run --project unit src/primitives/dates`. Expected: FAIL. Implement `calendarGrid.ts`, run again; expected PASS.

- [ ] **Step 2: Write the stories.** The story file defines sample `parse` and `format` for `dd/mm/yyyy` and `mm/yyyy`, and a Vietnamese sample locale.
  - **Calendar `play`** with `today='2026-10-20'` and value `2026-10-20`:
    - focus is on the 20th;
    - ArrowRight then Enter calls `onChange('2026-10-21')`;
    - PageDown shows November 2026;
    - with `minDate='2026-10-10'`, the 9th is disabled.
  - **DateInput `play`** (Review Focus 2):
    - typing `05/01/2026` then Enter calls `onChange('2026-01-05')`, and the input reads `05/01/2026`;
    - starting at value `2026-01-05`, typing `31/02/2026` then Tab does not call `onChange`, and the input reads `05/01/2026` again;
    - ArrowDown opens the calendar; choosing a day closes it and focus is back on the input.
  - **MonthPicker / MonthPickerInput `play`:** ArrowRight then Enter from `2026-10` chooses `2026-11`.
  - **States for the inputs:** empty, filled, disabled, read-only, error, clearable.

- [ ] **Step 3: Run and see the stories fail**

Run: `pnpm vitest run --project storybook src/primitives/dates`
Expected: FAIL.

- [ ] **Step 4: Implement** to the interfaces above.

- [ ] **Step 5: Run tests, typecheck, lint**

Run: `pnpm vitest run src/primitives/dates && pnpm typecheck && pnpm lint`
Expected: PASS.

- [ ] **Step 6: Export and commit**

```bash
git add packages/ui/src/primitives/dates packages/ui/src/index.ts
git commit -m "feat(ui): calendar, month picker and date inputs"
```

---

### Task 11: Phase 1 completion check

**Files:**
- Modify: `docs/superpowers/specs/2026-10-20-own-ui-kit-design.md` (Phase 1: add a line `Phase 1 built on <date>; see packages/ui.`)

- [ ] **Step 1: Inventory check.** Check every row of the spec's parity inventory against `src/index.ts`. `Flex` folds into `Group`, `Input`/`InputWrapper` into `TextInput`/`Field`, `FileButton` into `FileInput`, `Notification` into `toast`. Anything missing gets a story and an implementation in its folder, committed as `feat(ui): <name>` before Step 2.

- [ ] **Step 2: Full run, exactly as CI does it**

Run: `cd packages/ui && pnpm install --frozen-lockfile && pnpm gen && git diff --exit-code -- src/tokens/tokens.css && pnpm typecheck && pnpm lint && pnpm test && pnpm build-storybook`
Expected: every step exits 0, and story tests report no a11y violations.

- [ ] **Step 3: Isolation check**

Run: `git diff --stat main...HEAD -- . ':!packages/ui' ':!.github/workflows/ui.yml' ':!docs/superpowers'`
Expected: only `docs/adr/0037-own-ui-kit-on-base-ui.md` (from the design commit). No `web/`, `Makefile`, `Dockerfile` or `ci.yml` change.

- [ ] **Step 4: Push and watch the `UI kit` workflow**

Run: `git push -u github feat/own-ui-kit`, then `gh run watch` on the `UI kit` run.
Expected: green. Pushing is outward-facing: confirm with the owner first.

- [ ] **Step 5: Commit the spec note**

```bash
git add docs/superpowers/specs/2026-10-20-own-ui-kit-design.md
git commit -m "docs(ui): mark phase 1 of the UI kit built"
```

Then ask the owner to run `pnpm storybook` in `packages/ui` and look through it. Their look is the last condition for phase 1 to be done.
