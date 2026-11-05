import type { TFunction } from 'i18next'

// splitBalance puts a signed balance (debit minus credit) on its side: positive is a debit balance.
export const splitBalance = (n: number) => ({ debit: Math.max(n, 0), credit: Math.max(-n, 0) })

// kindLabel names a posting kind; a kind added after these screens shows as its code.
export const kindLabel = (t: TFunction, kind: string): string => t(`accounting.kind.${kind}`, { defaultValue: kind })
