import { Stack } from '@mantine/core'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Page } from '@/shared/ui/page'

export type Row = { id: number; code: string; name: string; amount: number }

export const rows: Row[] = [
  { id: 1, code: 'NV001', name: 'Nguyễn Văn An', amount: 12500000 },
  { id: 2, code: 'NV002', name: 'Trần Thị Bình', amount: 9800000 },
  { id: 3, code: 'NV003', name: 'Lê Văn Cường', amount: 11200000 },
]

// Demo frames one catalogue entry the way a real screen would: inside a Page.
export function Demo({ title, description, children }: { title: string; description?: string; children: ReactNode }) {
  const { t } = useTranslation()
  return (
    <Page title={title} description={description} breadcrumbs={[{ label: t('shared.devui.title'), to: '/dev/ui' }]}>
      <Stack gap="lg">{children}</Stack>
    </Page>
  )
}
