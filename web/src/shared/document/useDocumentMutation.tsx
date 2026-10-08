import { Button, Group, Modal, Stack, Text } from '@mantine/core'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiError } from '@/shared/api/error'
import { useInvalidateRecord } from '@/shared/area'

// useDocumentMutation runs a write on a document at the version the user sees, then
// refreshes everything the owning area declared. A version conflict opens the
// "data changed" dialog; render `dialog`. Other errors are rethrown for the caller to show.
export function useDocumentMutation(docType: string, id: number, version: number) {
  const { t } = useTranslation()
  const invalidate = useInvalidateRecord()
  const [conflict, setConflict] = useState(false)

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
    <Modal opened onClose={() => void reload()} title={t('shared.document.conflict.title')}>
      <Stack gap="md">
        <Text>{t('shared.document.conflict.message')}</Text>
        <Group justify="flex-end">
          <Button onClick={() => void reload()} data-autofocus>
            {t('shared.document.conflict.reload')}
          </Button>
        </Group>
      </Stack>
    </Modal>
  )
  return { run, dialog }
}
