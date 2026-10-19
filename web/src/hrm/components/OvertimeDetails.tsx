import { IconClockHour4 } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import type { Overtime } from '@/shared/api/hrm'
import { formatDate, formatDecimal } from '@/shared/i18n'
import { FieldList, MissingValue } from '@/shared/ui/FieldList'
import { EditablePanel } from '@/shared/ui/form'
import { OvertimeForm } from './OvertimeForm'

// OvertimeDetails is an overtime request's values as label–value pairs; a draft the user may edit
// opens its form in the same card.
export function OvertimeDetails({ overtime: o }: { overtime: Overtime }) {
  const { t } = useTranslation()
  const canEdit = o.allowed_actions.includes('edit')
  return (
    // Keyed by version: a save or a reload after a conflict starts from the saved data, closed.
    <EditablePanel key={o.version} title={t('hrm.overtime.section')} icon={IconClockHour4} editable={canEdit} edit={(close) => <OvertimeForm overtime={o} onClose={close} />}>
      {(open) => (
        <FieldList
          stacked
          rows={[
            [t('hrm.overtime.date'), formatDate(o.date)],
            [t('hrm.overtime.day_kind'), t(`hrm.overtime.day_kind.${o.day_kind}`)],
            [t('hrm.overtime.day_hours'), formatDecimal(o.day_hours)],
            [t('hrm.overtime.night_hours'), formatDecimal(o.night_hours)],
            // A draft offers to fill an empty value in; a decided request just has none.
            [t('hrm.overtime.reason'), o.reason || (canEdit ? <MissingValue label={t('hrm.overtime.reason')} onAdd={open} /> : '—'), 'wide'],
          ]}
        />
      )}
    </EditablePanel>
  )
}
