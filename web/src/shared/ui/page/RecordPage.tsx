import { Box, Group, Tabs } from '@mantine/core'
import type { ReactNode } from 'react'
import { useSearchParams } from 'react-router'
import { Page, readingWidth, type PageHeaderProps } from './Page'

// panel: a side panel of a document (attachments, discussion), kept at its reading width.
export type RecordTab = { value: string; label: string; content: ReactNode; panel?: boolean }

// RecordPage: one catalog record (e.g. an employee). The selected tab is on the URL.
export function RecordPage({ tabs, ...header }: PageHeaderProps & { tabs: RecordTab[] }) {
  const [search, setSearch] = useSearchParams()
  const first = tabs[0]?.value ?? ''
  const current = tabs.some((t) => t.value === search.get('tab')) ? (search.get('tab') as string) : first
  return (
    <Page {...header} divider={false}>
      <Tabs
        value={current}
        onChange={(v) => {
          const next = new URLSearchParams(search)
          if (!v || v === first) next.delete('tab')
          else next.set('tab', v)
          setSearch(next)
        }}
        keepMounted={false}
      >
        <Tabs.List>
          {tabs.map((t) => (
            <Tabs.Tab key={t.value} value={t.value}>
              {t.label}
            </Tabs.Tab>
          ))}
        </Tabs.List>
        {tabs.map((t) => (
          // The tab line runs the page's width; what is under it is centred, like a form page.
          <Tabs.Panel key={t.value} value={t.value} pt="lg" maw="var(--mmerp-content-width)" mx="auto">
            {t.panel ? (
              <Box maw={readingWidth} mx="auto">
                {t.content}
              </Box>
            ) : (
              t.content
            )}
          </Tabs.Panel>
        ))}
      </Tabs>
    </Page>
  )
}

// TabActions holds a tab's actions, on the right like the page's own.
export function TabActions({ children }: { children: ReactNode }) {
  return <Group justify="flex-end">{children}</Group>
}
