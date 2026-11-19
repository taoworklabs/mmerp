import { api, unwrap, type SupplierListItem, type LineItem } from '@/shared/api/purchase'
import { RemoteSelectField, type RemoteRow } from '@/shared/ui/form'
import { purchaseKeys } from '../keys'

// SupplierField picks an active supplier the user can see, by code, name or tax code.
export function SupplierField({
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
  onPicked?: (c: SupplierListItem) => void
}) {
  return (
    <RemoteSelectField
      name={name}
      label={label}
      current={current}
      readOnly={readOnly}
      required
      onPicked={onPicked}
      queryKey={purchaseKeys.suppliers.picker}
      search={async (q) => (await unwrap(api.GET('/purchase/suppliers', { params: { query: { q: q || undefined, active: 'true', page_size: 20 } } }))).items}
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
      queryKey={purchaseKeys.items.picker}
      search={async (q) => (await unwrap(api.GET('/purchase/line-items', { params: { query: { q: q || undefined } } }))).items}
    />
  )
}
