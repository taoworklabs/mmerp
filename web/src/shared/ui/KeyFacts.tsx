import { Text } from '@mmerp/ui'
import type { ReactNode } from 'react'
import classes from './KeyFacts.module.css'

// KeyFacts is a record's few figures read first (manager, tenure, leave left…), sunken tiles
// under its page title: label over value, side by side, two to a row on a phone.
export function KeyFacts({ facts }: { facts: [string, ReactNode][] }) {
  return (
    <dl className={classes.facts}>
      {facts.map(([label, value]) => (
        <div key={label} className={classes.fact}>
          <Text render={<dt />} size="xs" tone="dimmed">
            {label}
          </Text>
          <Text render={<dd />} weight={500}>
            {value}
          </Text>
        </div>
      ))}
    </dl>
  )
}
