import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { Text } from '../../primitives/typography/Text.tsx'
import styles from './KeyFacts.module.css'

export interface KeyFactsProps extends React.ComponentPropsWithRef<'dl'> {
  facts: [string, React.ReactNode][]
}

// KeyFacts is a record's few figures read first (manager, tenure, leave left…), sunken tiles under
// its page title: label over value, side by side, two to a row on a phone.
export function KeyFacts({ facts, className, ...props }: KeyFactsProps) {
  return (
    <dl className={cx(styles.root, className)} {...props}>
      {facts.map(([label, value], index) => (
        <div key={index} className={styles.fact}>
          <Text size="xs" tone="dimmed" render={<dt />}>
            {label}
          </Text>
          <Text weight={500} render={<dd />}>
            {value}
          </Text>
        </div>
      ))}
    </dl>
  )
}
