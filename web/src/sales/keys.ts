import type { Doc } from '@/shared/api/sales'

// The record types of the sales area, as registered by the backend.
export const quoteDocType = 'sales.quote'
export const orderDocType = 'sales.order'
export const customerDocType = 'sales.customer'
export const deliveryDocType = 'sales.delivery'
// Stock is a shared module; its documents open in this area.
export const adjustmentDocType = 'stock.adjustment'
// Shared master data, opened in this area.
export const partnerType = 'partner.partner'

// A quotation or a sales order: the two share every screen.
export type Kind = Doc['kind']
export const docTypeOf = (k: Kind) => (k === 'quote' ? quoteDocType : orderDocType)
export const pathOf = (k: Kind) => (k === 'quote' ? 'quotes' : 'orders')

// Query keys of the sales area; every parameter that changes a result is part of its key.
export type CustomerQuery = { q: string; active: string; sort: string; page: number; pageSize: number }
export type ItemQuery = { q: string; active: string; page: number; pageSize: number }
export type WarehouseQuery = { q: string; active: string; sort: string; page: number; pageSize: number }
export type OnHandQuery = { warehouse: string; item: string; q: string; expiring: string; page: number; pageSize: number }
export type StockCardQuery = { warehouse_id: string; item_id: string; from: string; to: string }
export type DocQuery = { q: string; status: string; sort: string; page: number; pageSize: number }
export type OrderQuery = DocQuery & { delivery_state: string }
export type DeliveryQuery = DocQuery & { order: string }

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
    all: () => ['sales', 'items'] as const,
    list: (q: ItemQuery) => ['sales', 'items', 'list', q] as const,
    picker: (q: string) => ['sales', 'items', 'picker', q] as const,
  },
  // Stock is a shared module; its screens live in this area.
  stock: {
    warehouses: {
      all: () => ['sales', 'stock', 'warehouses'] as const,
      list: (q: WarehouseQuery) => ['sales', 'stock', 'warehouses', 'list', q] as const,
    },
    onHand: {
      all: () => ['sales', 'stock', 'on-hand'] as const,
      list: (q: OnHandQuery) => ['sales', 'stock', 'on-hand', 'list', q] as const,
    },
    card: (q: StockCardQuery) => ['sales', 'stock', 'card', q] as const,
    // Every stock card, whatever its item and range.
    cards: () => ['sales', 'stock', 'card'] as const,
    adjustments: {
      all: () => ['sales', 'stock', 'adjustments'] as const,
      actions: () => ['sales', 'stock', 'adjustments', 'actions'] as const,
      list: (q: DocQuery) => ['sales', 'stock', 'adjustments', 'list', q] as const,
      detail: (id: number) => ['sales', 'stock', 'adjustments', 'detail', id] as const,
    },
    // The core item list, unlike items.picker which searches document line items.
    items: (q: string) => ['sales', 'stock', 'items', q] as const,
  },
  deliveries: {
    all: () => ['sales', 'deliveries'] as const,
    list: (q: DeliveryQuery) => ['sales', 'deliveries', 'list', q] as const,
    detail: (id: number) => ['sales', 'deliveries', 'detail', id] as const,
    // Under all, so a saved delivery refreshes what is left to deliver.
    options: (order: number, warehouse: number) => ['sales', 'deliveries', 'options', order, warehouse] as const,
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
