import { Dialog } from '@mmerp/ui'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

// FormModal holds a short form (six fields at most), one column at 440px; wide (620px) is for
// long text or a table. It is fullscreen below 1024px.
export function FormModal({ title, opened, onClose, wide, children }: { title: string; opened: boolean; onClose: () => void; wide?: boolean; children: ReactNode }) {
  const { t } = useTranslation()
  return (
    <Dialog opened={opened} onClose={onClose} title={title} size={wide ? 'lg' : 'md'} closeLabel={t('shared.modal.close')}>
      {children}
    </Dialog>
  )
}
