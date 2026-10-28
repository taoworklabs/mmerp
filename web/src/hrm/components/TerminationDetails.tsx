import { FieldList, Text } from '@mmerp/ui'
import { IconCash, IconClipboardList, IconUserOff } from '@tabler/icons-react'
import { Fragment } from 'react'
import { useTranslation } from 'react-i18next'
import type { Termination } from '@/shared/api/hrm'
import { formatDate, formatMoney, formatNumber } from '@/shared/i18n'
import { MissingValue } from '@/shared/ui/MissingValue'
import { Notice } from '@/shared/ui/Notice'
import { EditablePanel } from '@/shared/ui/form'
import { TerminationForm, type TerminationPart } from './TerminationForm'

// TerminationDetails is a termination's values in three cards, with what still stands past
// the date above them; a draft the user may edit opens one card's form in place. The
// allowance's amount shows only with access to salaries.
export function TerminationDetails({ termination: x }: { termination: Termination }) {
  const { t } = useTranslation()
  const canEdit = x.allowed_actions.includes('edit')
  const edit = (part: TerminationPart) => (close: () => void) => <TerminationForm termination={x} part={part} onClose={close} />
  const a = x.allowance
  return (
    // Keyed by version: a save or a reload after a conflict starts from the saved data, closed.
    <Fragment key={x.version}>
      {x.warnings.length > 0 && (
        <Notice tone="warning" title={t('hrm.termination.warnings')}>
          <ul>
            {x.warnings.map((w, i) => (
              <li key={i}>{t(`hrm.termination.warning.${w.code}`, w.params as Record<string, unknown>)}</li>
            ))}
          </ul>
        </Notice>
      )}
      <EditablePanel title={t('hrm.termination.section')} icon={IconUserOff} editable={canEdit} edit={edit('main')}>
        <FieldList
          stacked
          rows={[
            [t('hrm.termination.date'), formatDate(x.termination_date)],
            [t('hrm.termination.reason'), t(`hrm.termination.reason.${x.reason}`)],
            [t('hrm.termination.decision_date'), x.decision_date ? formatDate(x.decision_date) : <MissingValue label={t('hrm.termination.decision_date')} />],
          ]}
        />
      </EditablePanel>
      <EditablePanel title={t('hrm.termination.section_allowance')} icon={IconCash} editable={canEdit} edit={edit('allowance')}>
        <FieldList
          stacked
          rows={[
            [t('hrm.termination.allowance_kind'), t(`hrm.termination.allowance.${a.kind}`)],
            [t('hrm.termination.months_worked'), formatNumber(a.months_worked)],
            [t('hrm.termination.insured_months'), formatNumber(x.insured_months)],
            [t('hrm.termination.paid_months'), formatNumber(x.paid_months)],
            [t('hrm.termination.pension_eligible'), t(x.pension_eligible ? 'hrm.common.yes' : 'hrm.common.no')],
            [t('hrm.termination.years'), t('hrm.termination.years_value', { years: a.years, months: a.months_counted })],
            ...(a.amount != null && a.average != null
              ? ([
                  [t('hrm.termination.average'), formatMoney(a.average)],
                  [t('hrm.termination.amount'), formatMoney(a.amount)],
                ] as [string, string][])
              : []),
          ]}
        />
        {a.amount == null && (
          <Text size="sm" tone="dimmed">
            {t('hrm.termination.amount_hidden')}
          </Text>
        )}
      </EditablePanel>
      <EditablePanel title={t('hrm.termination.section_handover')} icon={IconClipboardList} editable={canEdit} edit={edit('handover')}>
        <FieldList
          stacked
          rows={[
            [t('hrm.termination.handover'), x.handover || <MissingValue label={t('hrm.termination.handover')} />, 'wide'],
            [t('hrm.termination.note'), x.note || <MissingValue label={t('hrm.termination.note')} />, 'wide'],
          ]}
        />
      </EditablePanel>
    </Fragment>
  )
}
