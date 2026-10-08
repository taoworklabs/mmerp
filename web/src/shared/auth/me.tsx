import { createContext, useContext, type ReactNode } from 'react'
import type { Me } from '@/shared/api/core'

const MeContext = createContext<Me | null>(null)

export function MeProvider({ me, children }: { me: Me; children: ReactNode }) {
  return <MeContext value={me}>{children}</MeContext>
}

// useCan only shows or hides menus and pages; buttons come from allowed_actions.
// A list means any one of them.
export function useCan(permission: string | string[] | undefined): boolean {
  return can(useMe(), permission)
}

export function can(me: Me, permission: string | string[] | undefined): boolean {
  if (!permission) return true
  if (Array.isArray(permission)) return permission.some((p) => can(me, p))
  return me.permissions[permission.split('.')[0] ?? '']?.includes(permission) ?? false
}

// useProductOn says whether a product is enabled; a disabled one that still shows is read-only.
// Only for write buttons that allowed_actions does not cover, such as catalog screens.
export function useProductOn(product: string): boolean {
  return useMe().products.includes(product)
}

// useMe is only valid inside the authenticated app.
export function useMe(): Me {
  const me = useContext(MeContext)
  if (!me) throw new Error('useMe outside MeProvider')
  return me
}
