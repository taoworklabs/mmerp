import { api, unwrap, type Item } from '@/shared/api/core'
import { RemoteSelectField, type RemoteRow } from '@/shared/ui/form'
import { masterKeys } from './keys'

// ItemPicker picks an active item of the catalogue by code or name.
export function ItemPicker({
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
      queryKey={masterKeys.items.picker}
      search={async (q) => (await unwrap(api.GET('/items', { params: { query: { q: q || undefined, active: 'true', page_size: 20 } } }))).items}
    />
  )
}
