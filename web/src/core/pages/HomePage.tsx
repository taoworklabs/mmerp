import { useTranslation } from 'react-i18next'
import type { AreaManifest } from '@/shared/area'
import { can, useMe } from '@/shared/auth/me'
import { Page } from '@/shared/ui/page'
import { EmptyState } from '@/shared/ui/states'
import { LinkTile, TileGrid } from '@/shared/ui/Tile'

// HomePage: a tile per area the user may enter, opening its first nav item the user may see.
// The areas shown are already those whose product is enabled or has data; a tile also needs a
// permission in the product or a guarded nav item (shared master data), so self-service alone
// (filing leave) gives no tile.
export default function HomePage({ areas }: { areas: AreaManifest[] }) {
  const { t } = useTranslation()
  const me = useMe()
  const tiles = areas.flatMap((a) => {
    const first = a.nav.find((item) => can(me, item.permission))
    const entitled = (me.permissions[a.product]?.length ?? 0) > 0 || a.nav.some((item) => item.permission && can(me, item.permission))
    return first && entitled ? [{ area: a, to: `${a.basePath}/${first.path}` }] : []
  })
  return (
    <Page title={t('core.home.title', { name: me.name })} description={t('core.home.description')}>
      {tiles.length === 0 ? (
        <EmptyState title={t('core.home.empty')} description={t('core.home.empty_description')} />
      ) : (
        <TileGrid>
          {tiles.map(({ area, to }) => (
            <LinkTile key={area.product} icon={area.homeIcon} title={t(area.label)} to={to} />
          ))}
        </TileGrid>
      )}
    </Page>
  )
}
