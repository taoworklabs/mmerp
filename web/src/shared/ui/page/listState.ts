export type ListQuery<T> = { isPending: boolean; isError: boolean; data?: { items: T[]; total: number } }

// listState picks what a list shows; an empty list the filters caused offers to clear them instead.
export function listState<T>(query: ListQuery<T>, filtered: boolean) {
  if (query.isPending) return 'loading'
  if (query.isError || !query.data) return 'error'
  if (query.data.items.length > 0) return 'rows'
  return filtered ? 'no_results' : 'empty'
}
