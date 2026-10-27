import { useTranslation } from 'react-i18next'
import type { AreaManifest } from '@/shared/area'
import { can, useMe } from '@/shared/auth/me'
import { Page } from '@/shared/ui/page'
import { EmptyState } from '@/shared/ui/states'
import { LinkTile, TileGrid } from '@/shared/ui/Tile'

// masterPerms: a nav item's permissions in a shared master-data namespace; core ones (approval
// rules) open no tile on their own.
const masterPerms = (perm: string | string[] | undefined) => [perm ?? []].flat().filter((p) => !p.startsWith('core.'))

// AreaTile: one per area, so each may call its own useEntitled hook. Without the hook a tile needs
// a permission in the product or one of its shared master-data nav items, so self-service alone
// (filing leave) gives no tile.
function AreaTile({ area }: { area: AreaManifest }) {
  const { t } = useTranslation()
  const me = useMe()
  const hook = area.useEntitled?.() // stable per area, so the hook order never changes
  const first = area.nav.find((item) => can(me, item.permission))
  const entitled = area.useEntitled
    ? hook === true
    : (me.permissions[area.product]?.length ?? 0) > 0 || area.nav.some((item) => masterPerms(item.permission).some((p) => can(me, p)))
  if (!first || !entitled) return null
  return <LinkTile icon={area.homeIcon} title={t(area.label)} to={`${area.basePath}/${first.path}`} />
}

// HomePage: a tile per area the user may enter, opening its first nav item the user may see.
// The areas shown are already those whose product is enabled or has data.
export default function HomePage({ areas }: { areas: AreaManifest[] }) {
  const { t } = useTranslation()
  const me = useMe()
  // Areas with their own hook decide in their tile, so the empty state only covers the others.
  const any = areas.some((a) => a.useEntitled || a.nav.some((item) => can(me, item.permission)))
  return (
    <Page title={t('core.home.title', { name: me.name })} description={t('core.home.description')}>
      {!any ? (
        <EmptyState title={t('core.home.empty')} description={t('core.home.empty_description')} />
      ) : (
        <TileGrid>
          {areas.map((area) => (
            <AreaTile key={area.basePath} area={area} />
          ))}
        </TileGrid>
      )}
    </Page>
  )
}
