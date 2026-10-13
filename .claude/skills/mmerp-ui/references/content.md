# UI text

Every visible string is an i18n key (`<area>.<screen>.<element>`) with both `vi` and `en` values, in the area's `i18n/{vi,en}/*.json`. ESLint rejects literal strings in JSX; `make lint` rejects a key missing in either language. Words for business concepts come from `CONTEXT.md`: one concept, one word, on every screen.

## Voice

- **Short, neutral, impersonal.** No "bạn", no "please", no exclamation marks.
- **Imperative for instructions:** "Chọn kỳ lương" (Choose the pay period), not "Bạn hãy chọn kỳ lương".
- **Sentence case** everywhere, buttons and titles included: capitalise the first word and proper names only. "Gửi duyệt", "Submit for approval".
- **Plain words over system words:** "Đã ghi sổ" (Posted), not the status code; never show `posted`, `pending_approval` or an error code in the main text.

## Labels

| Element | Rule | Example (vi / en) |
| --- | --- | --- |
| Field label | A noun phrase naming the value, no colon, no verb | "Mã khách hàng" / "Customer code" |
| Required | The `*` from `required`; never "(bắt buộc)" in the label | — |
| Help text | One sentence under the input, only when the label is not enough | "Dùng cho hóa đơn điện tử." |
| Filter placeholder | The filter's name, same as its `aria-label` | "Trạng thái" / "Status" |
| Search placeholder | What can be searched, plus how it applies | "Mã hoặc họ tên, nhấn Enter" |
| Field placeholder | Only a format: dates show it by themselves; otherwise a sample after "VD:", where the label does not make the format plain | "VD: 0912 345 678" / "e.g. 0912 345 678" |
| Column header | Short noun; the unit of a money column goes here | "Lương cơ bản (₫)" |
| Section title + description | Title is a noun; description one sentence on what the section holds | "Thông tin chính" · "Danh tính và vị trí của nhân viên trong tổ chức." |
| Page description | One line: what the list holds, or the record's key facts | "Hồ sơ nhân viên trong các đơn vị được phân quyền." |

## Buttons

- **A specific verb, plus an object when the page has several:** "Lưu" (Save), "Gửi duyệt" (Submit for approval), "Rút lại" (Withdraw), "Duyệt" (Approve), "Từ chối" (Reject), "Hủy chứng từ" (Cancel document), "Tạo khách hàng" (Create customer).
- Name the outcome, not the control: "OK", "Đồng ý" (Agree), "Có" (Yes) never appear.
- A confirmation dialog's confirm button repeats the verb of the action ("Hủy chứng từ"); its other button is "Quay lại" (Back).
- Icon-only buttons: the `aria-label` and tooltip say what the button opens or does ("Thao tác khác", "Thông báo").

## Messages

| Kind | Rule | Example |
| --- | --- | --- |
| Field error | What is wrong, without the field name (the label is right above) | "Chưa nhập." (Not entered.) |
| Document error (banner) | What happened, then what to do | "Kỳ tháng 3 đã khóa sổ. Liên hệ kế toán nếu cần sửa chứng từ này." |
| Version conflict | Always the reload dialog | "Dữ liệu đã thay đổi" + "Tải lại" |
| Success toast | The fact, past tense, a few words | "Đã thêm khách hàng." |
| Empty list | Why it is empty, short; the next step as a button | "Chưa có khách hàng" |
| No results | "Không có kết quả" + "Xóa bộ lọc" (Clear filters) | — |
| Forbidden | Explain and give the way out; never hide silently | "Tài khoản này không có quyền xem trang. Liên hệ quản trị hệ thống nếu cần quyền." |

Errors come from the API as codes with parameters; the frontend maps each code to a key through `errorText(t, err)`. Technical detail goes only in a collapsed section meant for support.

## Action labels

| Place | Pattern | Example (vi / en) |
| --- | --- | --- |
| Create button on a list | "Tạo" + the record | "Tạo khách hàng" / "Create customer" |
| Submit of a create form | The outcome | "Tạo bản nháp" / "Create draft" |
| Submit of an edit form | "Lưu" | "Lưu" / "Save" |
| Lifecycle | The fixed verbs | "Gửi duyệt", "Rút lại", "Duyệt", "Từ chối", "Xóa nháp", "Hủy chứng từ" |
| Menu item ("…") | Verb + object, sentence case, no ellipsis unless it opens a dialog that asks for more | "Tạo đơn bán hàng", "In phiếu lương…" |
| Bulk action | Verb + what it acts on, the count shown by the bar | "Gửi duyệt", "Xuất Excel" |
| Icon-only | The action as `aria-label` and tooltip | "Xóa dòng 2", "Thao tác khác" |
| Clear, undo | "Xóa bộ lọc", "Bỏ chọn", "Hoàn tác" | — |

## Confirmation dialogs

- **Title**: the action, as on its button: "Hủy chứng từ", "Xóa chế độ xem".
- **Message**: names the object, then the consequence, in one or two sentences: "Hủy báo giá BG-2026-00012? Báo giá đã hủy không tạo được đơn hàng và không khôi phục được."
- **Buttons**: "Quay lại" (Back), then the confirm button repeating the verb ("Hủy chứng từ"); red for a destructive step.
- **Bulk**: the count in the message: "Gửi duyệt 12 đơn nghỉ đã chọn?"
- **Reason required** (reject): the field label says what the reason is for: "Lý do từ chối".

## Empty states

| Case | Title | Description |
| --- | --- | --- |
| Nothing yet | What is missing: "Chưa có khách hàng" | What appears here and how it gets here, one sentence: "Khách hàng tạo ở đây dùng được cho báo giá và đơn hàng." Shown whatever the user's permissions; only the button depends on them |
| No results | "Không có kết quả" | None; the button "Xóa bộ lọc" |
| Empty sub-table | Short: "Chưa có tuần làm việc" | Only if the next step is not obvious |

No apologies, no exclamation, no "Oops".

## Validation messages

- Say what is wrong in a few words, without the field's name: "Chưa nhập." · "Không quá 50 ký tự." · "Ngày kết thúc phải sau ngày bắt đầu."
- With a limit, give the limit: "Tối đa 100." rather than "Giá trị không hợp lệ."
- Server codes get their own key per code under the product's prefix (`sales.error.<code>`, `hrm.error.<code>`; core codes under `shared.error.<code>`), with the parameters the API sends: "Mã {{code}} đã có khách hàng khác dùng."
- The footer line after a failed submit is fixed: "Còn N trường cần sửa · Đến trường tiếp theo".

## Counts

Text with a number uses i18next plurals: the key ends in `_one` / `_other` and takes `count`; pass the formatted number separately (`n: formatNumber(count)`), as `BulkBar` does. Vietnamese needs only `_other`; English needs both.

## Toasts

- **Success**: the fact in the past tense, the object named when it helps: "Đã lưu khách hàng." · "Đã gửi duyệt BG-2026-00012."
- **With undo**: the fact, then the button "Hoàn tác": "Đã ngừng dùng mặt hàng Máy in laser." [Hoàn tác]. The fact must still be true if the user ignores the toast.
- **Job failed**: what failed and where to look: "Nhập bảng công lỗi. Xem chi tiết ở màn hình tác vụ."

## Numbers, money, dates

Format only through `@/shared/i18n` (`formatNumber`, `formatMoney`, `formatDecimal`, `formatDate`, `formatMonth`, `formatDateTime(iso, me.timezone)`, `formatAgo`); never call `Intl` or `toLocaleString` in a screen.

| Data | vi | en |
| --- | --- | --- |
| Number | `1.234.567,5` | `1,234,567.5` |
| Money in a table (unit in the header): `formatNumber` | `1.234.567` | `1,234,567` |
| Money on its own: `formatMoney` | `1.234.567 ₫` | `₫1,234,567` |
| Date | `31/03/2026` | `31/03/2026` |
| Date and time (tenant time zone) | `31/03/2026 14:05` | `31/03/2026 14:05` |
| Date range | `01/03/2026 – 31/03/2026` | same |

Zero in a reconciliation table shows "–" in `dimmed`.

Do:

```tsx
<Button>{t('sales.common.save')}</Button>
<Text size="sm" c="dimmed">{t('shared.list.total', { total: formatNumber(total) })}</Text>
```

Don't:

```tsx
<Button>Save</Button>                          // literal string: lint fails
<Button>{t('common.ok')}</Button>              // "OK" names no action
<Text>{total.toLocaleString()} bản ghi</Text>  // formatting and text outside i18n
```
