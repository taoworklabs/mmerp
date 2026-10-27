import { Button, Dialog, Stack, Text } from '@mmerp/ui'
import { useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiError } from '@/shared/api/error'
import { useInvalidateRecord } from '@/shared/area'
import { ModalFooter } from '@/shared/ui/ModalFooter'

// useDocumentMutation runs a write on a document at the version the user sees, then
// refreshes everything the owning area declared. A version conflict opens the
// "data changed" dialog; render `dialog`. Other errors are rethrown for the caller to show.
export function useDocumentMutation(docType: string, id: number, version: number) {
  const { t } = useTranslation()
  const invalidate = useInvalidateRecord()
  const [conflict, setConflict] = useState(false)
  const reloadRef = useRef<HTMLButtonElement>(null)

  async function run<T>(call: (version: number) => Promise<T>): Promise<T> {
    try {
      const out = await call(version)
      await invalidate(docType, id)
      return out
    } catch (err) {
      if (err instanceof ApiError && err.code === 'version_conflict') setConflict(true)
      throw err
    }
  }

  const reload = async () => {
    setConflict(false)
    await invalidate(docType, id)
  }

  const dialog: ReactNode = conflict && (
    <Dialog opened onClose={() => void reload()} title={t('shared.document.conflict.title')} closeLabel={t('shared.modal.close')} initialFocus={reloadRef}>
      <Stack gap="md">
        <Text>{t('shared.document.conflict.message')}</Text>
        <ModalFooter>
          <Button onClick={() => void reload()} ref={reloadRef}>
            {t('shared.document.conflict.reload')}
          </Button>
        </ModalFooter>
      </Stack>
    </Dialog>
  )
  return { run, dialog }
}
