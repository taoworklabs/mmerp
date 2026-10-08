import { Text } from '@mantine/core'
import classes from './FieldList.module.css'

// FieldList shows read-only label/value pairs: labels on the left, values on the right,
// so the eye runs down one column of values.
export function FieldList({ rows }: { rows: [string, string][] }) {
  return (
    <dl className={classes.fields}>
      {rows.map(([label, value], i) => (
        <div key={i} className={classes.row}>
          <Text component="dt" size="sm" c="dimmed">
            {label}
          </Text>
          <Text component="dd">{value}</Text>
        </div>
      ))}
    </dl>
  )
}
