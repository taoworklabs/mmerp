import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { Text } from '../../primitives/typography/Text.tsx'
import styles from './FieldList.module.css'

// A row: label, value, and 'wide' for a long value (an address) taking the whole row when stacked.
export type FieldRow = [string, React.ReactNode] | [string, React.ReactNode, 'wide']

export interface FieldListProps extends React.ComponentPropsWithRef<'dl'> {
  rows: FieldRow[]
  // Label over value, two pairs to a row from 48em (one on a phone) instead of side by side.
  stacked?: boolean
}

// FieldList shows read-only label/value pairs: labels on the left, values on the right, so the eye
// runs down one column of values; stacked, the label sits over its value.
export function FieldList({ rows, stacked, className, ...props }: FieldListProps) {
  return (
    <dl className={cx(styles.root, className)} data-stacked={stacked ? '' : undefined} {...props}>
      {rows.map(([label, value, wide], index) => (
        <div key={index} className={styles.row} data-wide={wide ? '' : undefined}>
          <Text size="sm" tone="dimmed" render={<dt />}>
            {label}
          </Text>
          <Text render={<dd />}>{value}</Text>
        </div>
      ))}
    </dl>
  )
}
