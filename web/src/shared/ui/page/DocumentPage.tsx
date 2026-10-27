import { ActionBar, Panel, Tabs, useMediaQuery } from '@mmerp/ui'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { UnsavedProvider } from '../form'
import { Notice } from '../Notice'
import { Activity, type DocumentSection } from './Activity'
import { DetailPage } from './DetailPage'
import classes from './DocumentPage.module.css'
import { Page, type PageHeaderProps } from './Page'

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

// DocumentPage: one document. Header with number, status and actions; the document's values in
// the main column; approval, the sections and the activity in the side column (DetailPage), under
// the main one below 1024px, where the actions become a bar at the bottom. A full-width document
// moves the side column to tabs instead (details · approval · … · activity, on the URL).
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
    <Notice tone="danger">
      {error}
    </Notice>
  )

  if (!fullWidth)
    return (
      <DetailPage
        {...header}
        actions={actions}
        narrowActions={actions}
        banner={banner}
        side={side.map((p) => (
          <Panel key={p.key} plain title={p.title}>
            {p.content}
          </Panel>
        ))}
      >
        {children}
      </DetailPage>
    )

  // On a wide screen a full-width document gathers the side column in one tab, laid out like the
  // side column itself, rather than spreading a few lines over the whole width per tab.
  const gathered = wide
  const tabs: DocumentSection[] = gathered ? [{ key: 'activity', title: t('shared.document.tab.activity'), content: <Activity top={top} history={log} /> }] : side
  // The activity took over the discussion and history tabs, so old links to them still land there.
  const asked = ['discussion', 'history'].includes(search.get('tab') ?? '') ? 'activity' : search.get('tab')
  // A link to one of the gathered parts (?tab=approval) opens the tab that holds it.
  const tab = tabs.some((p) => p.key === asked) ? (asked as string) : gathered && side.some((p) => p.key === asked) ? 'activity' : 'details'
  return (
    <Tabs
      value={tab}
      onValueChange={(v) => {
        const next = new URLSearchParams(search)
        if (!v || v === 'details') next.delete('tab')
        else next.set('tab', v)
        // Replaced, not pushed: Back leaves the record rather than walking back through its tabs.
        setSearch(next, { replace: true })
      }}
    >
      <Page
        {...header}
        withBack
        actions={wide ? actions : undefined}
        tabs={
          <Tabs.List scroll onRule grow={!wide && tabs.length + 1 <= maxGrownTabs}>
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
        <Tabs.Panel value="details">{children}</Tabs.Panel>
        {tabs.map((p) => (
          <Tabs.Panel key={p.key} value={p.key} className={classes.sidePanel}>
            {p.content}
          </Tabs.Panel>
        ))}
        {actions && !wide && <ActionBar>{actions}</ActionBar>}
      </Page>
    </Tabs>
  )
}
