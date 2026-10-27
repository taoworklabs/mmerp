import { useMatch } from 'react-router'

// useRequestPaths: where a request's list, new form and detail page are. Under the personal
// space they stay there; elsewhere they are the HRM pages.
export function useRequestPaths(kind: 'leave' | 'overtime') {
  const mine = useMatch('/me/*') != null
  const base = mine ? `/me/requests/${kind}` : `/hrm/${kind}s`
  return { list: mine ? '/me/requests' : `/hrm/${kind}s`, open: (id: number) => `${base}/${id}` }
}
