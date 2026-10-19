import { Button, Group, Menu, Notification, Stack, Text } from '@mantine/core'
import { IconDownload, IconPencil, IconTrash } from '@tabler/icons-react'
import type { UseQueryResult } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiError } from '@/shared/api/error'
import { BulkBar } from '@/shared/ui/BulkBar'
import { useConfirm } from '@/shared/ui/confirm'
import { FieldList } from '@/shared/ui/FieldList'
import { MenuAction } from '@/shared/ui/MenuAction'
import { MoreActions } from '@/shared/ui/MoreActions'
import { Notice } from '@/shared/ui/Notice'
import { notifyError, notifySuccess, notifyUndo, UndoMessage } from '@/shared/ui/notify'
import { QueryView } from '@/shared/ui/QueryView'
import { SidePanel } from '@/shared/ui/SidePanel'
import { ContentSkeleton, EmptyState, ErrorState, ForbiddenPage, NotFoundPage, PageSkeleton, ReadOnlyBanner } from '@/shared/ui/states'
import { useAction } from '@/shared/ui/useAction'
import { Case, rows, type Row } from './Demo'
import devui from './DevUi.module.css'
import { DocPage, DoDont } from './DocPage'

// The main props of each component; data, not interface text.
const code = {
  moreActions: `<MoreActions place="page" | "row">
  <MenuAction icon onClick>{label}</MenuAction>
  <Menu.Divider />
  <MenuAction icon danger onClick>{label}</MenuAction>
</MoreActions>`,
  notice: `<Notice tone="info" | "success" | "warning" | "danger" | "discovery">{message}</Notice>`,
  notify: `notifySuccess(message)
notifyError(message)
notifyUndo(message, undoLabel, onUndo)`,
  states: `<EmptyState title description? action? icon? />
<ErrorState message onRetry? />
<PageSkeleton />  <ContentSkeleton />  <ReadOnlyBanner />
<NotFoundPage back? />  <ForbiddenPage />`,
  queryView: `<QueryView query={useQuery(…)} select? empty noResults? notFound?>
  {(data) => <DataTable rows={data} … />}
</QueryView>`,
  useAction: `const { run, error, dialog } = useAction()
run(() => api.call(), { confirm?: { title, message, confirmLabel, danger? }, success?: message })
{error && <Notice tone="danger">{error}</Notice>}
{dialog}`,
  useConfirm: `const [ask, dialog] = useConfirm()
if (await ask({ title, message, confirmLabel, danger? })) { … }
{dialog}`,
  bulkBar: `<BulkBar count={selected.size} onClear={clear}>{bulk actions}</BulkBar>`,
  sidePanel: `const peek = usePeek()   // the open row lives on the URL
<SidePanel title href opened onClose>{preview}</SidePanel>`,
}

export const queryStates = ['loading', 'error', 'empty', 'no_results', 'not_found', 'data'] as const
export type QueryCase = (typeof queryStates)[number]

// fakeQuery stands for a query in each state QueryView tells apart, without fetching anything.
export function fakeQuery(state: QueryCase): UseQueryResult<Row[]> {
  const error = state === 'error' ? new ApiError(503, 'network_error') : state === 'not_found' ? new ApiError(404, 'not_found') : null
  const data = state === 'empty' ? [] : rows
  return { isPending: state === 'loading', isError: error !== null, error, data: error ? undefined : data, refetch: async () => undefined } as unknown as UseQueryResult<Row[]>
}

function Items({ dangerFirst }: { dangerFirst?: boolean }) {
  const { t } = useTranslation()
  const danger = (
    <MenuAction icon={IconTrash} danger onClick={() => {}}>
      {t('shared.devui.danger')}
    </MenuAction>
  )
  return (
    <>
      {dangerFirst && danger}
      <MenuAction icon={IconDownload} onClick={() => {}}>
        {t('shared.devui.export')}
      </MenuAction>
      <MenuAction icon={IconPencil} onClick={() => {}}>
        {t('shared.devui.edit')}
      </MenuAction>
      {!dangerFirst && (
        <>
          <Menu.Divider />
          {danger}
        </>
      )}
    </>
  )
}

export function MoreActionsDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mmerp/more-actions"
      example={
        <Case>
          <Group gap="xs">
            <Button variant="default">{t('shared.devui.secondary')}</Button>
            <MoreActions place="page">
              <Items />
            </MoreActions>
            <Button>{t('shared.devui.primary')}</Button>
          </Group>
        </Case>
      }
      variants={
        <Case label={t('shared.devui.case_menu_row')}>
          <Group gap="xs">
            <MoreActions>
              <Items />
            </MoreActions>
          </Group>
        </Case>
      }
      doDont={
        <DoDont
          good={
            <Group>
              <MoreActions place="page">
                <Items />
              </MoreActions>
            </Group>
          }
          bad={
            <>
              <Group>
                <MoreActions place="page">
                  <Items dangerFirst />
                </MoreActions>
              </Group>
              <Text size="sm" c="dimmed">
                {t('shared.devui.dont_danger_first')}
              </Text>
            </>
          }
        />
      }
      code={code.moreActions}
    />
  )
}

const tones = ['info', 'success', 'warning', 'danger', 'discovery'] as const

export function NoticeDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mmerp/notice"
      example={
        <Case>
          <Notice tone="warning">{t('shared.devui.alert_warning')}</Notice>
        </Case>
      }
      variants={tones.map((tone) => (
        <Case key={tone} label={t(`shared.devui.case_tone_${tone}`)}>
          <Notice tone={tone}>{t(`shared.devui.alert_${tone}`)}</Notice>
        </Case>
      ))}
      code={code.notice}
    />
  )
}

// Each case shows the toast at rest, then a button that raises the real one.
export function NotifyDoc() {
  const { t } = useTranslation()
  const undo = () => notifySuccess(t('shared.devui.toast_undone'))
  return (
    <DocPage
      path="mmerp/notify"
      example={
        <Case>
          <Notification color="success" className={devui.toast}>
            {t('shared.devui.toast_saved')}
          </Notification>
          <Group>
            <Button variant="default" onClick={() => notifySuccess(t('shared.devui.toast_saved'))}>
              {t('shared.devui.show_toast')}
            </Button>
          </Group>
        </Case>
      }
      variants={
        <>
          <Case label={t('shared.devui.case_toast_error')}>
            <Notification color="danger" className={devui.toast}>
              {t('shared.devui.toast_job_failed')}
            </Notification>
            <Group>
              <Button variant="default" onClick={() => notifyError(t('shared.devui.toast_job_failed'))}>
                {t('shared.devui.show_toast')}
              </Button>
            </Group>
          </Case>
          <Case label={t('shared.devui.case_toast_undo')}>
            <Notification color="success" className={devui.toast}>
              <UndoMessage message={t('shared.devui.undo_done')} undoLabel={t('shared.undo')} onUndo={undo} />
            </Notification>
            <Group>
              <Button variant="default" onClick={() => notifyUndo(t('shared.devui.undo_done'), t('shared.undo'), undo)}>
                {t('shared.devui.show_toast')}
              </Button>
            </Group>
          </Case>
        </>
      }
      code={code.notify}
    />
  )
}

export function StatesDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mmerp/states"
      example={
        <Case>
          <EmptyState title={t('shared.devui.empty')} description={t('shared.devui.empty_description')} />
        </Case>
      }
      variants={
        <>
          <Case label={t('shared.devui.case_page_skeleton')}>
            <PageSkeleton />
          </Case>
          <Case label={t('shared.devui.case_content_skeleton')}>
            <ContentSkeleton />
          </Case>
          <Case label={t('shared.devui.case_read_only')}>
            <ReadOnlyBanner />
          </Case>
          <Case label={t('shared.devui.case_no_results')}>
            <EmptyState title={t('shared.devui.no_results')} action={<Button variant="default">{t('shared.list.clear_filters')}</Button>} />
          </Case>
          <Case label={t('shared.devui.case_error')}>
            <ErrorState message={t('shared.error.network_error')} onRetry={() => {}} />
          </Case>
          <Case label={t('shared.devui.case_not_found')}>
            <NotFoundPage back={{ to: '/dev/ui', label: t('shared.devui.back') }} />
          </Case>
          <Case label={t('shared.devui.case_forbidden')}>
            <ForbiddenPage />
          </Case>
        </>
      }
      code={code.states}
    />
  )
}

function StaticQuery({ state }: { state: QueryCase }) {
  const { t } = useTranslation()
  return (
    <QueryView query={fakeQuery(state)} empty={<EmptyState title={t('shared.devui.empty')} />}>
      {(data) => <Text size="sm">{data.map((r) => r.code).join(', ')}</Text>}
    </QueryView>
  )
}

export function QueryViewDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mmerp/query-view"
      example={
        <Case>
          <StaticQuery state="empty" />
        </Case>
      }
      variants={
        <>
          <Case label={t('shared.devui.query_loading')}>
            <StaticQuery state="loading" />
          </Case>
          <Case label={t('shared.devui.query_data')}>
            <StaticQuery state="data" />
          </Case>
        </>
      }
      code={code.queryView}
    />
  )
}

// UseActionDoc shows useAction: confirm first, the failure kept beside the region, success as a toast.
export function UseActionDoc() {
  const { t } = useTranslation()
  const { run, error, dialog } = useAction()
  return (
    <DocPage
      path="mmerp/use-action"
      example={
        <Case>
          <Stack gap="md">
            <Group justify="flex-end">
              <Button variant="default" onClick={() => void run(async () => undefined, { success: t('shared.devui.toast_test_sent') })}>
                {t('shared.devui.send_test')}
              </Button>
              <Button
                variant="subtle"
                color="danger"
                onClick={() =>
                  void run(
                    async () => {
                      throw new ApiError(503, 'network_error')
                    },
                    { confirm: { title: t('shared.devui.confirm'), message: t('shared.devui.confirm_message'), confirmLabel: t('shared.devui.danger'), danger: true } },
                  )
                }
              >
                {t('shared.devui.danger')}
              </Button>
            </Group>
            {error && <Notice tone="danger">{error}</Notice>}
            <Text size="sm" c="dimmed">
              {t('shared.devui.use_action_region')}
            </Text>
          </Stack>
          {dialog}
        </Case>
      }
      code={code.useAction}
    />
  )
}

export function UseConfirmDoc() {
  const { t } = useTranslation()
  const [ask, dialog] = useConfirm()
  return (
    <DocPage
      path="mmerp/use-confirm"
      example={
        <Case>
          <Group>
            <Button
              variant="default"
              onClick={() => void ask({ title: t('shared.devui.confirm'), message: t('shared.devui.confirm_message'), confirmLabel: t('shared.devui.danger'), danger: true })}
            >
              {t('shared.devui.confirm')}
            </Button>
          </Group>
          {dialog}
        </Case>
      }
      doDont={<DoDont good={<Text size="sm">{t('shared.devui.do_confirm_verb')}</Text>} bad={<Text size="sm">{t('shared.devui.dont_confirm_yes')}</Text>} />}
      code={code.useConfirm}
    />
  )
}

export function BulkBarDoc() {
  const { t } = useTranslation()
  const [count, setCount] = useState(2)
  return (
    <DocPage
      path="mmerp/bulk-bar"
      example={
        <Case>
          {count > 0 ? (
            <BulkBar count={count} onClear={() => setCount(0)}>
              <Button variant="default">{t('shared.devui.export')}</Button>
            </BulkBar>
          ) : (
            <Group>
              <Button variant="default" onClick={() => setCount(2)}>
                {t('shared.devui.bulk_select')}
              </Button>
            </Group>
          )}
        </Case>
      }
      code={code.bulkBar}
    />
  )
}

export function SidePanelDoc() {
  const { t } = useTranslation()
  const [opened, setOpened] = useState(false)
  const row = rows[0]!
  return (
    <DocPage
      path="mmerp/side-panel"
      example={
        <Case>
          <Group>
            <Button variant="default" onClick={() => setOpened(true)}>
              {t('shared.devui.open_panel')}
            </Button>
          </Group>
          <SidePanel title={row.name} href="/dev/ui/mmerp/side-panel" opened={opened} onClose={() => setOpened(false)}>
            <FieldList rows={[[t('shared.devui.code'), row.code]]} />
          </SidePanel>
        </Case>
      }
      code={code.sidePanel}
    />
  )
}
