import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '@/shared/api/sales'
import { salesKeys, type Kind } from '../keys'

// useCustomerActions and useDocActions: what the user may do before picking a record (create),
// as the API decides.
export function useCustomerActions() {
  return useQuery({
    queryKey: salesKeys.customers.actions(),
    queryFn: async () => (await unwrap(api.GET('/sales/customers/actions'))).allowed_actions,
  })
}

export function useDocActions(kind: Kind) {
  return useQuery({
    queryKey: salesKeys.docs.actions(kind),
    queryFn: async () => (await unwrap(kind === 'quote' ? api.GET('/sales/quotes/actions') : api.GET('/sales/orders/actions'))).allowed_actions,
  })
}

export function useCustomer(id: number) {
  return useQuery({
    queryKey: salesKeys.customers.detail(id),
    queryFn: () => unwrap(api.GET('/sales/customers/{id}', { params: { path: { id } } })),
  })
}

export function useDoc(kind: Kind, id: number) {
  return useQuery({
    queryKey: salesKeys.docs.detail(kind, id),
    queryFn: () =>
      unwrap(kind === 'quote' ? api.GET('/sales/quotes/{id}', { params: { path: { id } } }) : api.GET('/sales/orders/{id}', { params: { path: { id } } })),
  })
}
