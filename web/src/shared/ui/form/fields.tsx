import { Checkbox, Combobox, FileInput, MultiSelect, NumberInput, PasswordInput, Select, Textarea, TextInput } from '@mmerp/ui'
import { DateInput, MonthPickerInput } from '@mantine/dates'
import { IconCalendar } from '@tabler/icons-react'
import type { ReactElement } from 'react'
import { useController, useFormContext, type RegisterOptions } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { icon } from '../theme'
import { fieldId } from './Form'
import classes from './Form.module.css'

// bare drops the visible label (kept as aria-label) and the description, for a field in a table
// cell under a column header that already names it. short takes a third of a FormSection row
// instead of half, for a value that is short whatever it holds (a date, a choice, a phone number).
// error shows a message the form's own validation does not hold (a check made over the whole form).
type Base = { name: string; label: string; required?: boolean; description?: string; readOnly?: boolean; bare?: boolean; short?: boolean; error?: string }

// Every field: label above, * when required, error under the input, checked on blur and submit.
// A read-only field shows only its value: no *, no description (the kit greys its box).
function useField({ name, label, required, description, bare, short, readOnly, error }: Base, rules: RegisterOptions = {}) {
  const { t } = useTranslation()
  const { control } = useFormContext()
  const { field, fieldState } = useController({
    name,
    control,
    rules: { required: required ? t('shared.form.required') : false, ...rules },
  })
  return {
    field,
    common: {
      id: fieldId(name),
      label: bare ? undefined : label,
      'aria-label': bare ? label : undefined,
      required: required && !readOnly,
      description: bare || readOnly ? undefined : description,
      readOnly,
      error: fieldState.error?.message ?? error,
      onBlur: field.onBlur,
      ref: field.ref,
    },
    // short takes a third of a FormSection row through the kit's [data-short] hook.
    place: (el: ReactElement) => (short ? <div data-short="">{el}</div> : el),
  }
}

// mantine maps the common props onto a Mantine dates input.
function mantine({ required, ...common }: ReturnType<typeof useField>['common'], { readOnly, short }: Base) {
  return { ...common, withAsterisk: required, wrapperProps: { 'data-readonly': readOnly || undefined, 'data-short': short || undefined } }
}

export function TextField(
  props: Base & {
    type?: 'text' | 'email' | 'tel'
    maxLength?: number
    multiline?: boolean
    wide?: boolean
    autoComplete?: string
    // An example of the format ("VD: 0912 345 678"), only where the label does not make it plain.
    placeholder?: string
    rules?: RegisterOptions
  },
) {
  const { field, common, place } = useField(props, props.rules)
  const shared = {
    ...common,
    maxLength: props.maxLength,
    // A read-only field shows only its value; a sample there would read as data.
    placeholder: props.readOnly ? undefined : props.placeholder,
    // Long text spans the whole row of a FormSection grid.
    className: props.wide || props.multiline ? classes.wide : undefined,
    value: field.value ?? '',
  }
  if (props.multiline) return place(<Textarea {...shared} autosize minRows={2} onChange={(e) => field.onChange(e.currentTarget.value)} />)
  return place(<TextInput {...shared} type={props.type} autoComplete={props.autoComplete} onChange={(e) => field.onChange(e.currentTarget.value)} />)
}

// CodeField: an entry's code, typed or left empty to be issued at creation, read-only after.
export function CodeField({ name, label, existing }: { name: string; label: string; existing: boolean }) {
  const { t } = useTranslation()
  if (existing) return <TextField name={name} label={label} readOnly />
  return <TextField name={name} label={label} description={t('shared.form.code_hint')} maxLength={50} />
}

export function PasswordField(props: Base & { autoComplete?: string }) {
  const { t } = useTranslation()
  const { field, common, place } = useField(props)
  return place(
    <PasswordInput
      {...common}
      autoComplete={props.autoComplete ?? 'new-password'}
      value={field.value ?? ''}
      onChange={(e) => field.onChange(e.currentTarget.value)}
      visibilityLabels={{ show: t('shared.form.show_password'), hide: t('shared.form.hide_password') }}
    />,
  )
}

// parseDate reads a typed DD/MM/YYYY date; the default parser would take it as month first.
export function parseDate(input: string): string | null {
  const m = /^(\d{1,2})[/.-](\d{1,2})[/.-](\d{4})$/.exec(input.trim())
  if (!m) return null
  const [, d, mo, y] = m.map(Number) as [number, number, number, number]
  const date = new Date(Date.UTC(y, mo - 1, d))
  if (date.getUTCMonth() !== mo - 1 || date.getUTCDate() !== d) return null
  return date.toISOString().slice(0, 10)
}

// A date in a form shows a calendar and, while it can be edited, its format; a table cell has no
// room for either.
function dateLook({ bare, readOnly }: Base, format: string) {
  if (bare) return {}
  return { leftSection: <IconCalendar {...icon.text} />, leftSectionPointerEvents: 'none' as const, placeholder: readOnly ? undefined : format }
}

// DateField holds a business date as YYYY-MM-DD; shown in the user's format, weeks start on Monday.
export function DateField(props: Base & { clearable?: boolean }) {
  const { t } = useTranslation()
  const { field, common } = useField(props)
  return (
    <DateInput
      {...mantine(common, props)}
      {...dateLook(props, t('shared.form.date_placeholder'))}
      value={field.value ?? null}
      onChange={(v) => field.onChange(v ?? null)}
      valueFormat="DD/MM/YYYY"
      dateParser={parseDate}
      clearable={props.clearable}
    />
  )
}

// MonthField holds a month as YYYY-MM, shown as MM/YYYY.
export function MonthField(props: Base & { clearable?: boolean }) {
  const { t } = useTranslation()
  const { field, common } = useField(props)
  return (
    <MonthPickerInput
      {...mantine(common, props)}
      {...dateLook(props, t('shared.form.month_placeholder'))}
      value={field.value ? `${field.value}-01` : null}
      onChange={(v) => field.onChange(v ? v.slice(0, 7) : null)}
      valueFormat="MM/YYYY"
      clearable={props.clearable}
    />
  )
}

// FileField holds one chosen file; accept narrows the picker, e.g. ".xlsx".
export function FileField(props: Base & { accept?: string }) {
  const { t } = useTranslation()
  const { field, common, place } = useField(props)
  return place(
    <FileInput
      {...common}
      accept={props.accept}
      value={field.value ?? null}
      onChange={(f) => field.onChange(f)}
      clearLabel={t('shared.form.clear', { field: props.label })}
    />,
  )
}

export type Option = { value: string; label: string }

// useClearing gives a kit select its clear button, named after the field.
export function useClearing(clearable: boolean | undefined, label: string): { clearable: true; clearLabel: string } | { clearable?: false } {
  const { t } = useTranslation()
  return clearable ? { clearable: true, clearLabel: t('shared.form.clear', { field: label }) } : {}
}

// SelectField can be typed into when there are more than seven options, or when onSearch looks
// them up as the user types. onChange runs after the user picks, e.g. to fill other fields from the choice.
export function SelectField(props: Base & { data: Option[]; clearable?: boolean; onSearch?: (q: string) => void; onChange?: (v: string | null) => void }) {
  const clearing = useClearing(props.clearable, props.label)
  const { field, common, place } = useField(props)
  const shared = {
    ...common,
    data: props.data,
    value: field.value ?? null,
    onChange: (v: string | null) => {
      field.onChange(v)
      props.onChange?.(v)
    },
  }
  if (props.data.length > 7 || props.onSearch) return place(<Combobox {...shared} {...clearing} onSearchChange={props.onSearch} nothingFound="—" />)
  return place(<Select {...shared} {...clearing} />)
}

// MultiSelectField holds several values from a list (e.g. several approvers), typed into to narrow it.
export function MultiSelectField(props: Base & { data: Option[]; max?: number }) {
  const { t } = useTranslation()
  const { field, common, place } = useField(props)
  return place(
    <MultiSelect
      {...common}
      data={props.data}
      value={field.value ?? []}
      // Past max a pick is ignored.
      onChange={(v) => (props.max !== undefined && v.length > props.max ? undefined : field.onChange(v))}
      nothingFound="—"
      removeLabel={(item) => t('shared.form.remove_item', { item })}
    />,
  )
}

// A number field's form value may be a number, a numeric string or empty ('' or null).
const toNumber = (v: unknown) => (v === '' || v == null ? null : Number(v))

// separators of numbers follow the user's language: 1.234,5 in vi, 1,234.5 in en.
function useNumberLocale() {
  return useTranslation().i18n.language === 'en' ? 'en-US' : 'vi-VN'
}

// DecimalField holds a fractional quantity as a decimal string (e.g. "2.5"), never a float
// sent to the API; `scale` is the number of decimals the field allows.
export function DecimalField(props: Base & { scale: number; step?: number; min?: number; allowNegative?: boolean }) {
  const locale = useNumberLocale()
  const { field, common, place } = useField(props)
  return place(
    <NumberInput
      {...common}
      // In a table column of figures the digits line up on the right.
      className={props.bare ? classes.money : undefined}
      value={toNumber(field.value)}
      onChange={(v) => field.onChange(v === null ? '' : String(v))}
      locale={locale}
      format={{ maximumFractionDigits: props.scale }}
      step={props.step}
      min={props.min ?? (props.allowNegative ? undefined : 0)}
    />,
  )
}

export function CheckboxField(props: Omit<Base, 'required'>) {
  const { field, common, place } = useField(props)
  return place(
    <Checkbox
      id={common.id}
      ref={common.ref}
      label={common.label}
      aria-label={common['aria-label']}
      description={common.description}
      error={common.error}
      checked={!!field.value}
      disabled={props.readOnly}
      onChange={field.onChange}
      onBlur={common.onBlur}
    />,
  )
}

// MoneyField holds an amount in minor units (đồng: whole numbers) with grouped digits.
export function MoneyField(props: Base & { min?: number }) {
  const locale = useNumberLocale()
  const { field, common, place } = useField(props)
  return place(
    <NumberInput
      {...common}
      className={classes.money}
      value={toNumber(field.value)}
      onChange={field.onChange}
      locale={locale}
      format={{ maximumFractionDigits: 0 }}
      min={props.min ?? 0}
    />,
  )
}

// IntegerField holds a whole number (a port, a year, a count of levels) without digit grouping,
// so 8080 and 2026 read as typed.
export function IntegerField(props: Base & { min?: number; max?: number; rules?: RegisterOptions }) {
  const { field, common, place } = useField(props, props.rules)
  return place(
    <NumberInput
      {...common}
      value={toNumber(field.value)}
      onChange={field.onChange}
      format={{ maximumFractionDigits: 0, useGrouping: false }}
      min={props.min ?? 0}
      max={props.max}
    />,
  )
}
