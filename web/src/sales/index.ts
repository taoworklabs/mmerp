import { IconAddressBook, IconBuildingWarehouse, IconFileDollar, IconGitBranch, IconListDetails, IconPackage, IconPackages, IconReceipt, IconShoppingCart } from '@tabler/icons-react'
import type { AreaManifest } from '@/shared/area'
import { masterKeys } from '@/shared/master'
import { customerDocType, orderDocType, partnerType, quoteDocType, salesKeys } from './keys'

export const sales: AreaManifest = {
  product: 'sales',
  label: 'sales.nav.group',
  basePath: '/sales',
  routes: () => import('./routes'),
  nav: [
    { label: 'sales.nav.customers', path: 'customers', icon: IconAddressBook, permission: 'sales.customer.view' },
    { label: 'sales.nav.quotes', path: 'quotes', icon: IconFileDollar, permission: 'sales.quote.view' },
    { label: 'sales.nav.orders', path: 'orders', icon: IconReceipt, permission: 'sales.order.view' },
    { label: 'sales.nav.on_hand', path: 'stock/on-hand', icon: IconPackages, permission: 'stock.view', group: 'sales.nav.group_stock' },
    { label: 'sales.nav.stock_card', path: 'stock/card', icon: IconListDetails, permission: 'stock.view', group: 'sales.nav.group_stock' },
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
    [customerDocType]: {
      path: (id) => `customers/${id}`,
      // Names show in pickers too, so every customer query; its identity is its partner's.
      invalidate: () => [salesKeys.customers.all(), masterKeys.partners.all()],
    },
    // Partners open in Sales for now; their identity shows on every customer of theirs.
    [partnerType]: {
      path: (id) => `partners/${id}`,
      invalidate: () => [salesKeys.customers.all(), masterKeys.partners.all()],
    },
  },
}
