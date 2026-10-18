import { api, unwrap } from '@/shared/api/hrm'
import { RemoteSelectField } from '@/shared/ui/form'
import { hrmKeys } from '../keys'

// EmployeeField picks an employee by code or name among those the user can see.
export function EmployeeField({
  name,
  label,
  current,
  readOnly,
  required,
  withTerminated,
  managerOf,
  bare,
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
  bare?: boolean
}) {
  return (
    <RemoteSelectField
      name={name}
      label={label}
      current={current}
      readOnly={readOnly}
      required={required}
      bare={bare}
      queryKey={(q) => [...hrmKeys.employees.picker(q), !!withTerminated, managerOf ?? 0]}
      search={async (q) => {
        const page = await unwrap(
          api.GET('/hrm/employees', { params: { query: { q: q || undefined, page_size: 20, status: withTerminated ? undefined : 'active', manager_of: managerOf } } }),
        )
        return page.items.map((e) => ({ id: e.id, code: e.code, name: e.full_name }))
      }}
    />
  )
}
