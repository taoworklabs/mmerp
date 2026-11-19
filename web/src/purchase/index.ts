import { IconAddressBook, IconCash, IconFileInvoice, IconGitBranch, IconPackageImport, IconReceipt, IconShoppingBag } from '@tabler/icons-react'
import type { AreaManifest } from '@/shared/area'
import { masterKeys } from '@/shared/master'
import { goodsReceiptDocType, invoiceDocType, orderDocType, paymentDocType, purchaseKeys, supplierDocType } from './keys'

export const purchase: AreaManifest = {
  product: 'purchase',
  label: 'purchase.nav.group',
  basePath: '/purchase',
  routes: () => import('./routes'),
  nav: [
    { label: 'purchase.nav.suppliers', path: 'suppliers', icon: IconAddressBook, permission: 'purchase.supplier.view' },
    { label: 'purchase.nav.orders', path: 'orders', icon: IconReceipt, permission: 'purchase.order.view' },
    { label: 'purchase.nav.goods_receipts', path: 'goods-receipts', icon: IconPackageImport, permission: 'purchase.order.view' },
    { label: 'purchase.nav.invoices', path: 'invoices', icon: IconFileInvoice, permission: 'purchase.order.view' },
    { label: 'purchase.nav.payments', path: 'payments', icon: IconCash, permission: 'purchase.order.view' },
    {
      label: 'purchase.nav.approval_rules',
      path: 'approval-rules',
      icon: IconGitBranch,
      permission: ['purchase.approval.manage', 'core.approval.manage'],
      group: 'purchase.nav.group_config',
    },
  ],
  collapsibleGroups: ['purchase.nav.group_config'],
  homeIcon: IconShoppingBag,
  i18n: {
    meta: { vi: () => import('./i18n/vi/meta.json'), en: () => import('./i18n/en/meta.json') },
    main: { vi: () => import('./i18n/vi/main.json'), en: () => import('./i18n/en/main.json') },
  },
  recordTypes: {
    [orderDocType]: {
      path: (id) => `orders/${id}`,
      preview: () => import('./components/OrderPreview'),
      invalidate: () => [purchaseKeys.orders.all(), purchaseKeys.invoices.all()],
    },
    [goodsReceiptDocType]: {
      path: (id) => `goods-receipts/${id}`,
      preview: () => import('./components/GoodsReceiptPreview'),
      // Its order shows what is received.
      invalidate: () => [purchaseKeys.goodsReceipts.all(), purchaseKeys.orders.all()],
    },
    [invoiceDocType]: {
      path: (id) => `invoices/${id}`,
      preview: () => import('./components/InvoicePreview'),
      // Its order shows what is invoiced and paid.
      invalidate: () => [purchaseKeys.invoices.all(), purchaseKeys.orders.all()],
    },
    [paymentDocType]: {
      path: (id) => `payments/${id}`,
      preview: () => import('./components/PaymentPreview'),
      // Its invoices and their orders show what is paid.
      invalidate: () => [purchaseKeys.payments.all(), purchaseKeys.invoices.all(), purchaseKeys.orders.all()],
    },
    [supplierDocType]: {
      path: (id) => `suppliers/${id}`,
      // Names show in pickers too, so every supplier query; its identity is its partner's.
      invalidate: () => [purchaseKeys.suppliers.all(), masterKeys.partners.all()],
    },
  },
}
