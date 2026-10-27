import { Anchor, AppShell, Burger, Group, NavLink, Stack, Text, ThemeIcon, useDisclosure } from '@mmerp/ui'
import { IconComponents } from '@tabler/icons-react'
import { useEffect, type ComponentType, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { createBrowserRouter, Link, Navigate, Outlet, useLocation } from 'react-router'
import { client } from '@/shared/api/client'
import type { Me } from '@/shared/api/core'
import { MeProvider } from '@/shared/auth/me'
import { icon } from '@/shared/ui/theme'
import classes from '../AppLayout.module.css'
import { offline } from './api'
import devui from './DevUi.module.css'
import { type Entry, entries, entryByPath, key, layers, type Path } from './catalogue'
import { Linked, useEntryTitle } from './DocPage'
import { Overview } from './start'
import { ApprovalPanelDoc, ActivityFeedDoc, AttachmentPanelDoc, DocumentPreviewDoc, DocumentStatusDoc } from './documents'
import { BulkBarDoc, MoreActionsDoc, NoticeDoc, NotifyDoc, QueryViewDoc, SidePanelDoc, StatesDoc, UseActionDoc, UseConfirmDoc } from './actions'
import { BulkPattern, ConfirmDeletePattern, CreateEditPattern, DocumentApprovalPattern, FilterSearchPattern, LoadingEmptyErrorPattern } from './patterns'
import { EditablePanelDoc, FiltersDoc, FormDoc, FormModalDoc, InlineEditFieldDoc, LinesTableDoc } from './forms'
import { CellGridDoc, DataTableDoc, FieldListDoc, KeyFactsDoc, PanelDoc, PersonAvatarDoc, SectionDoc, StatusBadgeDoc, TilesDoc, TreeListDoc } from './display'
import { AnchorDoc, ButtonDoc, LayoutTextDoc, LoaderDoc, SkeletonDoc, TooltipDoc } from './kit'
import { Colour, Icons, Principles, Spacing, Surfaces, Typography } from './foundations'
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
  'kit/anchor': AnchorDoc,
  'kit/button': ButtonDoc,
  'kit/layout-text': LayoutTextDoc,
  'kit/loader': LoaderDoc,
  'kit/skeleton': SkeletonDoc,
  'kit/tooltip': TooltipDoc,
  'mmerp/activity-feed': ActivityFeedDoc,
  'mmerp/approval-panel': ApprovalPanelDoc,
  'mmerp/attachment-panel': AttachmentPanelDoc,
  'mmerp/bulk-bar': BulkBarDoc,
  'mmerp/cell-grid': CellGridDoc,
  'mmerp/data-table': DataTableDoc,
  'mmerp/document-preview': DocumentPreviewDoc,
  'mmerp/document-status': DocumentStatusDoc,
  'mmerp/editable-panel': EditablePanelDoc,
  'kit/field-list': FieldListDoc,
  'mmerp/filters': FiltersDoc,
  'mmerp/form': FormDoc,
  'mmerp/form-modal': FormModalDoc,
  'mmerp/inline-edit-field': InlineEditFieldDoc,
  'kit/key-facts': KeyFactsDoc,
  'mmerp/lines-table': LinesTableDoc,
  'mmerp/more-actions': MoreActionsDoc,
  'mmerp/notice': NoticeDoc,
  'mmerp/notify': NotifyDoc,
  'kit/panel': PanelDoc,
  'mmerp/person-avatar': PersonAvatarDoc,
  'mmerp/query-view': QueryViewDoc,
  'mmerp/section': SectionDoc,
  'mmerp/side-panel': SidePanelDoc,
  'mmerp/states': StatesDoc,
  'mmerp/status-badge': StatusBadgeDoc,
  'mmerp/tiles': TilesDoc,
  'mmerp/tree-list': TreeListDoc,
  'mmerp/use-action': UseActionDoc,
  'mmerp/use-confirm': UseConfirmDoc,
  'patterns/filter-search': FilterSearchPattern,
  'patterns/create-edit': CreateEditPattern,
  'patterns/confirm-delete': ConfirmDeletePattern,
  'patterns/loading-empty-error': LoadingEmptyErrorPattern,
  'patterns/bulk': BulkPattern,
  'patterns/document-approval': DocumentApprovalPattern,
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

// TemplateNote sits above a template, outside it: the problem it solves, when to reach for it and
// when another fits better, and the components it is built from.
function TemplateNote({ entry }: { entry: Entry }) {
  const k = key(entry)
  const { t } = useTranslation()
  const title = useEntryTitle()
  const line = (label: string, text: ReactNode) => (
    <Text size="sm">
      <Text render={<span />} weight={600}>
        {label}
      </Text>{' '}
      {text}
    </Text>
  )
  return (
    <Stack gap="xxxs" className={devui.usage}>
      <Text size="sm">{t(`shared.devui.about_${k}`)}</Text>
      {line(t('shared.devui.use_when'), t(`shared.devui.use_${k}`))}
      {line(t('shared.devui.avoid_when'), <Linked text={t(`shared.devui.avoid_${k}`)} />)}
      {entry.related &&
        line(
          t('shared.devui.built_from'),
          <Group render={<span />} gap="xs" className={devui.related}>
            {entry.related.map((p) => (
              <Anchor key={p} render={<Link to={`/dev/ui/${p}`} />} size="sm">
                {title(entryByPath(p))}
              </Anchor>
            ))}
          </Group>,
        )}
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
  // Each page opens at its top; opening a side panel (a search param) keeps the scroll.
  useEffect(() => {
    document.querySelector('main')?.scrollTo(0, 0)
  }, [pathname])
  return (
    <MeProvider me={me}>
      <AppShell
        navbarWidth={280}
        navbarOpened={drawer}
        header={
          <Group gap="xs" wrap={false} className={devui.bar}>
            <Burger opened={drawer} onClick={toggle} className={classes.burger} aria-label={t('core.shell.menu')} />
            <ThemeIcon size={28} aria-hidden>
              <IconComponents {...icon.text} />
            </ThemeIcon>
            <Text weight={600} truncate>
              {t('shared.devui.title')}
            </Text>
          </Group>
        }
        navbar={
          <Stack gap="md">
            {layers.map((l, i) => (
              <Stack key={l} gap="xxs" role="group" aria-label={t(`shared.devui.layer_${l}`)}>
                <Stack gap={0} className={devui.layerHead}>
                  <Text size="xs" weight={600}>
                    {`${i + 1} · ${t(`shared.devui.layer_${l}`)}`}
                  </Text>
                  <Text size="xs" tone="dimmed">
                    {t(`shared.devui.layer_${l}_note`)}
                  </Text>
                </Stack>
                <Stack gap={0} className={devui.entries}>
                  {all
                    .filter((e) => e.layer === l)
                    .map((e) => (
                      <NavLink key={e.path} compact render={<Link to={e.path} />} label={title(e)} active={pathname === `/dev/ui/${e.path}`} onClick={close} />
                    ))}
                </Stack>
              </Stack>
            ))}
          </Stack>
        }
      >
        {template && <TemplateNote entry={template} />}
        <Outlet />
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
