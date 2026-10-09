// The Sales API as the sales area sees it.
import type { components, operations } from './schema.gen'

import { productClient } from './client'

export { unwrap } from './client'
export const api = productClient<'/sales/'>()
export { ApiError } from './error'

export type Customer = components['schemas']['Customer']
export type CustomerFields = components['schemas']['CustomerFields']
export type CustomerListItem = components['schemas']['CustomerListItem']
export type CustomerSort = NonNullable<NonNullable<operations['list-customers']['parameters']['query']>['sort']>
export type Item = components['schemas']['Item']
export type ItemFields = components['schemas']['ItemFields']
export type Doc = components['schemas']['Doc']
export type DocFields = components['schemas']['DocFields']
export type DocListItem = components['schemas']['DocListItem']
export type LineInput = components['schemas']['LineInput']
export type DocSort = NonNullable<NonNullable<operations['list-quotes']['parameters']['query']>['sort']>
export type VatRate = LineInput['vat_rate']
