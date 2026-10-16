import { AppShell, Burger, Group, NavLink, ScrollArea, Stack, Text, ThemeIcon } from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { IconComponents } from '@tabler/icons-react'
import type { ComponentType } from 'react'
import { useTranslation } from 'react-i18next'
import { createBrowserRouter, Link, Navigate, Outlet, useLocation } from 'react-router'
import { client } from '@/shared/api/client'
import type { Me } from '@/shared/api/core'
import { MeProvider } from '@/shared/auth/me'
import { icon } from '@/shared/ui/theme'
import classes from '../AppLayout.module.css'
import { offline } from './api'
import devui from './DevUi.module.css'
import { Buttons, People, Sections, Signals, Tiles } from './basics'
import { Bulk, Grid, GroupedTable, Table } from './data'
import { Dialogs, States } from './feedback'
import { Colour, Icons, Principles, Spacing, Surfaces, Typography } from './foundations'
import { Fields, Filters, Inline, Lines } from './input'
import { DetailTemplate, DocumentTemplate, InboxTemplate, ListTemplate, PageTemplate } from './templates'

// An entry is labelled by an i18n key, or by its component name for the templates.
type Entry = { path: string; label?: string; name?: string; Component: ComponentType }

const groups: { label: string; entries: Entry[] }[] = [
  {
    label: 'shared.devui.group_foundations',
    entries: [
      { path: 'principles', label: 'shared.devui.principles', Component: Principles },
      { path: 'colour', label: 'shared.devui.colour', Component: Colour },
      { path: 'typography', label: 'shared.devui.typography', Component: Typography },
      { path: 'spacing', label: 'shared.devui.spacing', Component: Spacing },
      { path: 'surfaces', label: 'shared.devui.surfaces', Component: Surfaces },
      { path: 'icons', label: 'shared.devui.icons', Component: Icons },
    ],
  },
  {
    label: 'shared.devui.group_basics',
    entries: [
      { path: 'buttons', label: 'shared.devui.buttons', Component: Buttons },
      { path: 'signals', label: 'shared.devui.signals', Component: Signals },
      { path: 'people', label: 'shared.devui.people', Component: People },
      { path: 'tiles', label: 'shared.devui.tiles', Component: Tiles },
      { path: 'sections', label: 'shared.devui.section', Component: Sections },
    ],
  },
  {
    label: 'shared.devui.group_data',
    entries: [
      { path: 'table', label: 'shared.devui.table', Component: Table },
      { path: 'grouped', label: 'shared.devui.grouped', Component: GroupedTable },
      { path: 'grid', label: 'shared.devui.grid', Component: Grid },
      { path: 'bulk', label: 'shared.devui.bulk', Component: Bulk },
    ],
  },
  {
    label: 'shared.devui.group_input',
    entries: [
      { path: 'fields', label: 'shared.devui.form', Component: Fields },
      { path: 'filters', label: 'shared.devui.filters', Component: Filters },
      { path: 'inline', label: 'shared.devui.inline', Component: Inline },
      { path: 'lines', label: 'shared.devui.lines', Component: Lines },
    ],
  },
  {
    label: 'shared.devui.group_feedback',
    entries: [
      { path: 'states', label: 'shared.devui.states', Component: States },
      { path: 'dialogs', label: 'shared.devui.dialogs', Component: Dialogs },
    ],
  },
  {
    label: 'shared.devui.template',
    entries: [
      { path: 'page', name: 'Page', Component: PageTemplate },
      { path: 'list', name: 'ListPage', Component: ListTemplate },
      { path: 'record', name: 'DetailPage', Component: DetailTemplate },
      { path: 'document', name: 'DocumentPage', Component: DocumentTemplate },
      { path: 'inbox', name: 'InboxPage', Component: InboxTemplate },
    ],
  },
]

// The signed-in user the catalogue pretends to be: components such as the activity feed read it.
const me: Me = { authz_version: '0', id: 1, locale: 'vi', login: 'an.nguyen', name: 'Nguyễn Văn An', permissions: {}, products: [], products_with_data: [], timezone: 'Asia/Ho_Chi_Minh' }

// DevUiLayout stands in for the app shell: a sidebar of catalogue entries, no session and no API.
function DevUiLayout() {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  const [drawer, { toggle, close }] = useDisclosure()
  return (
    <MeProvider me={me}>
      <AppShell header={{ height: 48 }} navbar={{ width: 240, breakpoint: 'md', collapsed: { mobile: !drawer } }} padding={0}>
        <AppShell.Header px="md">
          <Group h="100%" gap="xs" wrap="nowrap">
            <Burger opened={drawer} onClick={toggle} hiddenFrom="md" size="sm" aria-label={t('core.shell.menu')} aria-expanded={drawer} />
            <ThemeIcon size={28} radius="sm" aria-hidden>
              <IconComponents {...icon.text} />
            </ThemeIcon>
            <Text fw={600} truncate>
              {t('shared.devui.title')}
            </Text>
          </Group>
        </AppShell.Header>
        <AppShell.Navbar>
          <AppShell.Section grow component={ScrollArea} p="xs">
            <Stack gap="md">
              {groups.map((g) => (
                <Stack key={g.label} gap="xxs" role="group" aria-label={t(g.label)}>
                  <Text size="xs" fw={600} px="xs">
                    {t(g.label)}
                  </Text>
                  <Stack gap={0} className={devui.entries}>
                    {g.entries.map((e) => (
                      <NavLink
                        key={e.path}
                        className={classes.item}
                        component={Link}
                        to={e.path}
                        label={e.name ?? t(e.label ?? '')}
                        active={pathname === `/dev/ui/${e.path}`}
                        variant="light"
                        onClick={close}
                      />
                    ))}
                  </Stack>
                </Stack>
              ))}
            </Stack>
          </AppShell.Section>
        </AppShell.Navbar>
        <AppShell.Main>
          <Outlet />
        </AppShell.Main>
      </AppShell>
    </MeProvider>
  )
}

// devUiRouter serves the catalogue under /dev/ui, outside the app and its session; called once.
export function devUiRouter() {
  client.use(offline)
  const entries = groups.flatMap((g) => g.entries)
  return createBrowserRouter([
    {
      path: '/dev/ui',
      Component: DevUiLayout,
      children: [{ index: true, element: <Navigate to={entries[0]!.path} replace /> }, ...entries.map((e) => ({ path: e.path, Component: e.Component }))],
    },
    { path: '*', element: <Navigate to="/dev/ui" replace /> },
  ])
}
