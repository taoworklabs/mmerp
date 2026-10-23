import { api as coreApi, type Item } from '@/shared/api/core'
import { api, unwrap, type CustomerListItem } from '@/shared/api/sales'
import { RemoteSelectField } from '@/shared/ui/form'
import { salesKeys } from '../keys'

type Current = { id: number; code: string; name: string }

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
  current?: Current
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

// ItemField picks an active item of the catalogue by code or name.
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
  current?: Current
  readOnly?: boolean
  bare?: boolean
  onPicked?: (i: Item) => void
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
      search={async (q) => (await unwrap(coreApi.GET('/items', { params: { query: { q: q || undefined, active: 'true', page_size: 20 } } }))).items}
    />
  )
}
