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
import { Alerts, Avatars, Badges, Buttons, FieldLists, IconButtons, Links, Menus, Overlays, Sections, TabsDemo, Tiles, Tree } from './basics'
import { Bulk, Grid, GroupedTable, Table } from './data'
import { ActivityDemo, Cards, DocumentPanels, Headers } from './blocks'
import { Actions, Dialogs, QueryViews, States, ToastsDemo } from './feedback'
import { Colour, Icons, Principles, Spacing, Surfaces, Typography } from './foundations'
import { Fields, Filters, Inline, Lines } from './input'
import { AuthTemplate, DocumentTemplate, InboxTemplate, ListTemplate, PageTemplate, RecordTemplate } from './templates'

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
    label: 'shared.devui.group_actions',
    entries: [
      { path: 'buttons', label: 'shared.devui.buttons', Component: Buttons },
      { path: 'icon-buttons', label: 'shared.devui.icon_buttons', Component: IconButtons },
      { path: 'menus', label: 'shared.devui.menus', Component: Menus },
    ],
  },
  {
    label: 'shared.devui.group_navigation',
    entries: [
      { path: 'links', label: 'shared.devui.links', Component: Links },
      { path: 'tabs', label: 'shared.devui.tabs', Component: TabsDemo },
    ],
  },
  {
    label: 'shared.devui.group_display',
    entries: [
      { path: 'badges', label: 'shared.devui.badges', Component: Badges },
      { path: 'avatars', label: 'shared.devui.avatars', Component: Avatars },
      { path: 'field-list', label: 'shared.devui.field_list', Component: FieldLists },
      { path: 'tree', label: 'shared.devui.tree', Component: Tree },
      { path: 'tiles', label: 'shared.devui.tiles', Component: Tiles },
    ],
  },
  {
    label: 'shared.devui.group_tables',
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
      { path: 'alerts', label: 'shared.devui.alerts', Component: Alerts },
      { path: 'states', label: 'shared.devui.states', Component: States },
      { path: 'query-view', label: 'shared.devui.query_view', Component: QueryViews },
      { path: 'toasts', label: 'shared.devui.toasts', Component: ToastsDemo },
      { path: 'use-action', label: 'shared.devui.use_action', Component: Actions },
    ],
  },
  {
    label: 'shared.devui.group_overlays',
    entries: [
      { path: 'overlays', label: 'shared.devui.overlays', Component: Overlays },
      { path: 'dialogs', label: 'shared.devui.dialogs', Component: Dialogs },
    ],
  },
  {
    label: 'shared.devui.group_blocks',
    entries: [
      { path: 'headers', label: 'shared.devui.headers', Component: Headers },
      { path: 'cards', label: 'shared.devui.cards', Component: Cards },
      { path: 'activity', label: 'shared.devui.activity', Component: ActivityDemo },
      { path: 'document-panels', label: 'shared.devui.doc_panels', Component: DocumentPanels },
    ],
  },
  {
    label: 'shared.devui.group_layout',
    entries: [
      { path: 'sections', label: 'shared.devui.section', Component: Sections },
      { path: 'page', name: 'Page', Component: PageTemplate },
      { path: 'list', name: 'ListPage', Component: ListTemplate },
      { path: 'record', name: 'RecordPage', Component: RecordTemplate },
      { path: 'document', name: 'DocumentPage', Component: DocumentTemplate },
      { path: 'inbox', name: 'InboxPage', Component: InboxTemplate },
      { path: 'auth', name: 'AuthPage', Component: AuthTemplate },
    ],
  },
]

// The signed-in user the catalogue pretends to be: components such as the activity feed read it.
const me: Me = { authz_version: '0', id: 1, locale: 'vi', login: 'an.nguyen', name: 'Nguyễn Văn An', permissions: {}, products: [], products_with_data: [], timezone: 'Asia/Ho_Chi_Minh' }

// Usage sits above a template, outside it: when to reach for it, and when another one fits better.
function Usage({ path }: { path: string }) {
  const { t } = useTranslation()
  return (
    <Stack gap="xxxs" px="md" py="xs" className={devui.usage}>
      <Text size="sm">
        <Text span fw={600}>
          {t('shared.devui.use_when')}
        </Text>{' '}
        {t(`shared.devui.use_${path}`)}
      </Text>
      <Text size="sm">
        <Text span fw={600}>
          {t('shared.devui.avoid_when')}
        </Text>{' '}
        {t(`shared.devui.avoid_${path}`)}
      </Text>
    </Stack>
  )
}

// DevUiLayout stands in for the app shell: a sidebar of catalogue entries, no session and no API.
function DevUiLayout() {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  const [drawer, { toggle, close }] = useDisclosure()
  const template = groups.flatMap((g) => g.entries).find((e) => e.name && pathname === `/dev/ui/${e.path}`)
  return (
    <MeProvider me={me}>
      <AppShell header={{ height: 48 }} navbar={{ width: 280, breakpoint: 'md', collapsed: { mobile: !drawer } }} padding={0}>
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
          {template && <Usage path={template.path} />}
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
