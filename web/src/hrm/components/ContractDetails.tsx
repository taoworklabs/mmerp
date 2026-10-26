import { FieldList, Text } from '@mmerp/ui'
import { IconCash, IconFileText, IconListDetails } from '@tabler/icons-react'
import { Fragment } from 'react'
import { useTranslation } from 'react-i18next'
import type { Contract } from '@/shared/api/hrm'
import { formatDate, formatMoney } from '@/shared/i18n'
import { EditablePanel } from '@/shared/ui/form'
import { ContractForm, SavedLines, type ContractPart } from './ContractForm'

// ContractDetails is a contract's values in three cards (the contract, the salary, the allowances
// and supports), read as label–value pairs; a draft the user may edit opens one card's form in place.
// Without access to salaries the terms say so and cannot be edited.
export function ContractDetails({ contract: c }: { contract: Contract }) {
  const { t } = useTranslation()
  const appendix = c.parent_id !== null && c.parent_id !== undefined
  const canEdit = c.allowed_actions.includes('edit')
  const hidden = c.terms === null
  const edit = (part: ContractPart) => (close: () => void) => <ContractForm contract={c} part={part} onClose={close} />
  return (
    // Keyed by version: a save or a reload after a conflict starts from the saved data, closed.
    <Fragment key={c.version}>
      <EditablePanel title={t(appendix ? 'hrm.contract.section_appendix' : 'hrm.contract.section')} icon={IconFileText} editable={canEdit} edit={edit('main')}>
        <FieldList
          stacked
          rows={[
            [t('hrm.contract.kind'), c.contract_type_name],
            [t('hrm.contract.start_date'), formatDate(c.start_date)],
            ...(appendix ? [] : [[t('hrm.contract.end_date'), c.end_date ? formatDate(c.end_date) : t('hrm.contract.no_end')] as [string, string]]),
          ]}
        />
      </EditablePanel>
      <EditablePanel title={t('hrm.contract.section_terms')} icon={IconCash} editable={canEdit && !hidden} edit={edit('terms')}>
        {hidden ? (
          <Text size="sm" tone="dimmed">
            {t('hrm.contract.terms_hidden')}
          </Text>
        ) : (
          <FieldList stacked rows={[[t('hrm.contract.salary'), formatMoney(c.terms!.salary)]]} />
        )}
      </EditablePanel>
      {!hidden && (
        <EditablePanel title={t('hrm.contract.section_lines')} icon={IconListDetails} editable={canEdit} edit={edit('lines')}>
          <SavedLines lines={c.terms!.lines} />
        </EditablePanel>
      )}
    </Fragment>
  )
}
