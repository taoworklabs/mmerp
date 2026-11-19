import { IconAdjustmentsHorizontal, IconArrowsExchange, IconBuildingWarehouse, IconListDetails, IconPackage, IconPackages } from '@tabler/icons-react'
import type { AreaManifest } from '@/shared/area'
import { adjustmentDocType, stockKeys, transferDocType } from './keys'

// Stock is shared, not a product: the area opens to any stock or item permission, whatever
// products are enabled.
export const stock: AreaManifest = {
  product: 'stock',
  shared: true,
  label: 'stock.nav.group',
  basePath: '/stock',
  routes: () => import('./routes'),
  nav: [
    { label: 'stock.nav.on_hand', path: 'on-hand', icon: IconPackages, permission: 'stock.view' },
    { label: 'stock.nav.stock_card', path: 'card', icon: IconListDetails, permission: 'stock.view' },
    { label: 'stock.nav.transfers', path: 'transfers', icon: IconArrowsExchange, permission: 'stock.view' },
    { label: 'stock.nav.adjustments', path: 'adjustments', icon: IconAdjustmentsHorizontal, permission: 'stock.view' },
    { label: 'stock.nav.items', path: 'items', icon: IconPackage, permission: 'item.view', group: 'stock.nav.group_config' },
    { label: 'stock.nav.warehouses', path: 'warehouses', icon: IconBuildingWarehouse, permission: 'stock.view', group: 'stock.nav.group_config' },
  ],
  collapsibleGroups: ['stock.nav.group_config'],
  homeIcon: IconBuildingWarehouse,
  i18n: {
    meta: { vi: () => import('./i18n/vi/meta.json'), en: () => import('./i18n/en/meta.json') },
    main: { vi: () => import('./i18n/vi/main.json'), en: () => import('./i18n/en/main.json') },
  },
  recordTypes: {
    [adjustmentDocType]: {
      path: (id) => `adjustments/${id}`,
      preview: () => import('./components/AdjustmentPreview'),
      // Posting or cancelling one moves stock.
      invalidate: () => [stockKeys.adjustments.all(), stockKeys.onHand.all(), stockKeys.cards()],
    },
    [transferDocType]: {
      path: (id) => `transfers/${id}`,
      preview: () => import('./components/TransferPreview'),
      invalidate: () => [stockKeys.transfers.all(), stockKeys.onHand.all(), stockKeys.cards()],
    },
  },
}
