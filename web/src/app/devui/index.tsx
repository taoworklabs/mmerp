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
import { Alerts, Badges, Menus } from './basics'
import { type Entry, entries, layers, type Path } from './catalogue'
import { key, useEntryTitle } from './DocPage'
import { Overview } from './start'
import { Bulk } from './data'
import { CellGridDoc, DataTableDoc, FieldListDoc, KeyFactsDoc, PanelDoc, PersonAvatarDoc, SectionDoc, StatusBadgeDoc, TilesDoc, TreeListDoc } from './display'
import { AnchorDoc, ButtonDoc, LayoutTextDoc, LoaderDoc, SkeletonDoc, TooltipDoc } from './mantine'
import { ActivityDemo, Cards, DocumentPanels } from './blocks'
import { Actions, Dialogs, QueryViews, States, ToastsDemo } from './feedback'
import { Colour, Icons, Principles, Spacing, Surfaces, Typography } from './foundations'
import { Fields, Filters, Inline, Lines } from './input'
import { AuthTemplate, DocumentTemplate, InboxTemplate, ListTemplate, PageTemplate, RecordTemplate } from './templates'

// Every entry's page; a missing or extra one fails tsc.
const pages: Record<Path, ComponentType> = {
  'start/overview': Overview,
  'foundations/principles': Principles,
  'foundations/colour': Colour,
  'foundations/typography': Typography,
  'foundations/spacing': Spacing,
  'foundations/surfaces': Surfaces,
  'foundations/icons': Icons,
  'mantine/anchor': AnchorDoc,
  'mantine/button': ButtonDoc,
  'mantine/layout-text': LayoutTextDoc,
  'mantine/loader': LoaderDoc,
  'mantine/skeleton': SkeletonDoc,
  'mantine/tooltip': TooltipDoc,
  'mmerp/activity-feed': ActivityDemo,
  'mmerp/approval-panel': DocumentPanels,
  'mmerp/attachment-panel': DocumentPanels,
  'mmerp/bulk-bar': Bulk,
  'mmerp/cell-grid': CellGridDoc,
  'mmerp/data-table': DataTableDoc,
  'mmerp/document-preview': DocumentPanels,
  'mmerp/document-status': Badges,
  'mmerp/editable-panel': Cards,
  'mmerp/field-list': FieldListDoc,
  'mmerp/filters': Filters,
  'mmerp/form': Fields,
  'mmerp/form-modal': Dialogs,
  'mmerp/inline-edit-field': Inline,
  'mmerp/key-facts': KeyFactsDoc,
  'mmerp/lines-table': Lines,
  'mmerp/more-actions': Menus,
  'mmerp/notice': Alerts,
  'mmerp/notify': ToastsDemo,
  'mmerp/panel': PanelDoc,
  'mmerp/person-avatar': PersonAvatarDoc,
  'mmerp/query-view': QueryViews,
  'mmerp/section': SectionDoc,
  'mmerp/side-panel': Bulk,
  'mmerp/states': States,
  'mmerp/status-badge': StatusBadgeDoc,
  'mmerp/tiles': TilesDoc,
  'mmerp/tree-list': TreeListDoc,
  'mmerp/use-action': Actions,
  'mmerp/use-confirm': Dialogs,
  'patterns/filter-search': Filters,
  'patterns/create-edit': Fields,
  'patterns/confirm-delete': Actions,
  'patterns/loading-empty-error': QueryViews,
  'patterns/bulk': Bulk,
  'patterns/document-approval': DocumentPanels,
  'templates/auth': AuthTemplate,
  'templates/document': DocumentTemplate,
  'templates/inbox': InboxTemplate,
  'templates/list': ListTemplate,
  'templates/page': PageTemplate,
  'templates/record': RecordTemplate,
}

const all = entries as readonly Entry[]

// The signed-in user the catalogue pretends to be: components such as the activity feed read it.
const me: Me = { authz_version: '0', id: 1, locale: 'vi', login: 'an.nguyen', name: 'Nguyễn Văn An', permissions: {}, products: [], products_with_data: [], timezone: 'Asia/Ho_Chi_Minh' }

// Usage sits above a template, outside it: when to reach for it, and when another one fits better.
function Usage({ entry }: { entry: Entry }) {
  const k = key(entry)
  const { t } = useTranslation()
  return (
    <Stack gap="xxxs" px="md" py="xs" className={devui.usage}>
      <Text size="sm">
        <Text span fw={600}>
          {t('shared.devui.use_when')}
        </Text>{' '}
        {t(`shared.devui.use_${k}`)}
      </Text>
      <Text size="sm">
        <Text span fw={600}>
          {t('shared.devui.avoid_when')}
        </Text>{' '}
        {t(`shared.devui.avoid_${k}`)}
      </Text>
    </Stack>
  )
}

// DevUiLayout stands in for the app shell: a sidebar of catalogue entries, no session and no API.
function DevUiLayout() {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  const [drawer, { toggle, close }] = useDisclosure()
  const title = useEntryTitle()
  const template = all.find((e) => e.layer === 'templates' && pathname === `/dev/ui/${e.path}`)
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
              {layers.map((l, i) => (
                <Stack key={l} gap="xxs" role="group" aria-label={t(`shared.devui.layer_${l}`)}>
                  <Stack gap={0} px="xs">
                    <Text size="xs" fw={600}>
                      {`${i + 1} · ${t(`shared.devui.layer_${l}`)}`}
                    </Text>
                    <Text size="xs" c="dimmed">
                      {t(`shared.devui.layer_${l}_note`)}
                    </Text>
                  </Stack>
                  <Stack gap={0} className={devui.entries}>
                    {all
                      .filter((e) => e.layer === l)
                      .map((e) => (
                        <NavLink
                          key={e.path}
                          className={classes.item}
                          component={Link}
                          to={e.path}
                          label={title(e)}
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
          {template && <Usage entry={template} />}
          <Outlet />
        </AppShell.Main>
      </AppShell>
    </MeProvider>
  )
}

// devUiRouter serves the catalogue under /dev/ui, outside the app and its session; called once.
export function devUiRouter() {
  client.use(offline)
  return createBrowserRouter([
    {
      path: '/dev/ui',
      Component: DevUiLayout,
      children: [{ index: true, element: <Navigate to="start/overview" replace /> }, ...all.map((e) => ({ path: e.path, Component: pages[e.path as Path] }))],
    },
    { path: '*', element: <Navigate to="/dev/ui/start/overview" replace /> },
  ])
}
