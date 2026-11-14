import type { TFunction } from 'i18next'

// splitBalance puts a signed balance (debit minus credit) on its side: positive is a debit balance.
export const splitBalance = (n: number) => ({ debit: Math.max(n, 0), credit: Math.max(-n, 0) })

// kindLabel names a posting kind; a kind added after these screens shows as its code.
export const kindLabel = (t: TFunction, kind: string): string => t(`accounting.kind.${kind}`, { defaultValue: kind })

// rowBalance is one side of a row's opening or closing balance; a totals row shows none, since
// adding signed balances would net debit balances against credit ones, and accounts together.
export const rowBalance = (r: { total?: boolean; opening: number; closing: number }, of: 'opening' | 'closing', side: 'debit' | 'credit'): number | null =>
  r.total ? null : splitBalance(r[of])[side]

type SavedLine = { partner_id?: number | null; partner_code?: string | null; partner_name?: string | null }

// savedPartner is the saved line's partner as a picker option, when the line now at its index
// still holds that partner (lines may have been removed or added since).
export const savedPartner = (l: SavedLine | undefined, partnerId: string | null) =>
  l?.partner_id && l.partner_code && l.partner_name && String(l.partner_id) === partnerId ? { id: l.partner_id, code: l.partner_code, name: l.partner_name } : undefined
