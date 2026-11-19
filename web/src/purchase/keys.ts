// The record types of the purchase area, as registered by the backend.
export const orderDocType = 'purchase.order'
export const supplierDocType = 'purchase.supplier'
export const goodsReceiptDocType = 'purchase.goods_receipt'
export const invoiceDocType = 'purchase.invoice'
export const paymentDocType = 'purchase.payment'

// Query keys of the purchase area; every parameter that changes a result is part of its key.
export type SupplierQuery = { q: string; active: string; sort: string; page: number; pageSize: number }
export type DocQuery = { q: string; status: string; sort: string; page: number; pageSize: number }
export type OrderQuery = DocQuery & { receipt_state: string; invoice_state?: string; payment_state?: string }
export type GoodsReceiptQuery = DocQuery & { order: string }
export type InvoiceQuery = DocQuery & { order: string; payment_state: string }
export type PaymentQuery = DocQuery & { method: string }

export const purchaseKeys = {
  suppliers: {
    all: () => ['purchase', 'suppliers'] as const,
    actions: () => ['purchase', 'suppliers', 'actions'] as const,
    lists: () => ['purchase', 'suppliers', 'list'] as const,
    list: (q: SupplierQuery) => ['purchase', 'suppliers', 'list', q] as const,
    picker: (q: string) => ['purchase', 'suppliers', 'picker', q] as const,
    // Under lists, so whatever refreshes the lists refreshes it too.
    partner: (partner: number) => ['purchase', 'suppliers', 'list', 'partner', partner] as const,
    detail: (id: number) => ['purchase', 'suppliers', id] as const,
  },
  items: {
    picker: (q: string) => ['purchase', 'items', 'picker', q] as const,
  },
  goodsReceipts: {
    all: () => ['purchase', 'goods-receipts'] as const,
    list: (q: GoodsReceiptQuery) => ['purchase', 'goods-receipts', 'list', q] as const,
    detail: (id: number) => ['purchase', 'goods-receipts', 'detail', id] as const,
    // Under all, so a saved receipt refreshes what is left to receive.
    options: (order: number) => ['purchase', 'goods-receipts', 'options', order] as const,
  },
  invoices: {
    all: () => ['purchase', 'invoices'] as const,
    list: (q: InvoiceQuery) => ['purchase', 'invoices', 'list', q] as const,
    detail: (id: number) => ['purchase', 'invoices', 'detail', id] as const,
    // Under all, so a saved invoice refreshes what is left to bill.
    options: (order: number) => ['purchase', 'invoices', 'options', order] as const,
  },
  payments: {
    all: () => ['purchase', 'payments'] as const,
    actions: () => ['purchase', 'payments', 'actions'] as const,
    list: (q: PaymentQuery) => ['purchase', 'payments', 'list', q] as const,
    detail: (id: number) => ['purchase', 'payments', 'detail', id] as const,
    // Under invoices, so a saved payment or invoice refreshes what is left to pay.
    open: (supplier: number, unit: number, payment: number) => ['purchase', 'invoices', 'open', supplier, unit, payment] as const,
  },
  orders: {
    all: () => ['purchase', 'orders'] as const,
    actions: () => ['purchase', 'orders', 'actions'] as const,
    lists: () => ['purchase', 'orders', 'list'] as const,
    list: (q: OrderQuery) => ['purchase', 'orders', 'list', q] as const,
    detail: (id: number) => ['purchase', 'orders', 'detail', id] as const,
    // Under lists, so whatever refreshes the lists refreshes it too.
    supplier: (supplier: number) => ['purchase', 'orders', 'list', 'supplier', supplier] as const,
  },
}
