// Pure helpers of the stock screens.

// monthToDate is the stock card's default range: the first of today's month to today.
export const monthToDate = (today: string) => ({ from: `${today.slice(0, 7)}-01`, to: today })

// stockCardPath opens the stock card of an item in a warehouse over the current month.
export function stockCardPath(warehouseId: number, itemId: number, today: string): string {
  const { from, to } = monthToDate(today)
  return `/sales/stock/card?${new URLSearchParams({ warehouse_id: String(warehouseId), item_id: String(itemId), from, to })}`
}

// A lot is expired once its expiry is before today; both are YYYY-MM-DD, so strings compare.
export const isExpired = (expiry: string | undefined, today: string) => !!expiry && expiry < today
