import { Anchor } from '@mantine/core'
import type { ReactNode } from 'react'
import { Link } from 'react-router'

// EmployeeLink is a person's name or code wherever it shows, leading to their profile.
export function EmployeeLink({ id, children }: { id: number; children: ReactNode }) {
  return (
    <Anchor component={Link} to={`/hrm/employees/${id}`} size="sm" inherit>
      {children}
    </Anchor>
  )
}
