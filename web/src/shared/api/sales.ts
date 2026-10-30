// The Sales API as the sales area sees it.
import type { components, operations } from './schema.gen'

import { productClient } from './client'

export { unwrap } from './client'
export const api = productClient<'/sales/'>()
export { ApiError } from './error'

export type Customer = components['schemas']['Customer']
export type CustomerUpdate = components['schemas']['CustomerUpdate']
export type NewCustomer = components['schemas']['NewCustomer']
export type CustomerListItem = components['schemas']['CustomerListItem']
export type LineItem = components['schemas']['LineItem']
export type CustomerSort = NonNullable<NonNullable<operations['list-customers']['parameters']['query']>['sort']>
export type Doc = components['schemas']['Doc']
// The editable fields of a quotation or order, as both create and update send them.
export type DocFields = Omit<components['schemas']['NewDoc'], '$schema' | 'request_id'>
export type DocListItem = components['schemas']['DocListItem']
export type LineInput = components['schemas']['LineInput']
export type DocSort = NonNullable<NonNullable<operations['list-quotes']['parameters']['query']>['sort']>
export type VatRate = LineInput['vat_rate']
export type Delivery = components['schemas']['Delivery']
export type DeliveryOptions = components['schemas']['DeliveryOptions']
export type DeliveryLineInput = components['schemas']['DeliveryLineInput']
export type DeliveryListItem = components['schemas']['DeliveryListItem']
export type DeliverySort = NonNullable<NonNullable<operations['list-deliveries']['parameters']['query']>['sort']>
