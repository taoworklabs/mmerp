import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { useTransfer } from '../hooks/useStock'
import { transferDocType } from '../keys'

// TransferPreview is a transfer in the approval inbox, when a rule holds one for approval. It is
// loaded outside the stock screens, so its strings live in the meta translations.
export default function TransferPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={transferDocType}
      query={useTransfer(id)}
      rows={(d) => [
        [t('stock.preview.from_warehouse'), `${d.from_warehouse_code} · ${d.from_warehouse_name}`],
        [t('stock.preview.to_warehouse'), `${d.to_warehouse_code} · ${d.to_warehouse_name}`],
        [t('stock.preview.date'), formatDate(d.date)],
        [t('stock.preview.lines'), String(d.lines.length)],
      ]}
    />
  )
}
