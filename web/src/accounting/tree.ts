import type { Account } from '@/shared/api/accounting'

type Node = { id: number; parent_id?: number | null }

// subtree is the ids of root and every node under it.
export function subtree(items: Node[], root: number): Set<number> {
  const out = new Set([root])
  // Repeat until no node joins: parents may come after their children in the list.
  for (let grew = true; grew; ) {
    grew = false
    for (const n of items)
      if (n.parent_id != null && out.has(n.parent_id) && !out.has(n.id)) {
        out.add(n.id)
        grew = true
      }
  }
  return out
}

// parentCandidates are the accounts an account of this type may hang under: same type, never
// itself or one of its descendants.
export function parentCandidates(accounts: Account[], type: string, self?: number): Account[] {
  const own = self ? subtree(accounts, self) : new Set<number>()
  return accounts.filter((a) => a.type === type && !own.has(a.id))
}
