import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { DocumentScreen } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { useLegalEntities } from '@/shared/ui/form'
import { loaded } from '@/shared/ui/loaded'
import { ClosingForm } from '../components/ClosingForm'
import { useClosing } from '../hooks/useAccounting'
import { closingDocType } from '../keys'

const list = '/accounting/closings'

// ClosingPage shows a closing entry: what it would move while a draft, what it moved once posted.
export function ClosingPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const entities = useLegalEntities({}).options
  const state = loaded(useClosing(id), t, { to: list, label: t('accounting.closings.back') })
  if (!state.ok) return state.fallback
  const d = state.data
  return (
    <DocumentScreen
      docType={closingDocType}
      doc={d}
      list={{ label: t('accounting.closings.title'), to: list }}
      facts={[entities.find((o) => o.value === String(d.legal_entity_id))?.label, formatDate(d.date), d.description]}
      attachments
      fullWidth
    >
      <ClosingForm key={d.version} doc={d} />
    </DocumentScreen>
  )
}
