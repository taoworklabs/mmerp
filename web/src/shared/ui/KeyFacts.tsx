import { Text } from '@mantine/core'
import type { ReactNode } from 'react'
import classes from './KeyFacts.module.css'

// KeyFacts is a record's few figures read first (manager, tenure, leave left…), a sunken strip
// under its page title: label over value, side by side, two to a row on a phone.
export function KeyFacts({ facts }: { facts: [string, ReactNode][] }) {
  return (
    <dl className={classes.facts}>
      {facts.map(([label, value]) => (
        <div key={label} className={classes.fact}>
          <Text component="dt" size="xs" c="dimmed">
            {label}
          </Text>
          <Text component="dd" fw={500}>
            {value}
          </Text>
        </div>
      ))}
    </dl>
  )
}
