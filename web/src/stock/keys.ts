// The record types of the stock area, as registered by the backend.
export const adjustmentDocType = 'stock.adjustment'
export const transferDocType = 'stock.transfer'
export const stocktakeDocType = 'stock.stocktake'

// Query keys of the stock area; every parameter that changes a result is part of its key.
export type ItemQuery = { q: string; active: string; page: number; pageSize: number }
export type WarehouseQuery = { q: string; active: string; sort: string; page: number; pageSize: number }
export type OnHandQuery = { warehouse: string; item: string; q: string; expiring: string; page: number; pageSize: number }
export type StockCardQuery = { warehouse_id: string; item_id: string; from: string; to: string }
export type DocQuery = { q: string; status: string; sort: string; page: number; pageSize: number }
export type StockDocQuery = DocQuery & { warehouse: string }

// docKeys are the keys of one stock document type.
const docKeys = (name: string) => ({
  all: () => ['stock', name] as const,
  actions: () => ['stock', name, 'actions'] as const,
  list: (q: DocQuery | StockDocQuery) => ['stock', name, 'list', q] as const,
  detail: (id: number) => ['stock', name, 'detail', id] as const,
})

export const stockKeys = {
  items: {
    all: () => ['stock', 'items'] as const,
    list: (q: ItemQuery) => ['stock', 'items', 'list', q] as const,
  },
  // Items with stock, for document lines and the stock card.
  pickerItems: (q: string) => ['stock', 'picker-items', q] as const,
  warehouses: {
    all: () => ['stock', 'warehouses'] as const,
    list: (q: WarehouseQuery) => ['stock', 'warehouses', 'list', q] as const,
  },
  onHand: {
    all: () => ['stock', 'on-hand'] as const,
    list: (q: OnHandQuery) => ['stock', 'on-hand', 'list', q] as const,
    lines: (warehouse: number, item: number) => ['stock', 'on-hand', 'lines', warehouse, item] as const,
  },
  card: (q: StockCardQuery) => ['stock', 'card', q] as const,
  // Every stock card, whatever its item and range.
  cards: () => ['stock', 'card'] as const,
  adjustments: docKeys('adjustments'),
  transfers: docKeys('transfers'),
  stocktakes: docKeys('stocktakes'),
}
