import { core } from '@/core'
import { hrm } from '@/hrm'
import { sales } from '@/sales'

// Order of the home tiles: product areas in this order, then core (administration) last.
export const areas = [hrm, sales, core]
