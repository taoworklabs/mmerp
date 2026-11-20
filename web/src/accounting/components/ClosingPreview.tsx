import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { useClosing } from '../hooks/useAccounting'
import { closingDocType } from '../keys'

// ClosingPreview is a closing entry in the approval inbox; its strings live in the meta translations.
export default function ClosingPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={closingDocType}
      query={useClosing(id)}
      rows={(d) => [
        [t('accounting.preview.date'), formatDate(d.date)],
        [t('accounting.preview.description'), d.description],
        [t('accounting.preview.lines'), String(d.lines.length)],
      ]}
    />
  )
}
