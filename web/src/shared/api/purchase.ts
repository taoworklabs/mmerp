// The Purchasing API as the purchase area sees it.
import type { components, operations } from './schema.gen'

import { productClient } from './client'

export { unwrap } from './client'
export const api = productClient<'/purchase/'>()
export { ApiError } from './error'

export type Supplier = components['schemas']['PurchaseSupplier']
export type SupplierUpdate = components['schemas']['PurchaseSupplierUpdate']
export type NewSupplier = components['schemas']['PurchaseNewSupplier']
export type SupplierListItem = components['schemas']['PurchaseSupplierListItem']
export type SupplierSort = NonNullable<NonNullable<operations['list-suppliers']['parameters']['query']>['sort']>
export type LineItem = components['schemas']['PurchaseLineItem']
export type Order = components['schemas']['PurchaseOrder']
// The editable fields of an order, as both create and update send them.
export type OrderFields = Omit<components['schemas']['PurchaseNewOrder'], '$schema' | 'request_id'>
export type OrderListItem = components['schemas']['PurchaseOrderListItem']
export type LineInput = components['schemas']['PurchaseLineInput']
export type OrderSort = NonNullable<NonNullable<operations['list-purchase-orders']['parameters']['query']>['sort']>
export type VatRate = LineInput['vat_rate']
export type GoodsReceipt = components['schemas']['PurchaseGoodsReceipt']
export type ReceiptOptions = components['schemas']['PurchaseReceiptOptions']
export type ReceiptLineInput = components['schemas']['PurchaseReceiptLineInput']
export type GoodsReceiptListItem = components['schemas']['PurchaseGoodsReceiptListItem']
export type GoodsReceiptSort = NonNullable<NonNullable<operations['list-goods-receipts']['parameters']['query']>['sort']>
export type Invoice = components['schemas']['PurchaseInvoice']
export type InvoiceOptions = components['schemas']['PurchaseInvoiceOptions']
export type InvoiceLineInput = components['schemas']['PurchaseInvoiceLineInput']
export type InvoiceListItem = components['schemas']['PurchaseInvoiceListItem']
export type InvoiceSort = NonNullable<NonNullable<operations['list-purchase-invoices']['parameters']['query']>['sort']>
export type PaymentState = Invoice['payment_state']
export type Payment = components['schemas']['PurchasePayment']
export type AllocationInput = components['schemas']['PurchaseAllocationInput']
export type Allocation = components['schemas']['PurchaseAllocation']
export type PaymentListItem = components['schemas']['PurchasePaymentListItem']
export type PaymentSort = NonNullable<NonNullable<operations['list-payments']['parameters']['query']>['sort']>
