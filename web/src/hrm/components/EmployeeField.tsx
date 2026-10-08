import { useQuery } from '@tanstack/react-query'
import { useDebouncedValue } from '@mantine/hooks'
import { useState } from 'react'
import { api, unwrap } from '@/shared/api/hrm'
import { SelectField } from '@/shared/ui/form'
import { hrmKeys } from '../keys'

// One label for an employee everywhere in the picker; a label that changes makes Select
// rewrite its search text, which re-queries and flickers.
const employeeLabel = (code: string, fullName: string) => (code ? `${code} · ${fullName}` : fullName)

// EmployeeField picks an employee by code or name among those the user can see.
export function EmployeeField({
  name,
  label,
  current,
  readOnly,
  required,
  withTerminated,
  managerOf,
}: {
  name: string
  label: string
  current?: { id: number; code: string; name: string }
  readOnly?: boolean
  required?: boolean
  // Also offer employees who have left, e.g. to recover an overpayment.
  withTerminated?: boolean
  // Picking this employee's manager: leave out the employee and everyone under them.
  managerOf?: number
}) {
  const [search, setSearch] = useState('')
  const [q] = useDebouncedValue(search, 300)
  const found = useQuery({
    queryKey: [...hrmKeys.employees.picker(q), !!withTerminated, managerOf ?? 0],
    queryFn: () =>
      unwrap(api.GET('/hrm/employees', { params: { query: { q: q || undefined, page_size: 20, status: withTerminated ? undefined : 'active', manager_of: managerOf } } })),
    enabled: !readOnly,
  })
  const options = (found.data?.items ?? []).map((e) => ({ value: String(e.id), label: employeeLabel(e.code, e.full_name) }))
  // Keep the current one selectable even when the search does not return it.
  if (current && !options.some((o) => o.value === String(current.id))) options.unshift({ value: String(current.id), label: employeeLabel(current.code, current.name) })
  // Select puts the chosen option's label into the search box; that is not a search.
  const onSearch = (q: string) => setSearch(options.some((o) => o.label === q) ? '' : q)
  return <SelectField name={name} label={label} data={options} onSearch={onSearch} clearable={!required} required={required} readOnly={readOnly} />
}
