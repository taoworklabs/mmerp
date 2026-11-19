import { IconShoppingBag } from '@tabler/icons-react'
import type { AreaManifest } from '@/shared/area'

export const purchase: AreaManifest = {
  product: 'purchase',
  label: 'purchase.nav.group',
  basePath: '/purchase',
  nav: [],
  homeIcon: IconShoppingBag,
  i18n: {
    meta: { vi: () => import('./i18n/vi/meta.json'), en: () => import('./i18n/en/meta.json') },
    main: { vi: () => import('./i18n/vi/main.json'), en: () => import('./i18n/en/main.json') },
  },
  recordTypes: {},
}
