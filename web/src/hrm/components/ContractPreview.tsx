import { Stack } from '@mantine/core'
import { useTranslation } from 'react-i18next'
import { DocumentStatus } from '@/shared/document'
import { errorText, formatDate, formatMoney } from '@/shared/i18n'
import { FieldList } from '@/shared/ui/FieldList'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { useContract } from '../hooks/useContract'
import { contractDocType } from '../keys'

// ContractPreview is the read-only view of a contract in the approval inbox. It is
// loaded outside the hrm screens, so its strings live in the meta translations.
export default function ContractPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  const contract = useContract(id)
  if (contract.isPending) return <ContentSkeleton />
  if (contract.isError) return <ErrorState message={errorText(t, contract.error)} onRetry={() => void contract.refetch()} />
  const c = contract.data
  const rows: [string, string][] = [
    [t('hrm.contract.employee'), `${c.employee_code} · ${c.employee_name}`],
    [t('hrm.contract.kind'), c.parent_number ? `${c.contract_type_name} · ${t('hrm.contract.appendix_of', { number: c.parent_number })}` : c.contract_type_name],
    [t('hrm.contract.start_date'), formatDate(c.start_date)],
    ...(c.parent_number ? [] : [[t('hrm.contract.end_date'), c.end_date ? formatDate(c.end_date) : t('hrm.contract.no_end')] as [string, string]]),
    // Terms come only with the salary permission.
    ...(c.terms ? [[t('hrm.contract.salary'), formatMoney(c.terms.salary)] as [string, string]] : []),
    ...(c.terms?.lines.map((l) => [l.name, formatMoney(l.amount)] as [string, string]) ?? []),
  ]
  return (
    <Stack gap="sm">
      <DocumentStatus docType={contractDocType} status={c.status} />
      <FieldList rows={rows} />
    </Stack>
  )
}
