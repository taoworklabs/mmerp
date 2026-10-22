# 0037. The frontend uses its own UI kit, `@mmerp/ui`, built on Base UI; Mantine is removed

Status: proposed · 2026-10-20 · supersedes the "Mantine" part of [ADR-0008](./0008-tech-stack.md)

Amended 2026-10-21: layout blocks move into the kit (see docs/superpowers/specs/2026-10-21-ui-kit-layout-blocks-design.md).

## Context

ADR-0008 chose React, TypeScript and Mantine for the frontend. Since then mmerp has grown its own design language: colour ramps, a type scale, the 30px control, radius per role, and `shared/ui` components and page templates that areas must use. Mantine is now mostly a styling engine that this language has to be translated into. `theme.ts` overrides Mantine's defaults component by component; CSS modules read `--mantine-*` variables; screens use Mantine style props. Four Mantine packages are imported by 123 files.

The owner wants fewer third-party dependencies. What mmerp actually needs from a library is behaviour that is hard to get right: focus management, keyboard navigation, aria wiring, positioning of popups. It does not need the styling. Base UI (`@base-ui/react`, from the Radix, Floating UI and Material UI teams) ships exactly that: unstyled, accessible primitives with no styling engine. Twenty builds its UI library, `twenty-ui`, the same way.

## Decision

- **`@mmerp/ui`** is a private package at `packages/ui` in the repo root, consumed from TypeScript source with no build step. It holds the design tokens (one TypeScript source, emitted as `--mmerp-*` CSS custom properties), the primitives, and a few React hooks.
- **Primitives wrap Base UI** where Base UI has a matching part, and are plain HTML and CSS otherwise. Styling is CSS modules reading the token variables. No Tailwind, Sass or CSS-in-JS.
- **`@mmerp/ui` knows nothing of mmerp**: no i18n, router, API, form library or domain term, and no user-facing text.
- **A component belongs in `@mmerp/ui` when it only looks and lays out; it belongs in `shared/ui` when it behaves.** Look and layout means: every text comes in as a prop, every link renders through a `render` prop or an `href` rather than a router, transient UI state and browser APIs are allowed, and there is no fetching, persistence, form state or navigation. Behaviour — translations, routing, React Hook Form wiring, queries, saved state, confirmations and notifications — stays in `shared/ui`, built on `@mmerp/ui`. Once a block lives in the kit, its look lives only there: one look, one home.
- **Areas import look-only blocks straight from `@mmerp/ui`; blocks whose look needs an mmerp label or route go through `shared/ui`** instead, which fills them in and is what the area imports.
- **Only `@mmerp/ui` imports `@base-ui/react`.** Areas import from `@mmerp/ui` only the layout and text primitives they may take from Mantine today; everything else comes through `shared/ui`.
- **Mantine is removed entirely**: `@mantine/core`, `dates`, `hooks` and `notifications`, its theme and provider, its MCP server and its vendored agent skills. Date input, which Base UI does not provide, is built in `@mmerp/ui`.

## Consequences

- One runtime UI dependency (`@base-ui/react`) instead of four. Mantine major upgrades no longer ripple through every screen.
- mmerp owns the code of every primitive: about 50 small components and a calendar. It maintains them and their styles itself. Accepted, because Base UI carries the hard part (accessibility and interaction) and the styling was already ours.
- The package is built first, apart from the app, which keeps Mantine until it is complete; only then does the swap touch most frontend files once, in tickets that each keep the app working.
- Swapping the primitive layer again later touches only `packages/ui`, because nothing else imports Base UI.
- Agents lose Mantine's docs and skills as a reference. The package's Storybook (with a11y and Vitest addons) becomes the reference for primitives, their props and states; `/dev/ui` and the `mmerp-ui` skill keep mmerp's own components and rules.
- The kit grows by the layout blocks, not just primitives; Storybook can show whole screens built only from kit exports.
