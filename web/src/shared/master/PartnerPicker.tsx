import { api, unwrap, type PartnerListItem } from '@/shared/api/core'
import { RemoteSelectField, type RemoteRow } from '@/shared/ui/form'
import { masterKeys } from './keys'

// PartnerPicker picks a partner of the tenant by code, name or tax code.
export function PartnerPicker({
  name,
  label,
  current,
  onPicked,
}: {
  name: string
  label: string
  current?: RemoteRow
  onPicked?: (p: PartnerListItem) => void
}) {
  return (
    <RemoteSelectField
      name={name}
      label={label}
      current={current}
      onPicked={onPicked}
      queryKey={masterKeys.partners.picker}
      search={async (q) => (await unwrap(api.GET('/partners', { params: { query: { q: q || undefined, page_size: 20 } } }))).items}
    />
  )
}
