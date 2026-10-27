import { Anchor } from '@mmerp/ui'
import type { ReactNode } from 'react'
import { Link } from 'react-router'

// EmployeeLink is a person's name or code wherever it shows, leading to their profile.
export function EmployeeLink({ id, children }: { id: number; children: ReactNode }) {
  return (
    <Anchor render={<Link to={`/hrm/employees/${id}`} />}>
      {children}
    </Anchor>
  )
}
