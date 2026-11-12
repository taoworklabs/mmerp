import { api, unwrap } from '@/shared/api/accounting'
import { RemoteSelectField, type RemoteRow } from '@/shared/ui/form'
import { accountingKeys } from '../keys'

// AccountingPartnerPicker picks a partner on a journal entry line; accounting serves the
// partners to whoever writes entries, so no partner role is needed.
export function AccountingPartnerPicker({
  name,
  label,
  current,
  required,
  bare,
}: {
  name: string
  label: string
  current?: RemoteRow
  required?: boolean
  bare?: boolean
}) {
  return (
    <RemoteSelectField
      name={name}
      label={label}
      current={current}
      required={required}
      bare={bare}
      queryKey={accountingKeys.partners}
      search={async (q) => (await unwrap(api.GET('/accounting/partners', { params: { query: { q: q || undefined } } }))).items}
    />
  )
}
