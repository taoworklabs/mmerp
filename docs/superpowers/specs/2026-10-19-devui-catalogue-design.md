# /dev/ui catalogue: layered navigation and a fixed page anatomy

Date: 2026-10-19. Status: draft for approval.

## Why

`/dev/ui` (`web/src/app/devui`) is the design-system catalogue: every foundation, `shared/ui` component and page template, rendered live in the browser with no backend. Today its sidebar has ten purpose groups (Foundations, Actions, Navigation, Display, Tables, Input, Feedback, Overlays, Blocks, Layout). They mix two different kinds of thing:

- plain Mantine primitives (`Button`, `ActionIcon`, `Badge`, `Alert`, `Tabs`, `Tooltip`, `Anchor`…), some of which areas may not even import;
- components mmerp built and owns (`shared/ui`, `shared/document`: `DataTable`, `FieldList`, `StatusBadge`, form fields, states, page templates…).

The owner finds the sidebar confusing and the pages uneven. Nothing makes clear which pieces are ready-made in `shared/ui`, and the Mantine basics look as if they had to go through `shared/ui` too. Pages also differ in what they show: only the page templates have a "use when / avoid when" note.

The new structure follows established design-system sites: GOV.UK (Styles / Components / Patterns), Carbon and Primer (components kept apart from patterns), SAP Fiori (floorplans as a layer of their own), Backstage ("use our components first, fall back to MUI"), and Nathan Curtis's component-page anatomy.

Readers are the owner and Claude, both building screens. A reader should be able to answer two questions quickly: "is there a ready-made piece for this?" and "how do I use it here?"

## Goals

1. A sidebar in six layers, ordered from the base up (decided, option A of the mockups):
   1. **Start here**: one overview page that explains the layers, says "Mantine directly vs mmerp components", and says where the full rules live (the `mmerp-ui` skill, `ui.md`).
   2. **Foundations**: principles, colour, typography, spacing, surfaces, icons (today's pages).
   3. **Mantine, used directly**: exactly the primitives areas may import from `@mantine/core`, per the allowlist in `web/eslint.config.js` (today `Anchor`, `Button`, `Divider`, `Group`, `List`, `Loader`, `Skeleton`, `Stack`, `Text`, `Tooltip`). Each page shows the mmerp conventions and links to the Mantine docs. The layout and text primitives (`Group`, `Stack`, `Divider`, `List`, `Text`) may share one page. Mantine components outside the allowlist (`Badge`, `Alert`, `Menu`, `ActionIcon`, `Tabs`…) get no page of their own; they appear only inside the mmerp components built on them. If the allowlist changes, layer 3 follows it.
   4. **mmerp components**: everything ready-made in `shared/ui` and `shared/document` (and `shared/jobs`/`shared/org` pieces listed in `ui.md`, if they render without a backend). This is the layer to make prominent.
   5. **Patterns**: how components combine to solve a task: filter and search, create/edit, confirm and delete, empty/loading/error states, bulk actions, document approval.
   6. **Page templates**: `Page`, `ListPage`, `RecordPage`, `DocumentPage`, `InboxPage`, `AuthPage`.
2. Every page in layers 3 and 4 follows one anatomy, in this order. A section with nothing to say is left out:
   1. Header: name, a **source tag** (`Mantine · used directly` or `mmerp · <import path>`), a one-line description.
   2. Example: the main live example.
   3. Use when / Don't use when: one line each. "Don't use" names the alternative and links to it.
   4. Variants and states: one labelled `Case` each (today's framing).
   5. Do / Don't: paired live examples with a green or red top border.
   6. Code: for mmerp, the import line and the main props. For Mantine, the props convention (for example `variant="subtle" color="gray"` for a quiet button).
   7. Related: other components, and the patterns and templates that use it.

   An "on this page" table of contents sits beside the content on wide screens.
3. Pattern and template pages use a shorter anatomy: the problem → a live example → the components it uses (linked) → what to avoid. Each template page lists the mmerp components it is built from (borrowed from mockup option C).

## Non-goals

- `/dev/ui` does not become the full rulebook. **The `mmerp-ui` skill and `ui.md` stay the single source of the rules.** The catalogue keeps what has to be *seen*: live examples, variants and states, visual Do/Don't pairs, a one-line use/avoid, the import and links. It has no "UI text" or "Accessibility" prose sections; the overview page points to the skill for those. (The owner decided this to avoid two copies drifting apart and doubling the vi/en text.)
- No new components, no restyling of existing components, no change to areas.
- No search box, no thumbnails grid, no props tables generated from types. Add them when the catalogue grows past easy scanning.
- No production exposure: the catalogue stays dev-build only and offline (`app/devui/api.ts`).

## Anti-goals (would make this a failure even if it works)

- A reader still cannot tell at a glance whether something is ready-made in mmerp or plain Mantine.
- The same component documented under two layers, or a Mantine page implying it should be wrapped.
- Rule text copied from the skill into i18n strings, so the two start disagreeing.

## Constraints

- Everything stays in `web/src/app/devui`. Labels and text go through i18n (`shared.devui.*`, both `vi` and `en`). The lint rules apply: no literal strings, no colour literals, theme tokens only. Component names (`DataTable`, `Button`) may be shown as-is, as the template entries already are (`Entry.name`).
- The catalogue uses `shared/ui` primitives and its own `DevUi.module.css`; the Do/Don't borders use theme colour tokens (`success`/`danger`), not literals.
- `make lint` and `make test` pass. Update `ui.md` (line 9, which describes the page), and the `mmerp-ui` skill's `SKILL.md` line 14 if its wording no longer fits.

## Decided

- Six layers in the order above. Option B (keep the purpose groups and tag each entry) and option C (templates first) were rejected. Only C's "template → components it uses" idea is kept.
- The anatomy above, the same order for mmerp and Mantine pages, with "Code" as the only section that differs.
- The split of roles between the catalogue and the skill, as stated under Non-goals.

## Builder's calls (made during design; check them)

- **Order inside a layer: alphabetical, no subgroups.** Each layer has about 10–25 entries. Revisit if layer 4 gets hard to scan; the first fallback is subgroups taken from `ui.md`'s list (Actions and feedback, States, Display, Forms).
- **Page granularity:** one page per component, except families that `ui.md` already treats as one (form fields, filters, states, tiles, toasts/notify), which get one page each, with one `Case` per member.
- **Mixed pages are split by layer.** Today's Badges page becomes mmerp `StatusBadge` (the count badge is shown where it is used), Menus becomes `MoreActions`/`MenuAction`, Avatars becomes `PersonAvatar`, and Alerts becomes `Notice`. Today's Icon buttons, Tabs and Overlays pages show Mantine outside the allowlist. Their material moves to the mmerp component or template that uses it (for example `RecordPage` tabs, the `Tooltip` page) or is dropped. Today's `Bulk`, `States`/`QueryView`, `use-action`/`Dialogs`, `Filters` and `Inline` material is reused for both the component pages and the pattern pages, without duplicating examples: a pattern links to the component's page and shows only the composed flow.
- **Each sidebar layer heading shows its number and a short note** (for example "Mantine, used directly — no wrapper"), and each entry can show its source tag, so the distinction is visible before a page opens.
- **Do/Don't examples are built from code that passes lint.** A Don't shows a legal mistake, such as two primary buttons, a wrong variant, or a missing empty state. A mistake lint would block (a colour literal, a hand-written table in an area) is shown as a one-line text Don't. There are no lint exceptions in `app/devui`.
- **Template pages keep today's full-page render.** The strip above the template (today's `Usage`) grows to hold the problem, use/avoid and the "built from" links to mmerp components. Template pages have no table of contents and are not nested in a catalogue frame.
- **The "on this page" table of contents is built from the sections a page renders**, not hand-written, so it cannot go stale.
- **Paths change** (for example `/dev/ui/mmerp/data-table`, `/dev/ui/mantine/button`). Nothing outside `app/devui` links to individual entries (checked: only `bootstrap.tsx` mounts the router), so old paths need no redirects beyond the existing catch-all.

## Left to the builder

- The exact entry list per layer: take it from `ui.md` "Where components live" and the current exports, and list any `shared/ui` export left out, with the reason.
- The shared building blocks for the anatomy (a page frame that renders the header, the source tag, the sections and the table of contents, plus `DoDont`), kept as small as the current `Demo`/`Case`.
- Which patterns get a page first: the six named above are the target. Any that the current examples cannot show live may come later and be noted as missing.
- Wording of the vi/en strings, following the skill's `content.md`.

## Mockups

`.superpowers/brainstorm/1908033-1792398421/content/sidebar.html` (sidebar options; A chosen) and `page-anatomy.html` (the page anatomy, mmerp `DataTable` vs Mantine `Button`). The "UI text" and "Accessibility" sections in that mockup were later dropped (see Non-goals).
