import { api, unwrap, type StockItem } from '@/shared/api/core'
import { RemoteSelectField, type RemoteRow } from '@/shared/ui/form'
import { stockKeys } from '../keys'

// StockItemField picks an active item with stock for a stock document line, by code or name;
// viewing stock is enough, no item role needed.
export function StockItemField({
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
  onPicked?: (i: StockItem) => void
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
      queryKey={stockKeys.pickerItems}
      search={async (q) => (await unwrap(api.GET('/stock/items', { params: { query: { q: q || undefined } } }))).items}
    />
  )
}
