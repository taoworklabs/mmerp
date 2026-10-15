import { Alert, Box, Flex, Stack, Tabs } from '@mantine/core'
import { useMediaQuery } from '@mantine/hooks'
import { IconAlertCircle } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { UnsavedProvider } from '../form'
import { icon } from '../theme'
import { Activity, Panel, type DocumentSection } from './Activity'
import classes from './DocumentPage.module.css'
import { Page, readingWidth, type PageHeaderProps } from './Page'

// maxGrownTabs is how many tabs share a phone's width evenly; more scroll in one row.
const maxGrownTabs = 3

export type { DocumentSection }

type Props = Omit<PageHeaderProps, 'actions' | 'leading' | 'tabs'> & {
  // The document's lifecycle buttons (DocumentActions).
  actions?: ReactNode
  // A document-level error, e.g. period_locked, shown as a banner at the top.
  error?: string | null
  approval: ReactNode
  // The document's activity feed (comments and history, ActivityFeed), last.
  activity: ReactNode
  // Shown between approval and the activity (attachments).
  sections?: DocumentSection[]
  // A wide table (payroll, timesheet) needs the whole width: the side column moves to tabs at
  // every screen size (on a wide screen one "activity" tab holding all of it), and the actions
  // stay in the header.
  fullWidth?: boolean
  children: ReactNode
}

// DocumentPage: one document. Header with number, status and actions; the form in the
// main column; approval, the sections and the activity in a side column. Below 1024px the
// side column becomes tabs (details · approval · … · activity, on the URL) and the actions a
// bar at the bottom.
// The page knows whether its form holds unsaved edits, so its actions can ask to save first.
export function DocumentPage(props: Props) {
  return (
    <UnsavedProvider>
      <DocumentLayout {...props} />
    </UnsavedProvider>
  )
}

function DocumentLayout({ actions, error, approval, activity, sections = [], fullWidth, children, ...header }: Props) {
  const { t } = useTranslation()
  const wide = useMediaQuery('(min-width: 64em)', true)
  const [search, setSearch] = useSearchParams()
  const top: DocumentSection[] = [{ key: 'approval', title: t('shared.document.tab.approval'), content: approval }, ...sections]
  const log: DocumentSection = { key: 'activity', title: t('shared.document.tab.activity'), content: activity }
  const side = [...top, log]
  const banner = error && (
    <Alert color="danger" icon={<IconAlertCircle {...icon.button} />} role="alert">
      {error}
    </Alert>
  )

  if (wide && !fullWidth)
    return (
      <Page {...header} withBack actions={actions} centered="calc(var(--mmerp-content-width) + var(--mmerp-aside-width) + var(--mantine-spacing-xl))">
        {banner}
        <Flex gap="xl" align="flex-start">
          <Box flex={1} miw={0}>
            {children}
          </Box>
          <Stack className={classes.side} gap="md">
            {side.map((p) => (
              <Panel key={p.key} title={p.title} plain={p === log}>
                {p.content}
              </Panel>
            ))}
          </Stack>
        </Flex>
      </Page>
    )

  // On a wide screen a full-width document gathers the side column in one tab, laid out like the
  // side column itself, rather than spreading a few lines over the whole width per tab.
  const gathered = wide && fullWidth
  const tabs: DocumentSection[] = gathered ? [{ key: 'activity', title: t('shared.document.tab.activity'), content: <Activity top={top} history={log} /> }] : side
  // The activity took over the discussion and history tabs, so old links to them still land there.
  const asked = ['discussion', 'history'].includes(search.get('tab') ?? '') ? 'activity' : search.get('tab')
  // A link to one of the gathered parts (?tab=approval) opens the tab that holds it.
  const tab = tabs.some((p) => p.key === asked) ? (asked as string) : gathered && side.some((p) => p.key === asked) ? 'activity' : 'details'
  return (
    <Tabs
      value={tab}
      onChange={(v) => {
        const next = new URLSearchParams(search)
        if (!v || v === 'details') next.delete('tab')
        else next.set('tab', v)
        // Replaced, not pushed: Back leaves the record rather than walking back through its tabs.
        setSearch(next, { replace: true })
      }}
      keepMounted={false}
    >
      <Page
        {...header}
        withBack
        actions={wide ? actions : undefined}
        tabs={
          <Tabs.List grow={!wide && tabs.length + 1 <= maxGrownTabs} className={classes.tabs}>
            <Tabs.Tab value="details">{t('shared.document.tab.details')}</Tabs.Tab>
            {tabs.map((p) => (
              <Tabs.Tab key={p.key} value={p.key}>
                {p.title}
              </Tabs.Tab>
            ))}
          </Tabs.List>
        }
      >
        {banner}
        {/* The tab line runs the page's width; what is under it is centred unless full width. */}
        <Tabs.Panel value="details" w="100%" maw={fullWidth ? undefined : 'var(--mmerp-content-width)'} mx="auto">
          {children}
        </Tabs.Panel>
        {/* On a wide screen a single part keeps the side column's reading width. */}
        {tabs.map((p) => (
          <Tabs.Panel key={p.key} value={p.key} w="100%" maw="var(--mmerp-content-width)" mx="auto">
            <Box maw={wide && !gathered ? readingWidth : undefined} mx="auto">
              {p.content}
            </Box>
          </Tabs.Panel>
        ))}
        {actions && !wide && (
          <>
            <div className={classes.barSpace} />
            <div className={classes.bar}>{actions}</div>
          </>
        )}
      </Page>
    </Tabs>
  )
}
