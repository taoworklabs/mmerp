import { accounting } from '@/accounting'
import { core } from '@/core'
import { hrm, hrmMe } from '@/hrm'
import { purchase } from '@/purchase'
import { sales } from '@/sales'

// Order of the home tiles: product areas in this order, then core (administration) last.
export const areas = [hrm, hrmMe, sales, purchase, accounting, core]
