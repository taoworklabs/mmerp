import { useTranslation } from 'react-i18next'
import { useAreasEntitled, type AreaManifest } from '@/shared/area'
import { can, useMe } from '@/shared/auth/me'
import { Page } from '@/shared/ui/page'
import { EmptyState } from '@/shared/ui/states'
import { LinkTile, TileGrid } from '@/shared/ui/Tile'

// HomePage: a tile per area the user may enter, opening its first nav item the user may see.
// The areas shown are already those whose product is enabled or has data.
export default function HomePage({ areas }: { areas: AreaManifest[] }) {
  const { t } = useTranslation()
  const me = useMe()
  const entitled = useAreasEntitled(areas)
  const tiles = areas.map((area, i) => {
    const first = area.nav.find((item) => can(me, item.permission))
    return { area, loading: entitled[i] === undefined, to: first && entitled[i] ? `${area.basePath}/${first.path}` : null }
  })
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
