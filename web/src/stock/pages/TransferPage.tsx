import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { DocumentScreen } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { loaded } from '@/shared/ui/loaded'
import { TransferForm } from '../components/TransferForm'
import { useTransfer } from '../hooks/useStock'
import { transferDocType } from '../keys'

const list = '/stock/transfers'

// TransferPage shows a transfer; a draft the user may edit is edited in place.
export function TransferPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const state = loaded(useTransfer(id), t, { to: list, label: t('stock.transfers.back') })
  if (!state.ok) return state.fallback
  const d = state.data
  return (
    <DocumentScreen
      docType={transferDocType}
      doc={d}
      list={{ label: t('stock.transfers.title'), to: list }}
      facts={[`${d.from_warehouse_code} → ${d.to_warehouse_code}`, formatDate(d.date)]}
      attachments
      // The editable lines table needs the whole width; approval and history become tabs.
      fullWidth
      printLabel={t('stock.transfer.print')}
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <TransferForm key={d.version} doc={d} />
    </DocumentScreen>
  )
}
