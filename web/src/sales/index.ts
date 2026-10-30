import { IconAddressBook, IconAdjustmentsHorizontal, IconBuildingWarehouse, IconFileDollar, IconTruckDelivery, IconGitBranch, IconListDetails, IconPackage, IconPackages, IconReceipt, IconShoppingCart } from '@tabler/icons-react'
import type { AreaManifest } from '@/shared/area'
import { masterKeys } from '@/shared/master'
import { adjustmentDocType, customerDocType, deliveryDocType, orderDocType, partnerType, quoteDocType, salesKeys } from './keys'

export const sales: AreaManifest = {
  product: 'sales',
  label: 'sales.nav.group',
  basePath: '/sales',
  routes: () => import('./routes'),
  nav: [
    { label: 'sales.nav.customers', path: 'customers', icon: IconAddressBook, permission: 'sales.customer.view' },
    { label: 'sales.nav.quotes', path: 'quotes', icon: IconFileDollar, permission: 'sales.quote.view' },
    { label: 'sales.nav.orders', path: 'orders', icon: IconReceipt, permission: 'sales.order.view' },
    { label: 'sales.nav.deliveries', path: 'deliveries', icon: IconTruckDelivery, permission: 'sales.order.view' },
    { label: 'sales.nav.on_hand', path: 'stock/on-hand', icon: IconPackages, permission: 'stock.view', group: 'sales.nav.group_stock' },
    { label: 'sales.nav.stock_card', path: 'stock/card', icon: IconListDetails, permission: 'stock.view', group: 'sales.nav.group_stock' },
    { label: 'sales.nav.adjustments', path: 'stock/adjustments', icon: IconAdjustmentsHorizontal, permission: 'stock.view', group: 'sales.nav.group_stock' },
    { label: 'sales.nav.warehouses', path: 'stock/warehouses', icon: IconBuildingWarehouse, permission: 'stock.view', group: 'sales.nav.group_stock' },
    { label: 'sales.nav.items', path: 'items', icon: IconPackage, permission: 'item.view', group: 'sales.nav.group_config' },
    {
      label: 'sales.nav.approval_rules',
      path: 'approval-rules',
      icon: IconGitBranch,
      permission: ['sales.approval.manage', 'core.approval.manage'],
      group: 'sales.nav.group_config',
    },
  ],
  collapsibleGroups: ['sales.nav.group_stock', 'sales.nav.group_config'],
  homeIcon: IconShoppingCart,
  i18n: {
    meta: { vi: () => import('./i18n/vi/meta.json'), en: () => import('./i18n/en/meta.json') },
    main: { vi: () => import('./i18n/vi/main.json'), en: () => import('./i18n/en/main.json') },
  },
  recordTypes: {
    [quoteDocType]: {
      path: (id) => `quotes/${id}`,
      preview: () => import('./components/QuotePreview'),
      // A quotation's order shows on it, and the order shows its quotation.
      invalidate: () => [salesKeys.docs.all('quote'), salesKeys.docs.all('order')],
    },
    [orderDocType]: {
      path: (id) => `orders/${id}`,
      preview: () => import('./components/OrderPreview'),
      invalidate: () => [salesKeys.docs.all('order'), salesKeys.docs.all('quote')],
    },
    [deliveryDocType]: {
      path: (id) => `deliveries/${id}`,
      preview: () => import('./components/DeliveryPreview'),
      // Its order shows what is delivered, and posting or cancelling one moves stock.
      invalidate: () => [salesKeys.deliveries.all(), salesKeys.docs.all('order'), salesKeys.stock.onHand.all(), salesKeys.stock.cards()],
    },
    [customerDocType]: {
      path: (id) => `customers/${id}`,
      // Names show in pickers too, so every customer query; its identity is its partner's.
      invalidate: () => [salesKeys.customers.all(), masterKeys.partners.all()],
    },
    [adjustmentDocType]: {
      path: (id) => `stock/adjustments/${id}`,
      preview: () => import('./components/AdjustmentPreview'),
      // Posting or cancelling one moves stock, which delivery options and draft orders' availability read.
      invalidate: () => [
        salesKeys.stock.adjustments.all(),
        salesKeys.stock.onHand.all(),
        salesKeys.stock.cards(),
        salesKeys.deliveries.all(),
        salesKeys.docs.all('order'),
      ],
    },
    // Partners open in Sales for now; their identity shows on every customer of theirs.
    [partnerType]: {
      path: (id) => `partners/${id}`,
      invalidate: () => [salesKeys.customers.all(), masterKeys.partners.all()],
    },
  },
}
