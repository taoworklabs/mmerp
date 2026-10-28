import { can, useMe } from '@/shared/auth/me'
import type { AreaManifest, NavItem } from './index'

// masterPerms: a nav item's permissions in a shared master-data namespace; core ones (approval
// rules) open no area on their own.
const masterPerms = (perm: string | string[] | undefined) => [perm ?? []].flat().filter((p) => !p.startsWith('core.'))

// useAreasEntitled says, per area, whether the user may enter it; undefined while the area's own
// hook loads. Without that hook an area needs a permission in its product or one of its shared
// master-data nav items, so self-service alone opens none. The home tiles, the area's routes and
// record links all follow it.
export function useAreasEntitled(areas: AreaManifest[]): (boolean | undefined)[] {
  const me = useMe()
  // areas is a fixed array, so the hook order never changes.
  return areas.map((area) => {
    const hook = area.useEntitled?.()
    if (area.useEntitled) return hook
    return (me.permissions[area.product]?.length ?? 0) > 0 || area.nav.some((item) => masterPerms(item.permission).some((p) => can(me, p)))
  })
}

// useNavHidden lists the nav items whose own hook hides them, across every area so the hook
// order never changes.
export function useNavHidden(areas: AreaManifest[]): Set<NavItem> {
  const items = areas.flatMap((a) => a.nav)
  const shown = items.map((item) => item.useShown?.() ?? true)
  return new Set(items.filter((_, i) => !shown[i]))
}
