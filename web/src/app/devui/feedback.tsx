import { Button, Group, Notification, SegmentedControl, Stack, Text } from '@mantine/core'
import type { UseQueryResult } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiError } from '@/shared/api/error'
import { formatNumber } from '@/shared/i18n'
import { useConfirm } from '@/shared/ui/confirm'
import { DataTable } from '@/shared/ui/DataTable'
import { ModalFooter } from '@/shared/ui/ModalFooter'
import { Notice } from '@/shared/ui/Notice'
import { notifyError, notifySuccess, notifyUndo, UndoMessage } from '@/shared/ui/notify'
import { QueryView } from '@/shared/ui/QueryView'
import { FormModal } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState, ForbiddenPage, NotFoundPage, PageSkeleton, ReadOnlyBanner } from '@/shared/ui/states'
import { useAction } from '@/shared/ui/useAction'
import { Case, Demo, rows, type Row } from './Demo'
import devui from './DevUi.module.css'

export function States() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.states')}>
      <Case label={t('shared.devui.case_page_skeleton')}>
        <PageSkeleton />
      </Case>
      <Case label={t('shared.devui.case_content_skeleton')}>
        <ContentSkeleton />
      </Case>
      <Case label={t('shared.devui.case_read_only')}>
        <ReadOnlyBanner />
      </Case>
      <Case label={t('shared.devui.case_empty')}>
        <EmptyState title={t('shared.devui.empty')} description={t('shared.devui.empty_description')} />
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
    </Demo>
  )
}

const queryStates = ['loading', 'error', 'empty', 'no_results', 'not_found', 'data'] as const
type QueryCase = (typeof queryStates)[number]

// fakeQuery stands for a query in each state QueryView tells apart, without fetching anything.
function fakeQuery(state: QueryCase): UseQueryResult<Row[]> {
  const error = state === 'error' ? new ApiError(503, 'network_error') : state === 'not_found' ? new ApiError(404, 'not_found') : null
  const data = state === 'empty' ? [] : rows
  return { isPending: state === 'loading', isError: error !== null, error, data: error ? undefined : data, refetch: async () => undefined } as unknown as UseQueryResult<Row[]>
}

export function QueryViews() {
  const { t } = useTranslation()
  const [state, setState] = useState<QueryCase>('data')
  return (
    <Demo title={t('shared.devui.query_view')} description={t('shared.devui.query_view_hint')}>
      <Case>
        <SegmentedControl w="fit-content" value={state} onChange={(v) => setState(v as QueryCase)} data={queryStates.map((s) => ({ value: s, label: t(`shared.devui.query_${s}`) }))} />
        <QueryView
          query={fakeQuery(state)}
          select={(data) => (state === 'no_results' ? [] : data)}
          empty={<EmptyState title={t('shared.devui.empty')} />}
          noResults={<EmptyState title={t('shared.devui.no_results')} action={<Button variant="default">{t('shared.list.clear_filters')}</Button>} />}
          notFound={<EmptyState title={t('shared.notfound.description')} />}
        >
          {(data) => (
            <DataTable
              label={t('shared.devui.table')}
              rows={data}
              rowKey={(r) => r.id}
              columns={[
                { key: 'code', header: t('shared.devui.code'), role: 'title', kind: 'code', render: (r) => r.code },
                { key: 'name', header: t('shared.devui.name'), role: 'meta', render: (r) => r.name },
                { key: 'amount', header: t('shared.devui.amount'), role: 'meta', kind: 'number', render: (r) => formatNumber(r.amount) },
              ]}
            />
          )}
        </QueryView>
      </Case>
    </Demo>
  )
}

// Actions shows useAction: confirm first, the failure kept beside the region, success as a toast.
export function Actions() {
  const { t } = useTranslation()
  const { run, error, dialog } = useAction()
  return (
    <Demo title={t('shared.devui.use_action')} description={t('shared.devui.use_action_hint')}>
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
    </Demo>
  )
}

export function Dialogs() {
  const { t } = useTranslation()
  const [modal, setModal] = useState<'form' | 'footer' | null>(null)
  const [ask, dialog] = useConfirm()
  return (
    <Demo title={t('shared.devui.dialogs')}>
      <Case label={t('shared.devui.case_form_modal')}>
        <Group>
          <Button variant="default" onClick={() => setModal('form')}>
            {t('shared.devui.open_modal')}
          </Button>
        </Group>
      </Case>
      <Case label={t('shared.devui.case_confirm')}>
        <Group>
          <Button
            variant="default"
            onClick={() => void ask({ title: t('shared.devui.confirm'), message: t('shared.devui.confirm_message'), confirmLabel: t('shared.devui.danger'), danger: true })}
          >
            {t('shared.devui.confirm')}
          </Button>
        </Group>
      </Case>
      <Case label={t('shared.devui.case_modal_footer')}>
        <Group>
          <Button variant="default" onClick={() => setModal('footer')}>
            {t('shared.devui.open_modal')}
          </Button>
        </Group>
      </Case>
      <FormModal opened={modal === 'form'} onClose={() => setModal(null)} title={t('shared.devui.open_modal')}>
        <EmptyState title={t('shared.devui.empty')} />
      </FormModal>
      <FormModal opened={modal === 'footer'} onClose={() => setModal(null)} title={t('shared.devui.export')}>
        <Stack gap="md">
          <Text>{`${rows[0]!.code} · ${rows[0]!.name}`}</Text>
          <ModalFooter>
            <Button onClick={() => setModal(null)}>{t('shared.devui.export')}</Button>
          </ModalFooter>
        </Stack>
      </FormModal>
      {dialog}
    </Demo>
  )
}

// Each case shows the toast at rest, then a button that raises the real one.
export function ToastsDemo() {
  const { t } = useTranslation()
  const undo = () => notifySuccess(t('shared.devui.toast_undone'))
  return (
    <Demo title={t('shared.devui.toasts')} description={t('shared.devui.toasts_hint')}>
      <Case label={t('shared.devui.case_toast_success')}>
        <Notification color="success" className={devui.toast}>
          {t('shared.devui.toast_saved')}
        </Notification>
        <Group>
          <Button variant="default" onClick={() => notifySuccess(t('shared.devui.toast_saved'))}>
            {t('shared.devui.show_toast')}
          </Button>
        </Group>
      </Case>
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
    </Demo>
  )
}
