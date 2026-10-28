import { Anchor } from '@mmerp/ui'
import type { ReactNode } from 'react'
import { Link, useMatch } from 'react-router'
import { useLeaveSelf } from '../hooks/useLeave'

// EmployeeLink is a person's name or code wherever it shows, leading to their profile. In the
// personal space only the user's own name leads anywhere, to their own profile: people there
// may not open /hrm/employees, and a report's request shows the report's name.
export function EmployeeLink({ id, children }: { id: number; children: ReactNode }) {
  const personal = useMatch('/me/*') != null
  const self = useLeaveSelf().data
  if (personal && self?.id !== id) return <>{children}</>
  return (
    <Anchor render={<Link to={personal ? '/me/profile' : `/hrm/employees/${id}`} />}>
      {children}
    </Anchor>
  )
}
