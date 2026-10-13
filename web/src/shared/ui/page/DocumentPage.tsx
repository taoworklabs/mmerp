import { Alert, Box, Flex, Paper, Stack, Tabs, Text } from '@mantine/core'
import { useMediaQuery } from '@mantine/hooks'
import { IconAlertCircle } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { UnsavedProvider } from '../form'
import { icon } from '../theme'
import classes from './DocumentPage.module.css'
import { Page, readingWidth, type PageHeaderProps } from './Page'

// maxGrownTabs is how many tabs share a phone's width evenly; more scroll in one row.
const maxGrownTabs = 3

// DocumentSection is one more part of the side column (attachments, discussion), its title
// also the tab's label on a narrow screen.
export type DocumentSection = { key: string; title: string; content: ReactNode }

type Props = Omit<PageHeaderProps, 'actions' | 'leading'> & {
  // The document's lifecycle buttons (DocumentActions).
  actions?: ReactNode
  // A document-level error, e.g. period_locked, shown as a banner at the top.
  error?: string | null
  approval: ReactNode
  history: ReactNode
  // Shown between approval and history.
  sections?: DocumentSection[]
  // A wide table (payroll, timesheet) needs the whole width: approval and history become tabs
  // at every screen size, and the actions stay in the header.
  fullWidth?: boolean
  children: ReactNode
}

// DocumentPage: one document. Header with number, status and actions; the form in the
// main column; approval, the sections and history in a 360px side column. Below 1024px the
// side column becomes tabs (details · approval · … · history, on the URL) and the actions a
// bar at the bottom.
// The page knows whether its form holds unsaved edits, so its actions can ask to save first.
export function DocumentPage(props: Props) {
  return (
    <UnsavedProvider>
      <DocumentLayout {...props} />
    </UnsavedProvider>
  )
}

function DocumentLayout({ actions, error, approval, history, sections = [], fullWidth, children, ...header }: Props) {
  const { t } = useTranslation()
  const wide = useMediaQuery('(min-width: 64em)', true)
  const [search, setSearch] = useSearchParams()
  const side: DocumentSection[] = [
    { key: 'approval', title: t('shared.document.tab.approval'), content: approval },
    ...sections,
    { key: 'history', title: t('shared.document.tab.history'), content: history },
  ]
  const banner = error && (
    <Alert color="danger" icon={<IconAlertCircle {...icon.button} />} role="alert">
      {error}
    </Alert>
  )

  if (wide && !fullWidth)
    return (
      <Page {...header} actions={actions} centered="calc(var(--mmerp-content-width) + var(--mmerp-side-width) + var(--mantine-spacing-xl))">
        {banner}
        <Flex gap="xl" align="flex-start">
          <Box flex={1} miw={0}>
            {children}
          </Box>
          <Stack className={classes.side} gap="md">
            {side.map((p) => (
              <Panel key={p.key} title={p.title}>
                {p.content}
              </Panel>
            ))}
          </Stack>
        </Flex>
      </Page>
    )

  const tab = side.some((p) => p.key === search.get('tab')) ? (search.get('tab') as string) : 'details'
  return (
    <Page {...header} actions={wide ? actions : undefined}>
      {banner}
      <Tabs
        value={tab}
        onChange={(v) => {
          const next = new URLSearchParams(search)
          if (!v || v === 'details') next.delete('tab')
          else next.set('tab', v)
          setSearch(next)
        }}
        keepMounted={false}
      >
        <Tabs.List grow={!wide && side.length + 1 <= maxGrownTabs} className={classes.tabs}>
          <Tabs.Tab value="details">{t('shared.document.tab.details')}</Tabs.Tab>
          {side.map((p) => (
            <Tabs.Tab key={p.key} value={p.key}>
              {p.title}
            </Tabs.Tab>
          ))}
        </Tabs.List>
        {/* The tab line runs the page's width; what is under it is centred unless full width. */}
        <Tabs.Panel value="details" pt="md" maw={fullWidth ? undefined : 'var(--mmerp-content-width)'} mx="auto">
          {children}
        </Tabs.Panel>
        {/* On a wide screen the panels keep the side column's reading width. */}
        {side.map((p) => (
          <Tabs.Panel key={p.key} value={p.key} pt="md" maw={fullWidth ? undefined : 'var(--mmerp-content-width)'} mx="auto">
            <Box maw={wide ? readingWidth : undefined} mx="auto">
              {p.content}
            </Box>
          </Tabs.Panel>
        ))}
      </Tabs>
      {actions && !wide && (
        <>
          <div className={classes.barSpace} />
          <div className={classes.bar}>{actions}</div>
        </>
      )}
    </Page>
  )
}

// Panel frames one part of the side column (approval, attachments, history).
function Panel({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Paper withBorder p="md">
      <Stack gap="sm">
        <Text fw={600}>{title}</Text>
        {children}
      </Stack>
    </Paper>
  )
}
