import { Stack } from '@mantine/core'
import { useTranslation } from 'react-i18next'
import { DocumentStatus } from '@/shared/document'
import { errorText, formatDate, formatNumber } from '@/shared/i18n'
import { FieldList } from '@/shared/ui/FieldList'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { useDoc } from '../hooks/useSales'
import { docTypeOf, type Kind } from '../keys'

// DocPreview is the read-only view of a quotation or order in the approval inbox. It is
// loaded outside the sales screens, so its strings live in the meta translations.
export function DocPreview({ kind, id }: { kind: Kind; id: number }) {
  const { t } = useTranslation()
  const doc = useDoc(kind, id)
  if (doc.isPending) return <ContentSkeleton />
  if (doc.isError) return <ErrorState message={errorText(t, doc.error)} onRetry={() => void doc.refetch()} />
  const d = doc.data
  const maxDiscount = d.lines.reduce((m, l) => Math.max(m, Number(l.discount_percent)), 0)
  const rows: [string, string][] = [
    [t('sales.preview.customer'), `${d.customer.code} · ${d.customer.name}`],
    [t('sales.preview.date'), formatDate(d.date)],
    ...(d.valid_until ? [[t('sales.preview.valid_until'), formatDate(d.valid_until)] as [string, string]] : []),
    [t('sales.preview.org_unit'), d.org_unit_name],
    [t('sales.preview.lines'), String(d.lines.length)],
    [t('sales.doc.max_discount'), `${maxDiscount}%`],
    [t('sales.preview.discount_total'), formatNumber(d.discount_total)],
    [t('sales.preview.total'), formatNumber(d.total)],
  ]
  return (
    <Stack gap="sm">
      <DocumentStatus docType={docTypeOf(kind)} status={d.status} />
      <FieldList rows={rows} />
    </Stack>
  )
}
