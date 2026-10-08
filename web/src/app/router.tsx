import { lazy, type ComponentType } from 'react'
import { createBrowserRouter, Navigate, type RouteObject } from 'react-router'
import i18n from 'i18next'
import { HomePage, InboxPage, JobsPage, nextFromLocation, NotificationsPage } from '@/core'
import type { AreaManifest } from '@/shared/area'
import { addTranslations, type Locale } from '@/shared/i18n'
import { NotFoundPage } from '@/shared/ui/states'
import { AppLayout } from './AppLayout'
import { BootError } from './BootError'

// An area's first visit loads its screens and its main translations before rendering;
// the layout's Suspense shows the skeleton inside the shell meanwhile.
function areaRoute(area: AreaManifest): RouteObject | null {
  const routes = area.routes
  if (!routes) return null
  const Area: ComponentType = lazy(async () => {
    const locale = i18n.language as Locale
    const [mod] = await Promise.all([routes(), area.i18n.main && addTranslations(locale, [area.i18n.main[locale]])])
    return mod
  })
  return { path: `${area.basePath}/*`, Component: Area }
}

const DevUi = lazy(() => import('./DevUi'))

export function createAppRouter(areas: AreaManifest[]) {
  const children: RouteObject[] = [
    { index: true, element: <HomePage areas={areas} /> },
    { path: 'inbox', Component: InboxPage },
    { path: 'jobs', Component: JobsPage },
    { path: 'notifications', Component: NotificationsPage },
    // Signed in already: continue where the login would have gone.
    { path: 'login', element: <Navigate to={nextFromLocation() ?? '/'} replace /> },
    ...areas.map(areaRoute).filter((r) => r !== null),
    { path: '*', Component: NotFoundPage },
  ]
  if (import.meta.env.DEV) children.push({ path: 'dev/ui', Component: DevUi })
  return createBrowserRouter([{ path: '/', element: <AppLayout areas={areas} />, errorElement: <BootError />, children }])
}
