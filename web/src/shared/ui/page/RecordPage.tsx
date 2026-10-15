import { Box, Flex, Group, ScrollArea, Tabs } from '@mantine/core'
import { useMediaQuery } from '@mantine/hooks'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { Page, readingWidth, type PageHeaderProps } from './Page'
import classes from './RecordPage.module.css'

// panel: a side panel (attachments), kept at its reading width.
export type RecordTab = { value: string; label: string; content: ReactNode; panel?: boolean }

// The tabs that the activity tab took over, so old links to them still land there.
const gathered = ['discussion', 'history']

type Props = Omit<PageHeaderProps, 'tabs'> & {
  tabs: RecordTab[]
  // The record's activity feed (comments and history), as the first tab.
  activity?: ReactNode
  // The record's own values, in a column left of the tabs from 1024px; below, the first tab. The
  // two columns then fill the window under the header and each scrolls on its own.
  aside?: { label: string; content: ReactNode }
}

// RecordPage: one catalog record (e.g. an employee). The selected tab is on the URL.
export function RecordPage({ tabs: own, activity, aside, ...header }: Props) {
  const { t } = useTranslation()
  const wide = useMediaQuery('(min-width: 64em)', true)
  const [search, setSearch] = useSearchParams()
  const activityTab: RecordTab[] = activity ? [{ value: 'activity', label: t('shared.document.tab.activity'), content: activity }] : []
  // What happened to the record leads; below 1024px its values come first, as they are not in view.
  const tabs: RecordTab[] = [...(aside && !wide ? [{ value: 'info', label: aside.label, content: aside.content }] : []), ...activityTab, ...own]
  const first = tabs[0]?.value ?? ''
  const asked = search.get('tab')
  const current = tabs.some((t) => t.value === asked) ? (asked as string) : activity && gathered.includes(asked ?? '') ? 'activity' : first
  const list = (
    <Tabs.List>
      {tabs.map((t) => (
        <Tabs.Tab key={t.value} value={t.value}>
          {t.label}
        </Tabs.Tab>
      ))}
    </Tabs.List>
  )
  const panels = tabs.map((t) => (
    // The tab line runs the page's width; what is under it is centred, like a form page, except
    // the activity feed, which reads from the tab line's left edge.
    <Tabs.Panel key={t.value} value={t.value} w="100%" maw={t.value === 'activity' ? undefined : 'var(--mmerp-content-width)'} mx="auto">
      {t.panel ? (
        <Box maw={readingWidth} mx="auto">
          {t.content}
        </Box>
      ) : (
        t.content
      )}
    </Tabs.Panel>
  ))
  return (
    <Tabs
      value={current}
      onChange={(v) => {
        const next = new URLSearchParams(search)
        if (!v || v === first) next.delete('tab')
        else next.set('tab', v)
        // Replaced, not pushed: Back leaves the record rather than walking back through its tabs.
        setSearch(next, { replace: true })
      }}
      keepMounted={false}
      className={aside && wide ? classes.root : undefined}
    >
      {aside && wide ? (
        <Page {...header} withBack plain className={classes.frame}>
          <Flex gap="xl" className={classes.columns}>
            <aside className={classes.aside} aria-label={aside.label}>
              <ScrollArea className={classes.fill} offsetScrollbars="y">
                <div className={classes.asideBody}>{aside.content}</div>
              </ScrollArea>
            </aside>
            <div className={classes.panel}>
              <div className={classes.tabs}>{list}</div>
              <ScrollArea className={classes.fill} offsetScrollbars="y">
                <div className={classes.scroll}>{panels}</div>
              </ScrollArea>
            </div>
          </Flex>
        </Page>
      ) : (
        <Page {...header} withBack tabs={list}>
          {panels}
        </Page>
      )}
    </Tabs>
  )
}

// TabActions holds a tab's actions, on the right like the page's own.
export function TabActions({ children }: { children: ReactNode }) {
  return <Group justify="flex-end">{children}</Group>
}
