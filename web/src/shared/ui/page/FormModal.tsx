import { Dialog } from '@mmerp/ui'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { UnsavedProvider, useUnsaved } from '../form'

type Props = { title: string; opened: boolean; onClose: () => void; wide?: boolean; children: ReactNode }

// FormModal holds a short form (six fields at most), one column at 440px; wide (620px) is for
// long text or a table. It is fullscreen below 1024px.
export function FormModal(props: Props) {
  return (
    <UnsavedProvider>
      <FormDialog {...props} />
    </UnsavedProvider>
  )
}

// A stray click outside must not drop unsaved edits; Escape and Cancel still close.
function FormDialog({ title, opened, onClose, wide, children }: Props) {
  const { t } = useTranslation()
  const unsaved = useUnsaved()
  return (
    <Dialog opened={opened} onClose={onClose} title={title} size={wide ? 'lg' : 'md'} closeLabel={t('shared.modal.close')} closeOnClickOutside={!unsaved}>
      {children}
    </Dialog>
  )
}
