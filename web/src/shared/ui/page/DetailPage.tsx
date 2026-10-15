import { Group, Paper, Stack, Tabs } from '@mantine/core'
import { useMediaQuery } from '@mantine/hooks'
import type { ReactNode } from 'react'
import { useSearchParams } from 'react-router'
import { MoreActions } from '../MoreActions'
import classes from './DetailPage.module.css'
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

// DetailPage: one record or document. Header with back, title, key facts and status, actions on
// the right; then the main column (the record's values, in Panels) and a side column at about
// 60/40, one column below 1024px with the side column under the main one. Header and columns
// share one width and left edge.
export function DetailPage({ more, banner, side, narrowActions, actions, children, ...header }: Props) {
  const wide = useMediaQuery('(min-width: 64em)', true)
  // Rendered in one place only, so each action exists once on the page.
  const bar = !wide && narrowActions
  const all = !bar && (actions || more) && (
    <Group gap="xs" justify="flex-end" wrap="nowrap">
      {actions}
      {more && <MoreActions place="page">{more}</MoreActions>}
    </Group>
  )
  return (
    <Page {...header} withBack actions={all} centered="var(--mmerp-detail-width)">
      {banner}
      <div className={classes.columns}>
        <Stack gap="md" miw={0}>
          {children}
        </Stack>
        <Stack gap="md" miw={0}>
          {side}
        </Stack>
      </div>
      {bar && (
        <>
          <div className={classes.barSpace} />
          <div className={classes.bar}>{narrowActions}</div>
        </>
      )}
    </Page>
  )
}

export type DetailTab = { value: string; label: string; content: ReactNode }

// DetailTabs is the record's own lists (contracts, dependants, roles…) as tabs in one card of the
// side column, the first open by default. The open tab is on the URL.
export function DetailTabs({ tabs }: { tabs: DetailTab[] }) {
  const [search, setSearch] = useSearchParams()
  const first = tabs[0]?.value ?? ''
  const asked = search.get('tab')
  const tab = tabs.some((x) => x.value === asked) ? (asked as string) : first
  return (
    <Paper withBorder p="md">
      <Tabs
        value={tab}
        onChange={(v) => {
          const next = new URLSearchParams(search)
          if (!v || v === first) next.delete('tab')
          else next.set('tab', v)
          // Replaced, not pushed: Back leaves the record rather than walking back through its tabs.
          setSearch(next, { replace: true })
        }}
        keepMounted={false}
      >
        <Stack gap="md">
          <Tabs.List>
            {tabs.map((x) => (
              <Tabs.Tab key={x.value} value={x.value}>
                {x.label}
              </Tabs.Tab>
            ))}
          </Tabs.List>
          {tabs.map((x) => (
            <Tabs.Panel key={x.value} value={x.value}>
              {x.content}
            </Tabs.Panel>
          ))}
        </Stack>
      </Tabs>
    </Paper>
  )
}

// TabActions holds a tab's actions, on the right like the page's own.
export function TabActions({ children }: { children: ReactNode }) {
  return <Group justify="flex-end">{children}</Group>
}
