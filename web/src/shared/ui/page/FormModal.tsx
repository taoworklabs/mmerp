import { Modal } from '@mantine/core'
import { useMediaQuery } from '@mantine/hooks'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

// FormModal holds a short form (six fields at most); it is fullscreen below 1024px.
export function FormModal({ title, opened, onClose, children }: { title: string; opened: boolean; onClose: () => void; children: ReactNode }) {
  const { t } = useTranslation()
  const wide = useMediaQuery('(min-width: 64em)', true)
  return (
    <Modal opened={opened} onClose={onClose} title={title} fullScreen={!wide} size="lg" closeButtonProps={{ 'aria-label': t('shared.modal.close') }}>
      {children}
    </Modal>
  )
}
