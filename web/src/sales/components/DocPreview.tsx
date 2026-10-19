import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate, formatDecimal, formatNumber } from '@/shared/i18n'
import { useDoc } from '../hooks/useSales'
import { docTypeOf, type Kind } from '../keys'

// DocPreview is the read-only view of a quotation or order in the approval inbox. It is
// loaded outside the sales screens, so its strings live in the meta translations.
export function DocPreview({ kind, id }: { kind: Kind; id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={docTypeOf(kind)}
      query={useDoc(kind, id)}
      rows={(d) => [
        [t('sales.preview.customer'), `${d.customer.code} · ${d.customer.name}`],
        [t('sales.preview.date'), formatDate(d.date)],
        d.valid_until && [t('sales.preview.valid_until'), formatDate(d.valid_until)],
        [t('sales.preview.org_unit'), d.org_unit_name],
        [t('sales.preview.lines'), String(d.lines.length)],
        [t('sales.doc.max_discount'), formatDecimal(d.max_discount)],
        [t('sales.preview.discount_total'), formatNumber(d.discount_total)],
        [t('sales.preview.total'), formatNumber(d.total)],
      ]}
    />
  )
}
