import { Anchor } from '@mmerp/ui'
import type { ReactNode } from 'react'
import { Link, useMatch } from 'react-router'

// EmployeeLink is a person's name or code wherever it shows, leading to their profile. In the
// personal space it leads to the own profile, as plain staff may not open /hrm/employees.
export function EmployeeLink({ id, children }: { id: number; children: ReactNode }) {
  const mine = useMatch('/me/*') != null
  return (
    <Anchor render={<Link to={mine ? '/me/profile' : `/hrm/employees/${id}`} />}>
      {children}
    </Anchor>
  )
}
