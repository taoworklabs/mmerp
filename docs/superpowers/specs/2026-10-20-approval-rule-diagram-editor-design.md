# Approval rule editor as a diagram: design

Status: draft for review · 2026-10-20

## Intent

The approval rule editor (`shared/document/ApprovalRules.tsx` and `flowEditor/`) shows the flow as nested form cards. Every step is a full form (approver kind, role or users, approvals, rejections, due time), always open, so one step fills the screen. Splits add panels inside panels. The person configuring a rule cannot see the shape of the flow, cannot tell who approves what without reading every field, and does not know where to start. The owner's words: "standing where the user stands, I don't know what I'm supposed to do."

The people configuring rules are low-code users: they don't write code, but branches and conditions don't scare them. They need to see the whole flow as a picture and edit one piece at a time.

The goal is to replace the card editor with a **diagram on the left and a side panel on the right**: the flow drawn as boxes and arrows, each box saying who approves in plain words; clicking a box opens its form in the panel; a ⊕ on every arrow inserts a step or split right there. Nothing changes in the rule model, the API or the backend.

A throwaway spike of this design runs in `/dev/ui/spike/approval-flow` (`web/src/app/devui/spike/`). The owner reviewed it and approved the direction. It shows the intended look and behaviour. It is not the code to keep: it hardcodes Vietnamese text, sample data and its own validation.

## Goals

1. **The whole flow is visible as a diagram.** Start box ("Gửi duyệt"), the elements top to bottom, end box ("Có hiệu lực"; "Có hiệu lực ngay" when the flow is empty). A split draws its branches side by side, joined by connector lines above and below; nested splits nest the same way.
2. **Each box reads on its own.** A step box shows who approves ("Quản lý trực tiếp", the role name, or up to two user names then "+n") and a dimmed detail line ("2/3 người đồng ý", "hạn 24 giờ") when those differ from the defaults. A branch shows its condition as a chip in words ("Nếu Số ngày nghỉ > 3", "Khi Đơn vị thuộc Kế toán", "Còn lại", "Luôn chạy"). An empty branch body shows a dashed "Không cần duyệt thêm" box.
3. **Edit one thing at a time in a side panel.** Selecting a step, a split, a branch or the otherwise branch shows that element's form in the panel. With nothing selected, the panel says how to use the editor. Selection is visible on the diagram.
4. **Insert where you point.** A ⊕ on every connector opens a menu: *Bước duyệt*, *Tùy trường hợp*, *Cùng lúc*, each with a one-line explanation. The new element is inserted at that spot and selected.
5. **Problems show on the diagram.** An element with a problem (the full list is under "Calls I made") is drawn in red with the problem in its detail line, before saving, and Save takes the user to it. A save refused by the server (`invalid_rule` with a `path`) marks and selects the element at that path, with the message in the panel.
6. **Plainer words.** The UI text changes to the low-code user's words (see "UI text" below). Rarely changed fields move under *Nâng cao*. The fallback role and level limit move to a *Cài đặt chung* view.
7. **A new rule is obvious.** Empty flow: start box, a large ⊕, the end box, and one line: "Chưa có ai duyệt. Bấm dấu + để thêm người duyệt đầu tiên."

## Non-goals

- Drag and drop, free positioning, zoom or pan. The flow is a tree; order changes use *Chuyển lên/xuống* (steps) and *Sang trái/phải* (branches). Zoom can be added later if wide flows become a real problem.
- Any change to the rule model, API, validation on the server, or the approval engine.
- Changing the rule list page (`RuleList`).
- A diagram library or `<canvas>`. The diagram is HTML and CSS (flexbox columns and rows, connector lines as borders), so Mantine inputs, keyboard focus, screen readers, text wrapping and tests work as everywhere else.

## Anti-goals

- A simple rule ("the direct manager approves") becoming harder to configure than today.
- Losing anything the card editor can express today: every approver kind, both counts, due time, nested splits up to the server's depth, nested condition groups, the otherwise branch, read-only mode.
- Losing the form behaviour `ui.md` requires: unsaved-changes guard on leaving, sticky `FormActions` footer with the unsaved indicator and *Bỏ thay đổi*, Ctrl+S, the "Còn N trường cần sửa" line.

## Constraints

- `ui.md`, `frontend.md` and the `mmerp-ui` skill apply. Finish with `ui.md`'s checklist. The editor stays in `shared/document` (it's shared by every product's area) and builds only from `shared/ui` primitives and Mantine. Colours come from theme tokens (the colour-literal scan runs in `make lint`).
- All text goes through i18n in `vi` and `en`. No literal strings in components.
- Form state stays in React Hook Form with the existing `flowForm.ts` model (`ElementValues`, `toFlowValues`, `fromFlowValues`, `formPathOf`). The wire format doesn't change.
- `allowed_actions` decides read-only. In read-only mode the diagram shows without ⊕ and the panel's fields are read-only.
- The server's limits stay enforced in the UI where the card editor enforces them today: counts 1–20, due time 1–720 hours, users per step ≤ 20, split depth ≤ 5 (the ⊕ menu hides the split items at depth 5), condition group depth ≤ 3.
- New or changed domain terms go into `CONTEXT.md` in the same change.

## Decided in conversation

- Direction B from three mockups: diagram plus side panel (not a compact inline list, not an outline plus form).
- Built in HTML/CSS, not canvas; no drag and drop.
- The spike's look and behaviour are approved as the target.

## Calls I made (the owner can overrule)

- **Problems block Save, checked over the whole flow.** Only one element's form is mounted at a time, so field-level React Hook Form rules can't see the others. A pure validator walks the whole form value and returns each problem with its element path. It covers missing role or users, approvals above the number of users, counts outside 1–20, due time outside 1–720, an unfinished condition leaf (a chip showing "…"), an empty top-level flow and a missing fallback role. Problem boxes and chips are red on the diagram as soon as the problem exists. Save with problems doesn't call the server. Instead it selects the first problem (or opens *Cài đặt chung* for the fallback role) and shows "Còn N chỗ cần sửa · Đến chỗ tiếp theo" next to Save. That link cycles through the problem elements by selecting them, not by focusing inputs. This is a variant of the shared `Form`/`FormActions` error line, made for this editor. How it plugs in (an option on `Form`/`FormActions` or a wrapper in `flowEditor`) is left to the builder, but the shared forms' behaviour must not change. The panel's own fields still show their field errors when touched. The server stays the backstop (goal 5).
- **A new rule starts empty, but an empty flow can't be saved.** The empty state (goal 7) is what a rule without a saved flow opens with (`toValues` stops seeding one step). Saving still needs at least one step, as today (`flow_required`). "No approval" stays expressed by deleting the rule (*Xóa quy tắc*), so an empty saved rule and a deleted rule never mean two different things.
- **Labels say what the engine does, not what's shortest.** Module approvers are chosen per document type: for leave and overtime that's the manager of the employee the request is for, not the person who submitted it. Role approvers are looked up at the document's org unit. So the kinds read *Quản lý trực tiếp (theo chứng từ)* with a hint naming what the type uses where one exists, *Người có vai trò, tại đơn vị của chứng từ*, and *Những người cụ thể*. The start box reads *Gửi duyệt*, not "Người gửi đơn". The max-levels text reads "Khi quản lý trực tiếp lại chính là người gửi, đi lên tối đa (cấp)". These texts override the approver rows of the UI text table below.
- **The page is full width.** Diagram plus aside doesn't fit the 1200px form page, so the editor page uses `fullWidth`. `ui.md` (and the `mmerp-ui` skill, which must agree with it) gains this screen next to the wide-table screens that may go full width.
- **Save stays in the sticky `FormActions` footer**, not in the page header as in the spike, because `ui.md` puts a page form's Save there and the footer carries the unsaved indicator. The page header holds *Cài đặt chung* (secondary) and *Xóa quy tắc* (danger, as today).
- **Settings view:** *Cài đặt chung* selects a "settings" view in the side panel holding the fallback role and the max levels, with the hint texts below. If the server rejects the fallback role, the settings view opens with the error.
- **Panel on narrow screens:** from `md` up the panel sits beside the diagram, sticky, at `--mmerp-aside-width`. Below `md` the selected element's form opens in a bottom or right `Drawer`, and the diagram takes the full width.
- **Wide diagrams scroll horizontally** inside the diagram frame. The frame is as tall as the window allows and scrolls inside itself.
- **The condition editor in the panel stacks vertically:** field on one row, comparison and value on the next. The panel is too narrow for the current three-column row. The condition editor's behaviour (all/any, groups, remove, field-dependent value inputs, searched values) stays.
- **Step panel shows "Cần bao nhiêu người đồng ý" only when it means something:** for *Những người cụ thể* with more than one user, and for role and module approvers, where the number of holders isn't known in advance. *Bao nhiêu người từ chối thì trả đơn* goes under *Nâng cao*, collapsed. If a saved rule already has a non-default rejection count, *Nâng cao* starts open.
- **Glossary:** the Vietnamese UI names of the two split kinds change to *Tùy trường hợp* (choose-one split) and *Cùng lúc* (parallel split). `CONTEXT.md` updates their `vi:` names in the same change; the English terms stay.
- **Accessibility:** every box and chip is a button with an accessible name built from its visible text (e.g. "Bước duyệt: Trưởng phòng nhân sự"). Each ⊕ is a button labelled "Thêm vào đây". The selected box has `aria-pressed` or `aria-current`. After an insert, focus moves to the first field of the new element's panel.
- **The spike is deleted** (`web/src/app/devui/spike/` and its route in `app/devui/index.tsx`) in the same change. Adding the editor to the `/dev/ui` catalogue is left out unless the catalogue rules require it.

## UI text

Vietnamese labels (English follows the same meaning). Where the current key exists, the key keeps its name and the text changes. New texts get new keys under `shared.rule.*`.

| Where | Today | New |
| --- | --- | --- |
| Choose-one split | Rẽ nhánh chọn một | Tùy trường hợp — menu hint "Chỉ đi một nhánh, theo điều kiện" |
| Parallel split | Rẽ nhánh song song | Cùng lúc — menu hint "Nhiều nhánh duyệt song song" |
| Step | Bước duyệt | Bước duyệt — menu hint "Một người hoặc nhóm người duyệt" |
| Otherwise branch | Còn lại | Còn lại (panel: "Đi nhánh này khi không trường hợp nào đúng. Để trống thì không cần duyệt thêm.") |
| Branch title | Nhánh {n} | Trường hợp {n} (choose-one) / Nhánh {n} (parallel) |
| Approver field | Người duyệt | Ai duyệt |
| Approver kinds | Quản lý trực tiếp (theo chứng từ) / Theo vai trò tại đơn vị của chứng từ / Người dùng cụ thể | Quản lý trực tiếp của người gửi / Người có vai trò, tại đơn vị của người gửi / Những người cụ thể (the module kind keeps its per-document-type meaning; the label may stay generic if a type's module approvers aren't the manager) |
| Approvals | Số người cần duyệt | Cần bao nhiêu người đồng ý |
| Rejections | Số người từ chối để loại | Bao nhiêu người từ chối thì trả đơn (under *Nâng cao*; hint "Thường là 1: một người từ chối là đủ.") |
| Due | Thời hạn (giờ) | Hạn duyệt (giờ) — hint "Để trống nếu không có hạn. Quá hạn thì nhắc người duyệt và người gửi." |
| Fallback section | Khi không có người duyệt / Vai trò dự phòng | Cài đặt chung / "Không tìm được người duyệt thì chuyển cho" |
| Max levels | Số cấp quản lý tối đa | "Quản lý trực tiếp là người gửi thì đi lên tối đa (cấp)" |
| Split panel hint (choose-one) | — | "Xét các trường hợp từ trái sang phải; đi theo trường hợp đầu tiên đúng. Không trường hợp nào đúng thì đi nhánh \"Còn lại\"." (matches the engine: first holding branch) |
| Split panel hint (parallel) | — | "Mọi nhánh có điều kiện đúng cùng chạy một lúc; đi tiếp khi tất cả đã duyệt xong." |
| Under a parallel split | — | "Đi tiếp khi mọi nhánh đã xong" |
| Empty panel | — | "Bấm vào một ô trên sơ đồ để sửa. Bấm dấu + trên đường nối để thêm bước hoặc tách nhánh ngay tại đó." |

Condition chips read the condition back in words: field label, comparison word (`>`, `≥`, "là", "thuộc", "có"), value label (choice label, org unit name, searched option name). Rows are joined with "và" / "hoặc" and groups go in parentheses. An unfinished leaf shows "…". No condition shows "Luôn đúng" (choose-one) or "Luôn chạy" (parallel). The builder settles the exact wording and the English texts.

## Switching split kind

Switching a split from *Tùy trường hợp* to *Cùng lúc* drops its otherwise branch. If the otherwise branch has steps, ask first, as today (`drop_otherwise`). The switch lives in the split's panel as a segmented control.

## Deleting

Deleting a step, a split or a branch from the panel doesn't ask: *Bỏ thay đổi* covers it (`ui.md`: edits inside an unsaved form don't ask). A split keeps at least one branch.

## Tests

- Unit tests (Vitest) for the pure parts: reading a condition back as words, a step's summary and its client-side problem, mapping a server `path` to the element to select (beyond `formPathOf`, which is already tested).
- E2E: the specs that drive the editor through the UI rewrite their selectors to the diagram. They are `approval-flows.spec.ts`, `approval-lines.spec.ts`, `approval-overdue.spec.ts`, `approval-actions.spec.ts`, `m4.spec.ts` and `m8.spec.ts`. Check each one; most only navigate or set rules through the API. What they cover stays covered: build a nested flow through the UI and check the saved JSON, a refused save marks the right element, switching to parallel asks before dropping otherwise steps, read-only mode, the due field.
- Specs that set rules only through the API don't change.

## Left to the builder

- Component structure inside `shared/document/flowEditor/` (which files, how selection state is held), and whether `StepCard`/`SplitCard`/`FlowBlock` are removed or reshaped.
- The connector CSS details, spacing, box widths and colours within the theme tokens. The spike's choices are a fair starting point: warning tones for choose-one, discovery for parallel, primary for start, success for end, danger for problems.
- Exact i18n keys and English wording.
- Whether the panel uses `Panel` or another `shared/ui` surface.
