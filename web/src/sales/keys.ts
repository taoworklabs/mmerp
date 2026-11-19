import type { Doc } from '@/shared/api/sales'

// The record types of the sales area, as registered by the backend.
export const quoteDocType = 'sales.quote'
export const orderDocType = 'sales.order'
export const customerDocType = 'sales.customer'
export const deliveryDocType = 'sales.delivery'
export const invoiceDocType = 'sales.invoice'
export const receiptDocType = 'sales.receipt'
// Shared master data, opened in this area.
export const partnerType = 'partner.partner'

// A quotation or a sales order: the two share every screen.
export type Kind = Doc['kind']
export const docTypeOf = (k: Kind) => (k === 'quote' ? quoteDocType : orderDocType)
export const pathOf = (k: Kind) => (k === 'quote' ? 'quotes' : 'orders')

// Query keys of the sales area; every parameter that changes a result is part of its key.
export type CustomerQuery = { q: string; active: string; sort: string; page: number; pageSize: number }
export type DocQuery = { q: string; status: string; sort: string; page: number; pageSize: number }
export type OrderQuery = DocQuery & { delivery_state: string; invoice_state?: string; payment_state?: string }
export type DeliveryQuery = DocQuery & { order: string }
export type InvoiceQuery = DocQuery & { order: string; payment_state: string }
export type ReceiptQuery = DocQuery & { method: string }

export const salesKeys = {
  customers: {
    all: () => ['sales', 'customers'] as const,
    actions: () => ['sales', 'customers', 'actions'] as const,
    lists: () => ['sales', 'customers', 'list'] as const,
    list: (q: CustomerQuery) => ['sales', 'customers', 'list', q] as const,
    picker: (q: string) => ['sales', 'customers', 'picker', q] as const,
    // Under lists, so whatever refreshes the lists refreshes it too.
    partner: (partner: number) => ['sales', 'customers', 'list', 'partner', partner] as const,
    detail: (id: number) => ['sales', 'customers', id] as const,
  },
  items: {
    picker: (q: string) => ['sales', 'items', 'picker', q] as const,
  },
  deliveries: {
    all: () => ['sales', 'deliveries'] as const,
    list: (q: DeliveryQuery) => ['sales', 'deliveries', 'list', q] as const,
    detail: (id: number) => ['sales', 'deliveries', 'detail', id] as const,
    // Under all, so a saved delivery refreshes what is left to deliver.
    options: (order: number, warehouse: number) => ['sales', 'deliveries', 'options', order, warehouse] as const,
  },
  invoices: {
    all: () => ['sales', 'invoices'] as const,
    list: (q: InvoiceQuery) => ['sales', 'invoices', 'list', q] as const,
    detail: (id: number) => ['sales', 'invoices', 'detail', id] as const,
    // Under all, so a saved invoice refreshes what is left to bill.
    options: (order: number) => ['sales', 'invoices', 'options', order] as const,
  },
  receipts: {
    all: () => ['sales', 'receipts'] as const,
    actions: () => ['sales', 'receipts', 'actions'] as const,
    list: (q: ReceiptQuery) => ['sales', 'receipts', 'list', q] as const,
    detail: (id: number) => ['sales', 'receipts', 'detail', id] as const,
    // Under invoices, so a saved receipt or invoice refreshes what is left to receive.
    open: (customer: number, unit: number, receipt: number) => ['sales', 'invoices', 'open', customer, unit, receipt] as const,
  },
  docs: {
    all: (k: Kind) => ['sales', k] as const,
    actions: (k: Kind) => ['sales', k, 'actions'] as const,
    lists: (k: Kind) => ['sales', k, 'list'] as const,
    list: (k: Kind, q: OrderQuery) => ['sales', k, 'list', q] as const,
    detail: (k: Kind, id: number) => ['sales', k, 'detail', id] as const,
    // Under lists, so whatever refreshes the lists refreshes it too.
    customer: (k: Kind, customer: number) => ['sales', k, 'list', 'customer', customer] as const,
  },
}
