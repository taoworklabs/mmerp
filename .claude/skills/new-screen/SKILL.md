---
name: new-screen
description: Build or change a frontend screen in web/src. Use when adding a page, form, list, or dialog to an area or to core.
---

# New screen

The rules live in `ui.md` and `frontend.md`; this skill is the order to apply them in. Each step ends on its completion criterion.

1. **Read the rules.** Read `ui.md` end to end, then `frontend.md` → "Cache" and "Permissions: the frontend computes nothing". For a specific UX question (a form pattern, a table behaviour, an accessibility outcome), consult the `ui-ux-pro-max` skill when it is installed; `ui.md` wins where they differ.
   Done when: you have named the page template from `ui.md` → "Page templates" that this screen uses.

2. **Assemble.** Build the screen from that `shared/ui/page` template and `shared/ui` primitives. Business components live in the area; a missing generic primitive goes into `shared/ui` with an entry on `/dev/ui` (`ui.md` → "Where components live").
   Wire it in: add the route to the area's `routes.tsx`, gated with `useCan(…) ? <Page /> : <ForbiddenPage />` when it needs a permission, and a `nav` entry in the area's `index.ts` (label, path, icon, group, permission matching the route) when it is a menu destination.
   A `DocumentPage` gets its discussion and, for types that take attachments, its attachments as `sections` (`useDiscussionSection`, `useAttachmentSection` from `shared/document`); a `RecordPage` shows the same two as tabs.
   Done when: the screen's layout comes entirely from the template, with every colour, size and spacing taken from theme tokens.

3. **Data.** Query keys come from the area's `keys.ts` factory and include every filter, sort, page and period parameter. Document writes go through `useDocumentMutation`. Filters, sort, page and tab live in the URL. Record buttons come from the record's `allowed_actions`; collection actions (create, import, export) from `GET /<product>/<x>/actions` under `<product>Keys.<x>.actions()` (see `hrm/hooks/useEmployeeActions.ts`), and a create page without `create` shows `<ForbiddenPage>`. A detail page resolves its query with `loaded()` (`hrm/components/loaded.tsx`). API types come only from the generated schema through the area's client; never redeclare an API type by hand.
   Done when: copying the URL reopens the same view, and the screen derives no permission from status or role.

4. **Text.** Every string is an i18n key with both `vi` and `en` values, using `CONTEXT.md` terms; numbers, money and dates go through `shared/i18n` formatters.
   Done when: `make lint` passes its literal-string and translation-key checks.

5. **States.** Handle loading, empty, error, forbidden, not found and read-only as listed in `ui.md`.
   Done when: each state is reachable and renders the `shared/ui` component for it.

6. **Verify.** Run the checklist at the end of `ui.md` item by item. Check the screen at 1280px and 1024px, and at 375px for self-service flows. For flows where a mistake has serious consequences, add a Playwright test (`techstack.md` → Testing and quality).
   Done when: every checklist item holds and `make lint test` is green.
