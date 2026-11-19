import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '@/shared/api/purchase'
import { purchaseKeys } from '../keys'

// useSupplierActions and useOrderActions: what the user may do before picking a record (create),
// as the API decides.
export function useSupplierActions() {
  return useQuery({
    queryKey: purchaseKeys.suppliers.actions(),
    queryFn: async () => (await unwrap(api.GET('/purchase/suppliers/actions'))).allowed_actions,
  })
}

export function useOrderActions() {
  return useQuery({
    queryKey: purchaseKeys.orders.actions(),
    queryFn: async () => (await unwrap(api.GET('/purchase/orders/actions'))).allowed_actions,
  })
}

export function useSupplier(id: number) {
  return useQuery({
    queryKey: purchaseKeys.suppliers.detail(id),
    queryFn: () => unwrap(api.GET('/purchase/suppliers/{id}', { params: { path: { id } } })),
  })
}

export function useOrder(id: number, enabled = true) {
  return useQuery({
    enabled,
    queryKey: purchaseKeys.orders.detail(id),
    queryFn: () => unwrap(api.GET('/purchase/orders/{id}', { params: { path: { id } } })),
  })
}

// useSupplierOrders: a supplier's latest orders, newest first, and how many there are.
export function useSupplierOrders(supplier: number) {
  return useQuery({
    queryKey: purchaseKeys.orders.supplier(supplier),
    // ponytail: the latest 100 only; page the tab if a supplier ever has more
    queryFn: () => unwrap(api.GET('/purchase/orders', { params: { query: { supplier_id: supplier, sort: '-date', page_size: 100 } } })),
  })
}

export function useGoodsReceipt(id: number) {
  return useQuery({
    queryKey: purchaseKeys.goodsReceipts.detail(id),
    queryFn: () => unwrap(api.GET('/purchase/goods-receipts/{id}', { params: { path: { id } } })),
  })
}

// receiptOptions is the query of the warehouses and what is left to receive of the order.
export const receiptOptions = (order: number) => ({
  queryKey: purchaseKeys.goodsReceipts.options(order),
  queryFn: () => unwrap(api.GET('/purchase/orders/{id}/receipt-options', { params: { path: { id: order } } })),
})

export function useInvoice(id: number, enabled = true) {
  return useQuery({
    enabled,
    queryKey: purchaseKeys.invoices.detail(id),
    queryFn: () => unwrap(api.GET('/purchase/invoices/{id}', { params: { path: { id } } })),
  })
}

// invoiceOptions is the query of the order's lines with what non-cancelled invoices leave.
export const invoiceOptions = (order: number) => ({
  queryKey: purchaseKeys.invoices.options(order),
  queryFn: () => unwrap(api.GET('/purchase/orders/{id}/invoice-options', { params: { path: { id: order } } })),
})

export function usePayment(id: number) {
  return useQuery({
    queryKey: purchaseKeys.payments.detail(id),
    queryFn: () => unwrap(api.GET('/purchase/payments/{id}', { params: { path: { id } } })),
  })
}

// openInvoices is the query of a supplier's posted invoices at an org unit with something left
// to pay; payment is the draft being edited, 0 for none.
export const openInvoices = (supplier: number, unit: number, payment: number) => ({
  queryKey: purchaseKeys.payments.open(supplier, unit, payment),
  queryFn: async () =>
    (
      await unwrap(
        api.GET('/purchase/suppliers/{id}/open-invoices', {
          params: { path: { id: supplier }, query: { org_unit_id: unit, payment_id: payment || undefined } },
        }),
      )
    ).items,
})

export function usePaymentActions() {
  return useQuery({
    queryKey: purchaseKeys.payments.actions(),
    queryFn: async () => (await unwrap(api.GET('/purchase/payments/actions'))).allowed_actions,
  })
}
