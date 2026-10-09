import { IconAddressBook, IconFileDollar, IconGitBranch, IconPackage, IconReceipt, IconShoppingCart } from '@tabler/icons-react'
import type { AreaManifest } from '@/shared/area'
import { customerDocType, orderDocType, quoteDocType, salesKeys } from './keys'

export const sales: AreaManifest = {
  product: 'sales',
  label: 'sales.nav.group',
  basePath: '/sales',
  routes: () => import('./routes'),
  nav: [
    { label: 'sales.nav.customers', path: 'customers', icon: IconAddressBook, permission: 'sales.customer.view' },
    { label: 'sales.nav.quotes', path: 'quotes', icon: IconFileDollar, permission: 'sales.quote.view' },
    { label: 'sales.nav.orders', path: 'orders', icon: IconReceipt, permission: 'sales.order.view' },
    { label: 'sales.nav.items', path: 'items', icon: IconPackage, permission: 'sales.item.view', group: 'sales.nav.group_config' },
    {
      label: 'sales.nav.approval_rules',
      path: 'approval-rules',
      icon: IconGitBranch,
      permission: ['sales.approval.manage', 'core.approval.manage'],
      group: 'sales.nav.group_config',
    },
  ],
  collapsibleGroups: ['sales.nav.group_config'],
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
      invalidate: (id) => [salesKeys.customers.detail(id), salesKeys.customers.lists()],
    },
  },
}
