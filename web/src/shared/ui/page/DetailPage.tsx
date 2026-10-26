import { ActionBar, Group, RecordColumns, useMediaQuery } from '@mmerp/ui'
import type { ReactNode } from 'react'
import { MoreActions } from '../MoreActions'
import { Page, type PageHeaderProps } from './Page'

type Props = Omit<PageHeaderProps, 'tabs' | 'withBack' | 'plain'> & {
  // Menu.Item children: what does not earn a button, in a "…" menu after the actions.
  more?: ReactNode
  // Under the header, over both columns: a page-level error.
  banner?: ReactNode
  // The side column: status and approval, attachments, then the activity, each a Panel.
  side: ReactNode
  // Below 1024px, where the header's actions go instead (a bar at the bottom of the screen).
  narrowActions?: ReactNode
  children: ReactNode
}

// DetailPage: the frame of a document's page (DocumentPage). Header with back, title and status,
// key facts, actions on the right; then the main column (the values, in cards) and a side column at
// about 60/40 of plain Panels parted by rules, one column below 1024px with the side column under
// the main one. Header and columns share one width and left edge.
export function DetailPage({ more, banner, side, narrowActions, actions, children, ...header }: Props) {
  const wide = useMediaQuery('(min-width: 64em)', true)
  // Rendered in one place only, so each action exists once on the page.
  const bar = !wide && narrowActions
  const all = !bar && (actions || more) && (
    <Group gap="xs" justify="end" wrap={false}>
      {actions}
      {more && <MoreActions place="page">{more}</MoreActions>}
    </Group>
  )
  return (
    <Page {...header} withBack actions={all} centered="detail">
      {banner}
      <RecordColumns ratio="document" main={children} side={side} />
      {bar && <ActionBar>{narrowActions}</ActionBar>}
    </Page>
  )
}
