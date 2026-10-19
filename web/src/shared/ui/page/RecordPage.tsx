import { Group } from '@mantine/core'
import type { ReactNode } from 'react'
import { MoreActions } from '../MoreActions'
import { Page, type PageHeaderProps } from './Page'

type Props = Omit<PageHeaderProps, 'tabs' | 'withBack' | 'plain' | 'current'> & {
  // Menu.Item children: the rarer and the destructive actions, in a "…" menu after the buttons.
  more?: ReactNode
  children: ReactNode
}

// RecordPage: one record with much to read (an employee). A summary header (breadcrumb ending with
// the page, title with status, key facts under it, actions on the right) over the record's content,
// centred at the detail width.
export function RecordPage({ more, actions, children, ...header }: Props) {
  const all = (actions || more) && (
    <Group gap="xs" justify="flex-end" wrap="nowrap">
      {actions}
      {more && <MoreActions place="page">{more}</MoreActions>}
    </Group>
  )
  return (
    <Page {...header} current actions={all} centered="var(--mmerp-detail-width)">
      {children}
    </Page>
  )
}
