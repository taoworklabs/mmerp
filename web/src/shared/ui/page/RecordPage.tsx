import { Group, Stack, Tabs } from '@mantine/core'
import type { ReactNode } from 'react'
import { useSearchParams } from 'react-router'
import { MoreActions } from '../MoreActions'
import type { DetailTab } from './DetailPage'
import { Page, type PageHeaderProps } from './Page'
import classes from './RecordPage.module.css'

type Props = Omit<PageHeaderProps, 'tabs' | 'withBack' | 'plain' | 'current'> & {
  // Menu.Item children: the rarer and the destructive actions, in a "…" menu after the buttons.
  more?: ReactNode
  // The record's parts, the first (its overview) open by default; a label may carry a count.
  tabs: DetailTab[]
  // Tab values of older links and the tab they now open.
  aliases?: Record<string, string>
  // Outside the tabs, e.g. a modal the header opens.
  children?: ReactNode
}

// RecordPage: one record with much to read (an employee). A summary header (breadcrumb ending with
// the page, title with status, key facts, actions on the right), then the record's parts as tabs on
// the header's rule, each the full width; the open tab is on the URL. Centred at the detail width.
export function RecordPage({ more, actions, tabs, aliases, children, ...header }: Props) {
  const [search, setSearch] = useSearchParams()
  const first = tabs[0]?.value ?? ''
  const asked = search.get('tab') ?? ''
  const wanted = aliases?.[asked] ?? asked
  const tab = tabs.some((x) => x.value === wanted) ? wanted : first
  const all = (actions || more) && (
    <Group gap="xs" justify="flex-end" wrap="nowrap">
      {actions}
      {more && <MoreActions place="page">{more}</MoreActions>}
    </Group>
  )
  return (
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
      <Page
        {...header}
        current
        actions={all}
        centered="var(--mmerp-detail-width)"
        tabs={
          <Tabs.List className={classes.tabs}>
            {tabs.map((x) => (
              <Tabs.Tab key={x.value} value={x.value}>
                {x.label}
              </Tabs.Tab>
            ))}
          </Tabs.List>
        }
      >
        {tabs.map((x) => (
          <Tabs.Panel key={x.value} value={x.value}>
            {x.content}
          </Tabs.Panel>
        ))}
        {children}
      </Page>
    </Tabs>
  )
}

// RecordColumns lays out a record's overview: its values on the left, digests of its parts on the
// right, about 1.6 to 1; one column below 1024px, the right one under the left.
export function RecordColumns({ main, side }: { main: ReactNode; side: ReactNode }) {
  return (
    <div className={classes.columns}>
      <Stack gap="md" miw={0}>
        {main}
      </Stack>
      <Stack gap="md" miw={0}>
        {side}
      </Stack>
    </div>
  )
}
