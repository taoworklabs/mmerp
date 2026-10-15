import { IconClockHour4 } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import type { Overtime } from '@/shared/api/hrm'
import { formatDate, formatDecimal } from '@/shared/i18n'
import { FieldList } from '@/shared/ui/FieldList'
import { EditablePanel } from '@/shared/ui/form'
import { OvertimeForm } from './OvertimeForm'

// OvertimeDetails is an overtime request's values as label–value pairs; a draft the user may edit
// opens its form in the same card.
export function OvertimeDetails({ overtime: o }: { overtime: Overtime }) {
  const { t } = useTranslation()
  return (
    // Keyed by version: a save or a reload after a conflict starts from the saved data, closed.
    <EditablePanel key={o.version} title={t('hrm.overtime.section')} icon={IconClockHour4} editable={o.allowed_actions.includes('edit')} edit={(close) => <OvertimeForm overtime={o} onClose={close} />}>
      <FieldList
        rows={[
          [t('hrm.overtime.date'), formatDate(o.date)],
          [t('hrm.overtime.day_kind'), t(`hrm.overtime.day_kind.${o.day_kind}`)],
          [t('hrm.overtime.day_hours'), formatDecimal(o.day_hours)],
          [t('hrm.overtime.night_hours'), formatDecimal(o.night_hours)],
          [t('hrm.overtime.reason'), o.reason || '—'],
        ]}
      />
    </EditablePanel>
  )
}
