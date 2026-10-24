import { api, unwrap, type CustomerListItem, type LineItem } from '@/shared/api/sales'
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

// ItemField picks an active item for a document line by code or name; writing quotations or
// orders is enough, no item role needed.
export function ItemField({
  name,
  label,
  current,
  readOnly,
  bare,
  onPicked,
}: {
  name: string
  label: string
  current?: RemoteRow
  readOnly?: boolean
  bare?: boolean
  onPicked?: (i: LineItem) => void
}) {
  return (
    <RemoteSelectField
      name={name}
      label={label}
      current={current}
      readOnly={readOnly}
      required
      bare={bare}
      onPicked={onPicked}
      queryKey={salesKeys.items.picker}
      search={async (q) => (await unwrap(api.GET('/sales/line-items', { params: { query: { q: q || undefined } } }))).items}
    />
  )
}
