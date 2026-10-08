import { Checkbox, FileInput, NumberInput, PasswordInput, Select, Textarea, TextInput } from '@mantine/core'
import { DateInput, MonthPickerInput } from '@mantine/dates'
import { IconFileSpreadsheet } from '@tabler/icons-react'
import { useController, useFormContext, type RegisterOptions } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { icon } from '../theme'
import { fieldId } from './Form'
import classes from './Form.module.css'

// separators of numbers in the user's language: 1.234,5 in vi, 1,234.5 in en.
function separators(language: string) {
  return language === 'en' ? { decimalSeparator: '.', thousandSeparator: ',' } : { decimalSeparator: ',', thousandSeparator: '.' }
}

type Base = { name: string; label: string; required?: boolean; description?: string; readOnly?: boolean }

// Long text spans the whole row of a FormSection grid.
const wideStyle = { gridColumn: '1 / -1' }

// Every field: label above, * when required, error under the input, checked on blur and submit.
function useField({ name, label, required }: Base, rules: RegisterOptions = {}) {
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
      label,
      withAsterisk: required,
      error: fieldState.error?.message,
      onBlur: field.onBlur,
      ref: field.ref,
    },
  }
}

export function TextField(
  props: Base & { type?: 'text' | 'email' | 'tel'; maxLength?: number; multiline?: boolean; wide?: boolean; autoComplete?: string; rules?: RegisterOptions },
) {
  const { field, common } = useField(props, props.rules)
  const shared = {
    ...common,
    description: props.description,
    readOnly: props.readOnly,
    maxLength: props.maxLength,
    style: props.wide || props.multiline ? wideStyle : undefined,
    value: field.value ?? '',
    onChange: (e: { currentTarget: { value: string } }) => field.onChange(e.currentTarget.value),
  }
  if (props.multiline) return <Textarea {...shared} autosize minRows={2} />
  return <TextInput {...shared} type={props.type} autoComplete={props.autoComplete} />
}

export function PasswordField(props: Base & { autoComplete?: string }) {
  const { t } = useTranslation()
  const { field, common } = useField(props)
  return (
    <PasswordInput
      {...common}
      description={props.description}
      autoComplete={props.autoComplete ?? 'new-password'}
      value={field.value ?? ''}
      onChange={(e) => field.onChange(e.currentTarget.value)}
      visibilityToggleButtonProps={{ 'aria-label': t('shared.form.toggle_password'), tabIndex: 0 }}
    />
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

// DateField holds a business date as YYYY-MM-DD; shown in the user's format, weeks start on Monday.
export function DateField(props: Base & { clearable?: boolean }) {
  const { field, common } = useField(props)
  return (
    <DateInput
      {...common}
      description={props.description}
      readOnly={props.readOnly}
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
  const { field, common } = useField(props)
  return (
    <MonthPickerInput
      {...common}
      description={props.description}
      readOnly={props.readOnly}
      value={field.value ? `${field.value}-01` : null}
      onChange={(v) => field.onChange(v ? v.slice(0, 7) : null)}
      valueFormat="MM/YYYY"
      clearable={props.clearable}
    />
  )
}

// FileField holds one chosen file; accept narrows the picker, e.g. ".xlsx".
export function FileField(props: Base & { accept?: string }) {
  const { field, common } = useField(props)
  return (
    <FileInput
      {...common}
      description={props.description}
      accept={props.accept}
      leftSection={<IconFileSpreadsheet {...icon.text} />}
      value={field.value ?? null}
      onChange={(f) => field.onChange(f)}
      clearable
    />
  )
}

export type Option = { value: string; label: string }

// SelectField adds a search box when there are more than seven options.
// belowError opens a dropdown under the field's error message rather than over it.
// ponytail: sized for a one-line message (8px gap + ~20px line); measure it if messages wrap.
export const belowError = (error?: string) => ({ offset: error ? 28 : 8 })

export function SelectField(props: Base & { data: Option[]; clearable?: boolean; onSearch?: (q: string) => void }) {
  const { field, common } = useField(props)
  return (
    <Select
      {...common}
      description={props.description}
      readOnly={props.readOnly}
      data={props.data}
      value={field.value ?? null}
      onChange={(v) => field.onChange(v)}
      searchable={props.data.length > 7 || !!props.onSearch}
      onSearchChange={props.onSearch}
      clearable={props.clearable}
      nothingFoundMessage="—"
      comboboxProps={belowError(common.error)}
    />
  )
}

// DecimalField holds a fractional quantity as a decimal string (e.g. "2.5"), never a float
// sent to the API; `scale` is the number of decimals the field allows.
export function DecimalField(props: Base & { scale: number; step?: number; min?: number; allowNegative?: boolean }) {
  const { i18n } = useTranslation()
  const { field, common } = useField(props)
  return (
    <NumberInput
      {...common}
      description={props.description}
      readOnly={props.readOnly}
      value={field.value ?? ''}
      onChange={(v) => field.onChange(v === '' ? '' : String(v))}
      decimalScale={props.scale}
      step={props.step}
      min={props.min}
      allowNegative={props.allowNegative ?? false}
      {...separators(i18n.language)}
    />
  )
}

export function CheckboxField(props: Omit<Base, 'required'>) {
  const { field, common } = useField(props)
  return (
    <Checkbox
      id={common.id}
      ref={common.ref}
      label={props.label}
      description={props.description}
      error={common.error}
      checked={!!field.value}
      disabled={props.readOnly}
      onChange={(e) => field.onChange(e.currentTarget.checked)}
      onBlur={common.onBlur}
    />
  )
}

// MoneyField holds an amount in minor units (đồng: whole numbers) with grouped digits.
export function MoneyField(props: Base & { min?: number }) {
  const { i18n } = useTranslation()
  const { field, common } = useField(props)
  return (
    <NumberInput
      {...common}
      description={props.description}
      readOnly={props.readOnly}
      value={field.value ?? ''}
      onChange={(v) => field.onChange(v === '' ? null : Number(v))}
      decimalScale={0}
      min={props.min ?? 0}
      allowNegative={(props.min ?? 0) < 0}
      allowDecimal={false}
      {...separators(i18n.language)}
      hideControls
      classNames={{ input: classes.money }}
    />
  )
}
