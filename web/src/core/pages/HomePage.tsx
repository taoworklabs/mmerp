import { useTranslation } from 'react-i18next'
import type { AreaManifest } from '@/shared/area'
import { can, useMe } from '@/shared/auth/me'
import { Page } from '@/shared/ui/page'
import { EmptyState } from '@/shared/ui/states'
import { LinkTile, TileGrid } from '@/shared/ui/Tile'

// masterPerms: a nav item's permissions in a shared master-data namespace; core ones (approval
// rules) open no tile on their own.
const masterPerms = (perm: string | string[] | undefined) => [perm ?? []].flat().filter((p) => !p.startsWith('core.'))

// useTile: the area's tile target, or null. Without the area's own hook a tile needs a permission
// in the product or one of its shared master-data nav items, so self-service alone gives no tile.
function useTile(area: AreaManifest) {
  const me = useMe()
  const hook = area.useEntitled?.() // stable per area (fixed array), so the hook order never changes
  const first = area.nav.find((item) => can(me, item.permission))
  const entitled = area.useEntitled
    ? hook === true
    : (me.permissions[area.product]?.length ?? 0) > 0 || area.nav.some((item) => masterPerms(item.permission).some((p) => can(me, p)))
  return { loading: hook === undefined && !!area.useEntitled, to: first && entitled ? `${area.basePath}/${first.path}` : null }
}

// HomePage: a tile per area the user may enter, opening its first nav item the user may see.
// The areas shown are already those whose product is enabled or has data.
export default function HomePage({ areas }: { areas: AreaManifest[] }) {
  const { t } = useTranslation()
  const me = useMe()
  const tiles = areas.map((area) => ({ area, ...useTile(area) }))
  const shown = tiles.filter((x) => x.to)
  return (
    <Page title={t('core.home.title', { name: me.name })} description={t('core.home.description')}>
      {shown.length === 0 && !tiles.some((x) => x.loading) ? (
        <EmptyState title={t('core.home.empty')} description={t('core.home.empty_description')} />
      ) : (
        <TileGrid>
          {shown.map(({ area, to }) => (
            <LinkTile key={area.basePath} icon={area.homeIcon} title={t(area.label)} to={to!} />
          ))}
        </TileGrid>
      )}
    </Page>
  )
}
