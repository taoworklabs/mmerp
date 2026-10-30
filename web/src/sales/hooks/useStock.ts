import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '@/shared/api/core'
import type { Option } from '@/shared/ui/form'
import { salesKeys } from '../keys'

// useWarehouseOptions lists the active warehouses, "code · name", for a filter or a picker.
export function useWarehouseOptions(): Option[] {
  // ponytail: one page of 100 warehouses; a picker with search when tenants have more.
  const query = { q: '', active: 'true', sort: 'code', page: 1, pageSize: 100 }
  const list = useQuery({
    queryKey: salesKeys.stock.warehouses.list(query),
    queryFn: () => unwrap(api.GET('/stock/warehouses', { params: { query: { active: 'true', sort: 'code', page_size: 100 } } })),
  })
  return (list.data?.items ?? []).map((w) => ({ value: String(w.id), label: `${w.code} · ${w.name}` }))
}

// useAdjustmentActions: what the user may do before picking a stock adjustment (create).
export function useAdjustmentActions() {
  return useQuery({
    queryKey: salesKeys.stock.adjustments.actions(),
    queryFn: async () => (await unwrap(api.GET('/stock/adjustments/actions'))).allowed_actions,
  })
}

export function useAdjustment(id: number) {
  return useQuery({
    queryKey: salesKeys.stock.adjustments.detail(id),
    queryFn: () => unwrap(api.GET('/stock/adjustments/{id}', { params: { path: { id } } })),
  })
}
