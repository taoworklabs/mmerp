import { useDebouncedValue } from '@mantine/hooks'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { api, unwrap, type CustomerListItem, type Item } from '@/shared/api/sales'
import { SelectField } from '@/shared/ui/form'
import { salesKeys } from '../keys'

// One label per record everywhere in a picker; a label that changes makes Select rewrite
// its search text, which re-queries and flickers.
const label = (code: string, name: string) => `${code} · ${name}`

type Current = { id: number; code: string; name: string }

// CustomerField picks an active customer the user can see, by code, name or tax code.
export function CustomerField({
  name,
  label: text,
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
  const [search, setSearch] = useState('')
  const [q] = useDebouncedValue(search, 300)
  const found = useQuery({
    queryKey: salesKeys.customers.picker(q),
    queryFn: () => unwrap(api.GET('/sales/customers', { params: { query: { q: q || undefined, active: 'true', page_size: 20 } } })),
    enabled: !readOnly,
  })
  const rows = found.data?.items ?? []
  return (
    <Picker
      name={name}
      label={text}
      current={current}
      readOnly={readOnly}
      rows={rows}
      onSearch={setSearch}
      onPicked={(id) => {
        const c = rows.find((r) => r.id === id)
        if (c) onPicked?.(c)
      }}
    />
  )
}

// ItemField picks an active item of the catalogue by code or name.
export function ItemField({
  name,
  label: text,
  current,
  readOnly,
  onPicked,
}: {
  name: string
  label: string
  current?: Current
  readOnly?: boolean
  onPicked?: (i: Item) => void
}) {
  const [search, setSearch] = useState('')
  const [q] = useDebouncedValue(search, 300)
  const found = useQuery({
    queryKey: salesKeys.items.picker(q),
    queryFn: () => unwrap(api.GET('/sales/items', { params: { query: { q: q || undefined, active: 'true', page_size: 20 } } })),
    enabled: !readOnly,
  })
  const rows = found.data?.items ?? []
  return (
    <Picker
      name={name}
      label={text}
      current={current}
      readOnly={readOnly}
      rows={rows}
      onSearch={setSearch}
      onPicked={(id) => {
        const i = rows.find((r) => r.id === id)
        if (i) onPicked?.(i)
      }}
    />
  )
}

function Picker({
  name,
  label: text,
  current,
  readOnly,
  rows,
  onSearch,
  onPicked,
}: {
  name: string
  label: string
  current?: Current
  readOnly?: boolean
  rows: { id: number; code: string; name: string }[]
  onSearch: (q: string) => void
  onPicked: (id: number) => void
}) {
  const options = rows.map((r) => ({ value: String(r.id), label: label(r.code, r.name) }))
  // Keep the current one selectable even when the search does not return it.
  if (current && !options.some((o) => o.value === String(current.id))) options.unshift({ value: String(current.id), label: label(current.code, current.name) })
  return (
    <SelectField
      name={name}
      label={text}
      data={options}
      required
      readOnly={readOnly}
      // Select puts the chosen option's label into the search box; that is not a search.
      onSearch={(q) => onSearch(options.some((o) => o.label === q) ? '' : q)}
      onChange={(v) => v && onPicked(Number(v))}
    />
  )
}
