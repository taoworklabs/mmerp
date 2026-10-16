import { Button, Group } from '@mantine/core'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useConfirm } from '@/shared/ui/confirm'
import { notifyError, notifySuccess, notifyUndo } from '@/shared/ui/notify'
import { FormModal } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState, NotFoundPage, ReadOnlyBanner } from '@/shared/ui/states'
import { Demo } from './Demo'

export function States() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.states')}>
      <ContentSkeleton />
      <ReadOnlyBanner />
      <EmptyState title={t('shared.devui.empty')} description={t('shared.devui.empty_description')} />
      <EmptyState title={t('shared.devui.no_results')} action={<Button variant="default">{t('shared.devui.clear_filters')}</Button>} />
      <ErrorState message={t('shared.error.network_error')} onRetry={() => {}} />
      <NotFoundPage back={{ to: '/dev/ui', label: t('shared.devui.back') }} />
    </Demo>
  )
}

export function Dialogs() {
  const { t } = useTranslation()
  const [modal, setModal] = useState(false)
  const [ask, dialog] = useConfirm()
  return (
    <Demo title={t('shared.devui.dialogs')}>
      <Group>
        <Button variant="default" onClick={() => setModal(true)}>
          {t('shared.devui.open_modal')}
        </Button>
        <Button
          variant="default"
          onClick={() => void ask({ title: t('shared.devui.confirm'), message: t('shared.devui.confirm_message'), confirmLabel: t('shared.devui.danger'), danger: true })}
        >
          {t('shared.devui.confirm')}
        </Button>
        <Button variant="default" onClick={() => notifySuccess(t('shared.devui.undo_done'))}>
          {t('shared.devui.status_positive')}
        </Button>
        <Button variant="default" onClick={() => notifyError(t('shared.error.network_error'))}>
          {t('shared.devui.error')}
        </Button>
        <Button variant="default" onClick={() => notifyUndo(t('shared.devui.undo_done'), t('shared.undo'), () => undefined)}>
          {t('shared.devui.undo')}
        </Button>
      </Group>
      <FormModal opened={modal} onClose={() => setModal(false)} title={t('shared.devui.open_modal')}>
        <EmptyState title={t('shared.devui.empty')} />
      </FormModal>
      {dialog}
    </Demo>
  )
}
