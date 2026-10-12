# Forms

Forms use React Hook Form through `@/shared/ui/form`: `Form`, `FormSection`, `FormRow`, `InputRow`, `FormActions` and the fields. A field is wired by `name`; the screen never renders a bare Mantine input inside a form.

## Page or modal

| The form | Where |
| --- | --- |
| Is the record itself (a document, an employee, a customer) | A page: `Page` + `Form` to create, the record's own page to view and edit (view = edit) |
| Needs at most 6 fields, and the result is worked on its own page or lives in a list (timesheet period, payroll period, item, user, a setting) | `FormModal` on the list |
| Configuration with more than 6 fields or tables of its own (approval rules, mail server) | A page |
| One value of a master-data record, changed often | `InlineEditField` on the record page (below) |

There is no "Create another" option; series entry goes through Excel import.

## Structure

```
┌ FormSection ───────────────────────────────────────────────────────────┐
│ ▣ Section title          Label *                 Label                 │
│ One-sentence             [               ]       [               ]     │
│ description              help text                                     │
│                          Label                                         │
│                          [                                         ]   │  long text: wide
├────────────────────────────────────────────────────────────────────────┤  1px rule, 24px apart
│ FormSection wide: title and description on top, content full width      │
└────────────────────────────────────────────────────────────────────────┘
▔ FormActions (page): Unsaved changes · Discard            [Save] ▔ sticky
```

- Sections (`FormSection`) are unframed, separated by a rule; from 1280px the title column is on the left and fields go two per row; long text (`wide` on `TextField`, `multiline`) spans the row. A section is at most 1200px wide.
- `FormSection wide` puts the title on top and gives its content the full width: for a lines table or any table inside a form.
- `FormRow` spans the field grid for something that is not a field (a table, totals). `InputRow` puts a checkbox or a remove button level with the inputs beside it.
- Order fields by how people fill them: what identifies the record first, then what it is about, then terms and notes. Group what is filled together; at most about eight fields per section.
- A field that depends on another appears or changes only after that one is set (contract kind decides whether an end date shows). Picking from a catalogue fills dependent fields that stay editable (item → description, price, VAT); fill only empty fields when the user may already have typed one (customer → payment terms).

## Labels, help, required

- Label above every field, always visible, 12px weight 600 `dimmed` (the theme does it); never a placeholder instead of a label.
- `required` adds `*` after the label; no legend, no "(required)".
- `description` is one sentence under the input, only when the label is not enough.
- In a table cell, `bare` drops the visible label (kept as `aria-label`) and the description: the column header names the field.

## Validation

| When | What happens |
| --- | --- |
| Leaving a field (`mode: 'onBlur'`) | That field is checked; its error shows under it, linked by `aria-describedby` |
| Submit | Every field is checked; focus moves to the first invalid field; each `FormSection` title carries its error count; next to the submit button "N fields to fix · Go to next field" cycles through them |
| Server answers with a code that names a field | `fieldOf` maps it onto that field (`err.code === 'customer_code_taken' ? 'code' : undefined`); for a line, `lines.<i>.<field>` |
| Server answers with anything else | A banner at the top of the form (`Form` does it); a document-level error (period locked) at the top of the page |
| Version conflict | The "Data has changed · Reload" dialog (`useDocumentMutation`) |

Error text says what is wrong without repeating the label: "Chưa nhập." (Not entered.). The submit button is never disabled to signal invalid input; it stays clickable and validation explains.

## Footer and saving

- **On a page**: `FormActions` without `onCancel`. Hidden while nothing changed; once dirty it sticks to the bottom: "Unsaved changes", a red Discard text button (asks first), Save. Ctrl/Cmd+S saves. Leaving the page with changes asks first.
- **In a modal**: `FormActions` with `onCancel`: Cancel and the submit button, right-aligned, not sticky.
- The submit button names the outcome: "Lưu", "Tạo khách hàng", "Tạo bản nháp".
- While saving the button is busy (spinner only after 300ms); after a successful save: `notifySuccess`, then a create navigates to the new record's page.

```tsx
<Page title={t('sales.customers.create')} breadcrumbs={[{ label: t('sales.customers.title'), to: '/sales/customers' }]}>
  <Form form={form} onSubmit={save} fieldOf={(err) => (err.code === 'customer_code_taken' ? 'code' : undefined)}>
    <FormSection title={t('sales.customer.section.general')} description={t('sales.customer.section.general_hint')} icon={IconAddressBook}>
      <TextField name="code" label={t('sales.customer.code')} required maxLength={50} />
      <TextField name="name" label={t('sales.customer.name')} required maxLength={300} />
      <TextField name="address" label={t('sales.customer.address')} maxLength={500} wide />
    </FormSection>
    <FormActions submitLabel={t('sales.customers.create')} />
  </Form>
</Page>
```

## Saving a document

A document's form saves through `useDocumentMutation(docType, id, version)` (`@/shared/document`), which sends the version and handles a conflict:

```tsx
const mutation = useDocumentMutation(docType, doc?.id ?? 0, doc?.version ?? 0)
const [requestId] = useState(() => crypto.randomUUID())                 // one per form: a retried create is not a second document
async function save(v: Values) {
  if (doc) await mutation.run((version) => unwrap(api.PUT('/sales/quotes/{id}', { params: { path: { id: doc.id } }, body: { version, ...toBody(v) } })))
  else onCreated((await unwrap(api.POST('/sales/quotes', { body: { request_id: requestId, ...toBody(v) } }))).id)
}
…
<Form form={form} onSubmit={save} fieldOf={fieldOf}>… {mutation.dialog}</Form>  // render the conflict dialog
```

The page keys the form on the version (`<DocForm key={d.version} … />`), so a saved or reloaded document starts clean. Option lists that come from settings (VAT rates, units) come from the area's own hook (`useVatOptions()` in sales), never typed into the screen.

## Fields by data

| Data | Field |
| --- | --- |
| Text, long text | `TextField` (`multiline`, `wide`) |
| Password | `PasswordField` |
| Yes/no | `CheckboxField` (in `InputRow` beside inputs) |
| Date, month | `DateField`, `MonthField` |
| Whole number (port, year, count) | `IntegerField` (no digit grouping) |
| Fractional quantity, percent | `DecimalField` with its `scale` |
| Money | `MoneyField` (minor units, grouped, right-aligned) |
| One of a list | `SelectField` (search past 7 options; server search with `onSearch`) |
| A business record (customer, item, employee) | The owning area's picker (`CustomerField`, `ItemField`), built on `SelectField`, label `code · name` |
| Org unit | `OrgUnitField` |
| File | `FileField` |
| Sensitive value | `SensitiveField` |

The same data uses the same field everywhere (a month is never a text input checked by a pattern).

## Document lines

A document with lines (quotation, order) edits them in `LinesTable`, inside a `FormSection wide`, on a `DocumentPage fullWidth` (the table needs the width; approval and history move to tabs):

```
# │ Item *        │ Description *   │ Qty * │ Unit price * │ Disc. % │ VAT *  │
1 │ [MIN-LZ · …▾] │ [Máy in laser ] │ [  3] │ [ 3.900.000] │ [    0] │ [10%▾] │ 🗑
2 │ [          ▾] │ [            ]  │ [  1] │ [         0] │ [    0] │ [   ▾] │ 🗑
+ Add line
                                    Amounts as of the last save; save to recompute.
                                                      Subtotal (đ)     19.200.000
                                                      VAT (đ)           1.920.000
                                                      Total (đ)        21.120.000
```

```tsx
const lineColumns: LineColumn[] = [
  { key: 'item', header: t('sales.line.item_col'), required: true, width: 200,
    cell: (i, bare, line) => <ItemField name={`lines.${i}.item_id`} label={t('sales.line.item', { n: i + 1 })} bare={bare} onPicked={…} /> },
  { key: 'quantity', header: t('sales.line.quantity_col'), numeric: true, required: true, width: 96,
    cell: (i, bare) => <DecimalField name={`lines.${i}.quantity`} label={t('sales.line.quantity', { n: i + 1 })} bare={bare} required scale={3} min={0} /> },
  …
]
<FormSection title={t('sales.doc.section.lines')} description={…} icon={IconListDetails} wide>
  <FormRow>
    <LinesTable
      name="lines" label={t('sales.doc.section.lines')} columns={lineColumns}
      newLine={() => ({ ...emptyLine })} addLabel={t('sales.line.add')}
      lineLabel={(n) => t('sales.line.title', { n })} removeLabel={(n) => t('sales.line.remove', { n })}
      minLines={1}
      footer={doc && <LinesTotals note={t('sales.doc.totals_saved')} rows={[[t('sales.doc.subtotal'), formatNumber(doc.subtotal)], …, [t('sales.doc.total'), formatNumber(doc.total)]]} />}
    />
  </FormRow>
</FormSection>
```

- One row per line, `#` first, a grey remove button last (no confirmation: Discard covers mistakes; hidden at `minLines`). "Add line" appends an empty line at the end.
- Each column gives its smallest `width`; the one without takes the rest. A frame too narrow scrolls sideways.
- Cells use the normal fields with `bare`; their `aria-label` names the line ("Số lượng (1)"). Numeric cells are right-aligned without spin buttons.
- Keyboard: Tab moves cell by cell; Enter moves to the next cell instead of submitting (an open dropdown keeps Enter to pick).
- **Totals are the server's.** Rounding happens only on the server, so the footer shows the stored totals with a note that they are as of the last save; a new document shows none. Never multiply, divide, apply a percentage or round money in the screen. Adding or subtracting stored amounts of one document (net = amount − discount) is exact and allowed only when the API sends no field for it.
- Line errors from the server map with `fieldOf` to `lines.<i>.<field>` (the server's `line` parameter is 1-based).
- Below 1024px each line is a card with labelled fields.
- Rows saved on their own, apart from the document's form (a payroll's adjustments), sit in a `Section` holding its own `Form`, `LinesTable` and page `FormActions`; see `hrm/components/AdjustmentsSection.tsx`. A table never goes into a `FormModal`.
- A read-only document (no `edit` in `allowed_actions`) shows its lines as a `DataTable` instead of `LinesTable`: description, quantity with unit, unit price, discount, VAT rate, net amount, all from the stored line; then `LinesTotals` without the note.

## Inline edit

`InlineEditField` edits one value of a master-data record on its page, without a form:

```tsx
<InlineEditField
  label={t('sales.customer.phone')}
  value={c.phone ?? ''}
  readOnly={!c.allowed_actions.includes('edit')}
  validate={(v) => (v ? null : t('shared.form.required'))}
  onSave={(phone) => save({ ...current, phone })} // sends the whole record with its version
/>
```

| Interaction | Behaviour |
| --- | --- |
| Read view | The value as text under its label, in the input's exact box; a tint and border on hover and focus show it is editable |
| Start | Click, or Tab to it and Enter |
| Save | Enter or ✓: validates, saves, returns to the read view; the error shows under the input otherwise |
| Cancel | Esc or ✕: back to the read view, unchanged |
| Leaving the input | Keeps it open: a half-typed value is neither saved nor lost |
| No permission | `readOnly`: plain text, not a button |

Use it only for single values of master data that people change one at a time (phone, email, address, a name). Never for documents (they save as a whole with their version and approval), money, rates, quantities, sensitive fields, or a value whose change affects other fields; those stay in the form with one Save. Each inline save is one request, one version, one audit entry, and a toast is not needed: the value itself shows the result.

## Do / Don't

```tsx
// Do
<IntegerField name="port" label={t('core.mail.port')} required min={1} max={65535} />
<FormActions submitLabel={t('sales.common.save')} />                  // page form: no onCancel

// Don't
<TextField name="month" rules={{ pattern: /^\d{4}-\d{2}$/ }} … />     // use MonthField
<Button type="submit" disabled={!form.formState.isValid}>…</Button>   // never disable to signal errors
<Text fz={13}>{t('…label')}</Text>                                    // labels come from the field
const total = lines.reduce((s, l) => s + l.qty * l.price, 0)          // totals are the server's
```
