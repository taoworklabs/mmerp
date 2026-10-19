import { Stack, Text } from '@mantine/core'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Page } from '@/shared/ui/page'
import classes from './DevUi.module.css'

export type Row = { id: number; code: string; name: string; amount: number }

export const rows: Row[] = [
  { id: 1, code: 'NV001', name: 'Nguyễn Văn An', amount: 12500000 },
  { id: 2, code: 'NV002', name: 'Trần Thị Bình', amount: 9800000 },
  { id: 3, code: 'NV003', name: 'Lê Văn Cường', amount: 11200000 },
]

// Demo frames one catalogue entry the way a real screen would: inside a Page, centred, since an
// entry rarely fills the width.
export function Demo({ title, description, children }: { title: string; description?: string; children: ReactNode }) {
  const { t } = useTranslation()
  return (
    <Page centered title={title} description={description} breadcrumbs={[{ label: t('shared.devui.title'), to: '/dev/ui' }]}>
      <Stack gap="xl">{children}</Stack>
    </Page>
  )
}

// Case is one labelled example inside an entry, framed so it reads apart from the next one: what
// it is for in a strip on top, then the thing itself on its own canvas. Without a label, only the canvas.
export function Case({ label, children }: { label?: string; children: ReactNode }) {
  return (
    <section className={classes.case} aria-label={label}>
      {label && (
        <Text size="sm" fw={600} className={classes.label}>
          {label}
        </Text>
      )}
      <Stack gap="md" className={classes.canvas}>
        {children}
      </Stack>
    </section>
  )
}
