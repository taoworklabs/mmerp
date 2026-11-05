import { IconCalculator } from '@tabler/icons-react'
import type { AreaManifest } from '@/shared/area'

export const accounting: AreaManifest = {
  product: 'accounting',
  label: 'accounting.nav.group',
  basePath: '/accounting',
  nav: [],
  homeIcon: IconCalculator,
  i18n: {
    meta: { vi: () => import('./i18n/vi/meta.json'), en: () => import('./i18n/en/meta.json') },
    main: { vi: () => import('./i18n/vi/main.json'), en: () => import('./i18n/en/main.json') },
  },
  recordTypes: {},
}
