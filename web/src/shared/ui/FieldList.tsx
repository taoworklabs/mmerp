import { Anchor, Text } from '@mmerp/ui'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import classes from './FieldList.module.css'

// A row: label, value, and 'wide' for a long value (an address) taking the whole row of a stacked list.
export type FieldRow = [string, ReactNode] | [string, ReactNode, 'wide']

// FieldList shows read-only label/value pairs: labels on the left, values on the right, so the
// eye runs down one column of values; stacked, the label sits over its value, two pairs to a row
// (one on a phone).
export function FieldList({ rows, stacked }: { rows: FieldRow[]; stacked?: boolean }) {
  return (
    <dl className={classes.fields} data-stacked={stacked || undefined}>
      {rows.map(([label, value, wide], i) => (
        <div key={i} className={classes.row} data-wide={wide && true}>
          <Text render={<dt />} size="sm" tone="dimmed">
            {label}
          </Text>
          <Text render={<dd />}>{value}</Text>
        </div>
      ))}
    </dl>
  )
}

// MissingValue stands for an empty value: a link to fill it in for who may (onAdd opens the form,
// to goes where it is entered), otherwise a quiet "not filled in".
export function MissingValue({ onAdd, to, label }: { onAdd?: () => void; to?: string; label: string }) {
  const { t } = useTranslation()
  const text = t('shared.field.add')
  const name = t('shared.field.add_label', { label })
  if (to)
    return (
      <Anchor render={<Link to={to} />} size="sm" aria-label={name}>
        {text}
      </Anchor>
    )
  if (onAdd)
    return (
      <Anchor render={<button />} type="button" size="sm" onClick={onAdd} aria-label={name}>
        {text}
      </Anchor>
    )
  return (
    <Text render={<span />} size="sm" tone="dimmed">
      {t('shared.field.missing')}
    </Text>
  )
}
