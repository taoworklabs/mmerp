import { api, unwrap, type CustomerListItem } from '@/shared/api/sales'
import { RemoteSelectField, type RemoteRow } from '@/shared/ui/form'
import { salesKeys } from '../keys'

// CustomerField picks an active customer the user can see, by code, name or tax code.
export function CustomerField({
  name,
  label,
  current,
  readOnly,
  onPicked,
}: {
  name: string
  label: string
  current?: RemoteRow
  readOnly?: boolean
  onPicked?: (c: CustomerListItem) => void
}) {
  return (
    <RemoteSelectField
      name={name}
      label={label}
      current={current}
      readOnly={readOnly}
      required
      onPicked={onPicked}
      queryKey={salesKeys.customers.picker}
      search={async (q) => (await unwrap(api.GET('/sales/customers', { params: { query: { q: q || undefined, active: 'true', page_size: 20 } } }))).items}
    />
  )
}
