import '@fontsource-variable/inter'
import '@mantine/core/styles.css'
import '@mantine/dates/styles.css'
import '@mantine/notifications/styles.css'
import '@/shared/ui/global.css'

import { MantineProvider } from '@mantine/core'
import { DatesProvider } from '@mantine/dates'
import 'dayjs/locale/vi'
import 'dayjs/locale/en-gb'
import { QueryClientProvider } from '@tanstack/react-query'
import { StrictMode, type ReactNode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router'
import i18n from 'i18next'
import { LoginPage, core } from '@/core'
import { api, type Me } from '@/shared/api/core'
import { RecordTypesProvider } from '@/shared/area'
import { MeProvider } from '@/shared/auth/me'
import { queryClient, startSession } from '@/shared/auth/session'
import { addTranslations, initI18n, rememberLocale, setErrorPrefixes, switchLocale, type Locale } from '@/shared/i18n'
import { cssVariablesResolver, theme } from '@/shared/ui/theme'
import { Toasts } from '@/shared/ui/Toasts'
import { BootError } from './BootError'
import { areas } from './areas'
import { createAppRouter } from './router'

// fetchMe returns null for an anonymous visitor; a 401 here never ends a session.
async function fetchMe(): Promise<Me | null> {
  const { data, response } = await api.GET('/me')
  if (response.status === 401) return null
  if (!data) throw new Error(`GET /me: ${response.status}`)
  return data
}

export async function bootstrap(root: HTMLElement) {
  const provisional = await initI18n()
  // Every area may own error wording; its meta bundle is loaded before anything can fail.
  setErrorPrefixes(areas.map((a) => a.product))
  await addTranslations(provisional, [core.i18n.meta[provisional]])

  let page: ReactNode
  try {
    const me = await fetchMe()
    if (!me) {
      page = <LoginPage />
    } else {
      startSession(me.authz_version)
      const locale = me.locale as Locale
      if (locale !== provisional) {
        await addTranslations(locale, [core.i18n.meta[locale]])
        await switchLocale(locale)
      }
      rememberLocale(locale)
      // A disabled product with data stays readable, so its area shows.
      const shown = areas.filter((a) => a.product === 'core' || me.products.includes(a.product) || me.products_with_data.includes(a.product))
      await addTranslations(locale, shown.filter((a) => a !== core).map((a) => a.i18n.meta[locale]))
      page = (
        <MeProvider me={me}>
          <RecordTypesProvider areas={shown}>
            <RouterProvider router={createAppRouter(shown)} />
          </RecordTypesProvider>
        </MeProvider>
      )
    }
  } catch {
    page = <BootError />
  }

  createRoot(root).render(
    <StrictMode>
      <MantineProvider theme={theme} cssVariablesResolver={cssVariablesResolver}>
        <DatesProvider settings={{ locale: i18n.language === 'en' ? 'en-gb' : 'vi', firstDayOfWeek: 1 }}>
          <Toasts />
          <QueryClientProvider client={queryClient}>{page}</QueryClientProvider>
        </DatesProvider>
      </MantineProvider>
    </StrictMode>,
  )
}
