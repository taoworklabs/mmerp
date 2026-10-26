---
name: mmerp-ui
description: mmerp's design system: tokens, components, layouts, forms, tables, actions, feedback, UI text, review checklist. Use for any change in web/src that touches a screen, a component, styling, layout, a form or table, or how a user flow behaves (create, edit, confirm, bulk actions, errors).
---

# mmerp UI

The visual language and interaction rules of every screen in `web/src`. Values live in code; this skill says which value, component and behaviour to use where, and why.

Sources of truth in code:

- `web/src/shared/ui/theme.ts`: colour scales, type, spacing, radius, shadows, component defaults, the `icon` and `filterWidth` constants, and the semantic CSS variables (`cssVariablesResolver`).
- `web/src/shared/ui/global.css`: control heights and the 44px touch target below 1024px.
- `web/src/shared/ui/` and `web/src/shared/document/`: the components; `/dev/ui` (dev build) shows, in six layers, the foundations (read live from the theme), the Mantine primitives areas import directly (`web/eslint.config.js` allowlist), the mmerp components, patterns and page templates, each component with a live example, its states, do and don't, and its import. A new component gets an entry in `web/src/app/devui/catalogue.ts`.
- `docs/ui.md`: the written rules of the app. This skill and `docs/ui.md` must agree; when you change a rule, change both.

## Principles

1. **Dense and calm.** Users work at a desk all day: 14px text on 20px lines, 30px controls, 40px table rows. Space comes from the 4px scale, separation from thin borders, never from decoration.
2. **Flat until it floats.** Pages are flat white surfaces divided by borders; only overlays (menus, popovers, modals, toasts) cast a shadow, and a page form's footer while the form scrolls under it.
3. **Colour means something.** Primary blue marks action and selection; green, orange, red, blue and purple mark state. Neutral grey carries everything else. A colour always comes with text or an icon.
4. **Radius follows role.** 4px for tags, 6px for controls, frames and modals, 8px for menus, round for people and page numbers.
5. **One way per job.** Two screens doing the same job look the same, because both use the same `shared/ui` component.
6. **The backend decides, the UI displays.** Buttons come from `allowed_actions`, messages from error codes.

## UX principles

1. **A record has one page.** Lists lead to it; a side panel only previews; viewing and editing are the same page (`RecordPage` for a record, `DocumentPage` for a document): each group of values is a card edited in place, with its own Save, showing every value its form edits.
2. **One primary action per place**, named by its outcome; the rest are secondary or behind "…".
3. **Ask before what cannot be undone; offer Undo for what can.** Lifecycle actions confirm and never undo; reversible side-effect-free changes apply at once with Undo.
4. **Errors stay next to their cause**; successes toast; nothing fails silently.
5. **The URL is the state.** Filters, sort, page, tab, the open side panel and saved views are all URLs; Back always works.
6. **The server computes.** Totals, numbers and statuses come from the API; the screen never recomputes money.

## Rules

- **Tokens only.** Colour, font size, spacing, radius and shadow come from theme keys (`gap="xs"`, `c="dimmed"`, `color="danger"`, `radius="lg"`) or theme CSS variables (`var(--mantine-color-default-border)`, `var(--mmerp-elevation-surface-sunken)`). A pixel number or colour code is written only in `theme.ts`, and a size specific to one component (row height, column width) only in that component's CSS module in `shared/ui`.
- **Reuse first.** Look in the area's own `pages/` and `components/` for a screen or piece that already does the job (a list of the same records, a progress badge), then build from a `shared/ui/page` template and the existing components; `references/components.md` lists them.
- **New components follow `references/components.md`**: built from theme tokens, in `shared/ui` with a CSS module when needed, every state covered, shown on `/dev/ui`.
- **Areas carry no styling.** No `style` attribute, no CSS file, no colour code in an area: lint (`make lint`) rejects all three.

## Which reference to read

| Task | Read, in order |
| --- | --- |
| New or changed list screen (table, filters, bulk actions, side panel) | `layout.md`, `tables.md`, `actions.md`, `feedback.md`, then `patterns.md` for the recipe |
| Create or edit form, a document, document lines | `layout.md`, `forms.md`, `actions.md`, `feedback.md`, then `patterns.md` |
| Detail screen (document or record page, editable cards, inline edit) | `layout.md`, `forms.md`, `actions.md`, then `patterns.md` |
| Settings list, dashboard, report, inbox | `layout.md`, `patterns.md`, then the part it uses (`tables.md`, `forms.md`) |
| A confirmation, undo, status change, shortcut | `actions.md` |
| Loading, empty, error or success messages | `feedback.md` |
| Style of a component, a colour, size, space, radius, icon | `tokens.md`, `components.md` |
| Use, change or add a shared component | `components.md` |
| Any visible text (label, button, dialog, toast, error) | `content.md` |
| Finish any UI change | `checklist.md` (always, last) |

This skill covers how a screen looks and behaves; how it gets data, permissions and routes is the `new-screen` skill. Read a component's props from its source file: the skill names the component, the file is the signature.
