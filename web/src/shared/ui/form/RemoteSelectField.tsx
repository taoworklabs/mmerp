import { useDebouncedValue } from '@mantine/hooks'
import { useQuery, type QueryKey } from '@tanstack/react-query'
import { useState } from 'react'
import { SelectField, type Option } from './fields'

export type RemoteRow = { id: number; code: string; name: string }

// One label per record everywhere in the select; a label that changes makes Select rewrite
// its search text, which re-queries and flickers.
export const rowLabel = (r: RemoteRow) => (r.code ? `${r.code} · ${r.name}` : r.name)

// remoteOptions lists the found rows, with the kept ones (the chosen value) first when the
// search did not return them, so the field never loses the label of its value.
export function remoteOptions(rows: RemoteRow[], keep: (RemoteRow | undefined)[]): Option[] {
  const seen = new Set(rows.map((r) => r.id))
  const kept: RemoteRow[] = []
  for (const r of keep) {
    if (!r || seen.has(r.id)) continue
    seen.add(r.id)
    kept.push(r)
  }
  return [...kept, ...rows].map((r) => ({ value: String(r.id), label: rowLabel(r) }))
}

// RemoteSelectField picks a record searched on the server: a page of 20 per debounced search,
// labelled "code · name". `current` is the saved value, shown before any search.
export function RemoteSelectField<T extends RemoteRow>({
  name,
  label,
  queryKey,
  search,
  current,
  readOnly,
  required,
  bare,
  onPicked,
}: {
  name: string
  label: string
  queryKey: (q: string) => QueryKey
  // Returns at most 20 rows matching q (all rows when q is empty).
  search: (q: string) => Promise<T[]>
  current?: RemoteRow
  readOnly?: boolean
  required?: boolean
  bare?: boolean
  onPicked?: (row: T) => void
}) {
  const [text, setText] = useState('')
  const [q] = useDebouncedValue(text, 300)
  const [picked, setPicked] = useState<T>()
  const found = useQuery({ queryKey: queryKey(q), queryFn: () => search(q), enabled: !readOnly })
  const rows = found.data ?? []
  const options = remoteOptions(rows, [picked, current])
  return (
    <SelectField
      name={name}
      label={label}
      data={options}
      clearable={!required}
      required={required}
      readOnly={readOnly}
      bare={bare}
      // Select puts the chosen option's label into the search box; that is not a search.
      onSearch={(s) => setText(options.some((o) => o.label === s) ? '' : s)}
      onChange={(v) => {
        const row = rows.find((r) => String(r.id) === v)
        if (!row) return
        setPicked(row)
        onPicked?.(row)
      }}
    />
  )
}
