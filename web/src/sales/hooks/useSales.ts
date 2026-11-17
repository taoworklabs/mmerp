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

export function useDoc(kind: Kind, id: number, enabled = true) {
  return useQuery({
    enabled,
    queryKey: salesKeys.docs.detail(kind, id),
    queryFn: () =>
      unwrap(kind === 'quote' ? api.GET('/sales/quotes/{id}', { params: { path: { id } } }) : api.GET('/sales/orders/{id}', { params: { path: { id } } })),
  })
}

// useCustomerDocs: a customer's latest quotations or orders, newest first, and how many there are.
export function useCustomerDocs(kind: Kind, customer: number) {
  return useQuery({
    queryKey: salesKeys.docs.customer(kind, customer),
    queryFn: () => {
      // ponytail: the latest 100 only; page the tab if a customer ever has more
      const p = { params: { query: { customer_id: customer, sort: '-date' as const, page_size: 100 as const } } }
      return unwrap(kind === 'quote' ? api.GET('/sales/quotes', p) : api.GET('/sales/orders', p))
    },
  })
}

export function useDelivery(id: number) {
  return useQuery({
    queryKey: salesKeys.deliveries.detail(id),
    queryFn: () => unwrap(api.GET('/sales/deliveries/{id}', { params: { path: { id } } })),
  })
}

// deliveryOptions is the query of what a delivery from the order may take from the warehouse
// (0: the legal entity's default).
export const deliveryOptions = (order: number, warehouse: number) => ({
  queryKey: salesKeys.deliveries.options(order, warehouse),
  queryFn: () =>
    unwrap(api.GET('/sales/orders/{id}/delivery-options', { params: { path: { id: order }, query: { warehouse_id: warehouse || undefined } } })),
})

export function useInvoice(id: number, enabled = true) {
  return useQuery({
    enabled,
    queryKey: salesKeys.invoices.detail(id),
    queryFn: () => unwrap(api.GET('/sales/invoices/{id}', { params: { path: { id } } })),
  })
}

// invoiceOptions is the query of the order's lines with what non-cancelled invoices leave.
export const invoiceOptions = (order: number) => ({
  queryKey: salesKeys.invoices.options(order),
  queryFn: () => unwrap(api.GET('/sales/orders/{id}/invoice-options', { params: { path: { id: order } } })),
})

export function useReceipt(id: number) {
  return useQuery({
    queryKey: salesKeys.receipts.detail(id),
    queryFn: () => unwrap(api.GET('/sales/receipts/{id}', { params: { path: { id } } })),
  })
}

// openInvoices is the query of a customer's posted invoices at an org unit with something left
// to receive; receipt is the draft being edited, 0 for none.
export const openInvoices = (customer: number, unit: number, receipt: number) => ({
  queryKey: salesKeys.receipts.open(customer, unit, receipt),
  queryFn: async () =>
    (
      await unwrap(
        api.GET('/sales/customers/{id}/open-invoices', { params: { path: { id: customer }, query: { org_unit_id: unit, receipt_id: receipt || undefined } } }),
      )
    ).items,
})
