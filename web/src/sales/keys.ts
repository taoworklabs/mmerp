import type { Doc } from '@/shared/api/sales'

// The record types of the sales area, as registered by the backend.
export const quoteDocType = 'sales.quote'
export const orderDocType = 'sales.order'
export const customerDocType = 'sales.customer'

// A quotation or a sales order: the two share every screen.
export type Kind = Doc['kind']
export const docTypeOf = (k: Kind) => (k === 'quote' ? quoteDocType : orderDocType)
export const pathOf = (k: Kind) => (k === 'quote' ? 'quotes' : 'orders')

// Query keys of the sales area; every parameter that changes a result is part of its key.
export type CustomerQuery = { q: string; active: string; sort: string; page: number; pageSize: number }
export type ItemQuery = { q: string; active: string; page: number; pageSize: number }
export type DocQuery = { q: string; status: string; sort: string; page: number; pageSize: number }

export const salesKeys = {
  customers: {
    all: () => ['sales', 'customers'] as const,
    actions: () => ['sales', 'customers', 'actions'] as const,
    lists: () => ['sales', 'customers', 'list'] as const,
    list: (q: CustomerQuery) => ['sales', 'customers', 'list', q] as const,
    picker: (q: string) => ['sales', 'customers', 'picker', q] as const,
    detail: (id: number) => ['sales', 'customers', id] as const,
  },
  items: {
    all: () => ['sales', 'items'] as const,
    list: (q: ItemQuery) => ['sales', 'items', 'list', q] as const,
    picker: (q: string) => ['sales', 'items', 'picker', q] as const,
  },
  docs: {
    all: (k: Kind) => ['sales', k] as const,
    actions: (k: Kind) => ['sales', k, 'actions'] as const,
    lists: (k: Kind) => ['sales', k, 'list'] as const,
    list: (k: Kind, q: DocQuery) => ['sales', k, 'list', q] as const,
    detail: (k: Kind, id: number) => ['sales', k, 'detail', id] as const,
  },
}
