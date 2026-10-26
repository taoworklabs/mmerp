import { Anchor, Group, Stack } from '@mmerp/ui'
import { Tabs } from '@mantine/core'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useSearchParams } from 'react-router'
import { MoreActions } from '../MoreActions'
import { Notice } from '../Notice'
import { Page, tabsList, type PageHeaderProps } from './Page'

// A part of the record: its tab's value on the URL, label (a count once known) and content.
export type RecordTab = { value: string; label: string; content: ReactNode }

type Props = Omit<PageHeaderProps, 'tabs' | 'withBack' | 'plain' | 'current'> & {
  // Menu.Item children: the rarer and the destructive actions, in a "…" menu after the buttons.
  more?: ReactNode
  // The record's parts, the first (its overview) open by default; a label may carry a count.
  tabs: RecordTab[]
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
    <Group gap="xs" justify="end" wrap={false}>
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
        centered="detail"
        tabs={
          <Tabs.List className={tabsList}>
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

// SeeAll, on a digest's title line, opens the tab holding the whole of it.
export function SeeAll({ tab }: { tab: string }) {
  const { t } = useTranslation()
  return (
    <Anchor render={<Link to={`?tab=${tab}`} replace />} size="sm">
      {t('shared.record.see_all')}
    </Anchor>
  )
}

// TabContent is the body of a tab holding a list: its actions on the right like the page's own,
// the failure of the last action, then the list (and the tab's dialogs).
export function TabContent({ actions, error, children }: { actions?: ReactNode; error?: string | null; children: ReactNode }) {
  return (
    <Stack gap="md">
      {actions && <Group justify="end">{actions}</Group>}
      {error && <Notice tone="danger">{error}</Notice>}
      {children}
    </Stack>
  )
}
