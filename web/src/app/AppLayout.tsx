import { ActionIcon, AppShell, Box, Burger, Group, Indicator, Menu, NavLink, ScrollArea, Stack, Text, ThemeIcon, Tooltip, UnstyledButton } from '@mantine/core'
import { useDisclosure, useLocalStorage, useMediaQuery } from '@mantine/hooks'
import { IconBell, IconBuildingSkyscraper, IconCheck, IconHourglass, IconInbox, IconChevronDown, IconChevronRight, IconLayoutSidebarLeftCollapse, IconLayoutSidebarLeftExpand, IconLogout } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { Suspense, useEffect, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, Outlet, useLocation } from 'react-router'
import { notificationKeys } from '@/core'
import { api, unwrap } from '@/shared/api/core'
import type { AreaManifest, NavItem } from '@/shared/area'
import { can, useMe } from '@/shared/auth/me'
import { endSession } from '@/shared/auth/session'
import { documentKeys } from '@/shared/document'
import { errorText, locales, rememberLocale, type Locale } from '@/shared/i18n'
import { PersonAvatar } from '@/shared/ui/avatar'
import { pageFrame } from '@/shared/ui/page'
import { ErrorState, PageSkeleton, ReadOnlyBanner } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'

const join = (base: string, path: string) => `${base.replace(/\/$/, '')}/${path}`
const inArea = (a: AreaManifest, pathname: string) => pathname === a.basePath || pathname.startsWith(`${a.basePath}/`)

export function AppLayout({ areas }: { areas: AreaManifest[] }) {
  const { t, i18n } = useTranslation()
  const me = useMe()
  const { pathname } = useLocation()
  const area = areas.find((a) => inArea(a, pathname))
  // Home and the shared pages (notifications, inbox, jobs) have no sidebar.
  const nav = area ? area.nav.filter((item) => can(me, item.permission)) : []
  const [drawer, { toggle: toggleDrawer, close: closeDrawer }] = useDisclosure()
  // Capture phase: the burger's tooltip consumes Escape before it bubbles.
  useEffect(() => {
    if (!drawer) return
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && closeDrawer()
    document.addEventListener('keydown', onKey, true)
    return () => document.removeEventListener('keydown', onKey, true)
  }, [drawer, closeDrawer])
  const wide = useMediaQuery('(min-width: 80em)', true)
  // Remembered per user; until the user chooses, it collapses below 1280px.
  const [pref, setPref] = useLocalStorage<boolean | null>({ key: `mmerp.nav.collapsed.${me.id}`, defaultValue: null })
  const collapsed = pref ?? !wide

  const [error, setError] = useState<string | null>(null)
  const inbox = useQuery({ queryKey: documentKeys.inbox(), queryFn: () => unwrap(api.GET('/approvals/inbox')) })
  const waiting = inbox.data?.items.length ?? 0
  // Polled: notifications arrive from other users' actions, with nothing on this page to refresh them.
  const unread = useQuery({
    queryKey: notificationKeys.unread(),
    queryFn: () => unwrap(api.GET('/notifications/unread-count')),
    refetchInterval: 60_000,
  }).data?.count ?? 0
  const narrow = collapsed && !drawer

  // A 401 ends the session in the API client; any other failure stays on screen.
  async function run(action: () => Promise<unknown>, then: () => void) {
    setError(null)
    try {
      await action()
    } catch (err) {
      setError(errorText(t, err))
      return
    }
    then()
  }

  const setLocale = (locale: Locale) =>
    run(
      () => unwrap(api.PATCH('/me', { body: { locale } })),
      () => {
        rememberLocale(locale)
        location.reload()
      },
    )

  // Only a logout the server confirmed ends the session; otherwise the cookie would still work.
  const logout = () => run(() => unwrap(api.POST('/auth/logout')), endSession)

  return (
    <AppShell
      header={{ height: 48 }}
      navbar={nav.length > 0 ? { width: collapsed ? 56 : 240, breakpoint: 'md', collapsed: { mobile: !drawer } } : undefined}
      padding={0}
    >
      <AppShell.Header px="md">
        <Group h="100%" justify="space-between" wrap="nowrap">
          <Group gap="xs" wrap="nowrap" miw={0}>
            {nav.length > 0 && (
              <Tooltip label={t('core.shell.menu')}>
                <Burger opened={drawer} onClick={toggleDrawer} hiddenFrom="md" size="sm" aria-label={t('core.shell.menu')} aria-expanded={drawer} />
              </Tooltip>
            )}
            {/* The logo goes home, where the products are. */}
            <Tooltip label={t('core.nav.home')}>
              <UnstyledButton component={Link} to="/" aria-label={t('core.nav.home')} onClick={closeDrawer}>
                <ThemeIcon size={28} radius="sm" aria-hidden>
                  <IconBuildingSkyscraper {...icon.text} />
                </ThemeIcon>
              </UnstyledButton>
            </Tooltip>
            <Text fw={600} truncate>{t(area?.label ?? 'shared.app.title')}</Text>
          </Group>
          <Group gap="xs" wrap="nowrap" flex="none">
            {/* Every user has notifications, an inbox and background jobs, wherever they are. */}
            <Tooltip label={t('core.nav.notifications')}>
              <Indicator label={unread} size={16} disabled={unread === 0} offset={4}>
                <ActionIcon
                  component={Link}
                  to="/notifications"
                  variant={pathname === '/notifications' ? 'light' : 'subtle'}
                  color={pathname === '/notifications' ? undefined : 'gray'}
                  size="lg"
                  aria-label={unread > 0 ? t('core.nav.notifications_count', { count: unread }) : t('core.nav.notifications')}
                >
                  <IconBell {...icon.button} />
                </ActionIcon>
              </Indicator>
            </Tooltip>
            <Tooltip label={t('core.nav.inbox')}>
              <Indicator label={waiting} size={16} disabled={waiting === 0} offset={4}>
                <ActionIcon
                  component={Link}
                  to="/inbox"
                  variant={pathname === '/inbox' ? 'light' : 'subtle'}
                  color={pathname === '/inbox' ? undefined : 'gray'}
                  size="lg"
                  aria-label={waiting > 0 ? t('core.nav.inbox_count', { count: waiting }) : t('core.nav.inbox')}
                >
                  <IconInbox {...icon.button} />
                </ActionIcon>
              </Indicator>
            </Tooltip>
            <Tooltip label={t('core.nav.jobs')}>
              <ActionIcon
                component={Link}
                to="/jobs"
                variant={pathname === '/jobs' ? 'light' : 'subtle'}
                color={pathname === '/jobs' ? undefined : 'gray'}
                size="lg"
                aria-label={t('core.nav.jobs')}
              >
                <IconHourglass {...icon.button} />
              </ActionIcon>
            </Tooltip>
            <Menu position="bottom-end">
              <Menu.Target>
                <Tooltip label={t('core.shell.account')}>
                <UnstyledButton aria-label={`${t('core.shell.account')}: ${me.name}`} flex="none" h={36} px="xs">
                  <Group gap="xs" wrap="nowrap">
                    <PersonAvatar name={me.name} />
                    <Text truncate maw={200} fw={500} visibleFrom="sm">{me.name}</Text>
                    <IconChevronDown {...icon.text} />
                  </Group>
                </UnstyledButton>
                </Tooltip>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>{t('core.shell.language')}</Menu.Label>
                {locales.map((l) => (
                  <Menu.Item
                    key={l}
                    lang={l}
                    aria-current={l === i18n.language ? 'true' : undefined}
                    onClick={() => l !== i18n.language && void setLocale(l)}
                    rightSection={l === i18n.language ? <IconCheck {...icon.text} /> : null}
                  >
                    {t(`shared.locale.${l}`)}
                  </Menu.Item>
                ))}
                <Menu.Divider />
                <Menu.Item leftSection={<IconLogout {...icon.text} />} onClick={() => void logout()}>
                  {t('core.shell.logout')}
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>
          </Group>
        </Group>
      </AppShell.Header>

      {nav.length > 0 && area && (
        <AppShell.Navbar>
          <AppShell.Section grow component={ScrollArea} p="xs">
            {/* Only the open area's menu; other areas are reached from home. Ungrouped items
                first, then each group in manifest order; a group emptied by permissions is gone. */}
            <Stack gap="xxs">
              {[...new Set(nav.map((item) => item.group))].map((group) => {
                const links = nav.filter((item) => item.group === group).map((item) => <AreaLink key={item.path} area={area} item={item} narrow={narrow} onClick={closeDrawer} />)
                return group && !narrow ? (
                  <NavGroup key={group} userId={me.id} group={group} collapsible={area.collapsibleGroups?.includes(group) ?? false}>
                    {links}
                  </NavGroup>
                ) : (
                  links
                )
              })}
            </Stack>
          </AppShell.Section>
          <AppShell.Section p="xs" visibleFrom="md">
            <Tooltip label={t(collapsed ? 'core.shell.expand' : 'core.shell.collapse')} position="right">
              <ActionIcon
                variant="subtle"
                color="gray"
                size="lg"
                aria-label={t(collapsed ? 'core.shell.expand' : 'core.shell.collapse')}
                onClick={() => setPref(!collapsed)}
              >
                {collapsed ? <IconLayoutSidebarLeftExpand {...icon.button} /> : <IconLayoutSidebarLeftCollapse {...icon.button} />}
              </ActionIcon>
            </Tooltip>
          </AppShell.Section>
        </AppShell.Navbar>
      )}

      <AppShell.Main>
        {error && (
          <Box p="lg" pb={0}>
            <ErrorState message={error} />
          </Box>
        )}
        {area && area.product !== 'core' && !me.products.includes(area.product) && (
          <Box px={pageFrame.px} pt="lg">
            <ReadOnlyBanner />
          </Box>
        )}
        {/* Keyed by area, so entering one shows the skeleton rather than the previous page. */}
        <Suspense key={pathname.split('/')[1]} fallback={<PageSkeleton />}>
          <Outlet />
        </Suspense>
      </AppShell.Main>
    </AppShell>
  )
}

function AreaLink({ area, item, narrow, onClick }: { area: AreaManifest; item: NavItem; narrow: boolean; onClick: () => void }) {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  const to = join(area.basePath, item.path)
  const label = t(item.label)
  const link = (
    <NavLink
      component={Link}
      to={to}
      label={narrow ? undefined : label}
      aria-label={label}
      leftSection={<item.icon {...icon.button} />}
      active={pathname === to || pathname.startsWith(`${to}/`)}
      variant="light"
      onClick={onClick}
    />
  )
  return narrow ? (
    <Tooltip label={label} position="right">
      {link}
    </Tooltip>
  ) : (
    link
  )
}

// NavGroup heads a group of an area's items; a collapsible one folds, remembered per user.
function NavGroup({ userId, group, collapsible, children }: { userId: number; group: string; collapsible: boolean; children: ReactNode }) {
  const { t } = useTranslation()
  const label = t(group)
  // Keyed by the i18n key, so the state survives a language change.
  const [folded, setFolded] = useLocalStorage<boolean>({ key: `mmerp.nav.folded.${userId}.${group}`, defaultValue: false })
  const open = !collapsible || !folded
  return (
    <Stack gap="xxs" role="group" aria-label={label}>
      {collapsible ? (
        <UnstyledButton px="xs" onClick={() => setFolded(open)} aria-expanded={open}>
          <Group gap="xxs" justify="space-between" wrap="nowrap">
            <Text size="xs" c="dimmed">
              {label}
            </Text>
            {open ? <IconChevronDown {...icon.text} /> : <IconChevronRight {...icon.text} />}
          </Group>
        </UnstyledButton>
      ) : (
        <Text size="xs" c="dimmed" px="xs">
          {label}
        </Text>
      )}
      {open && children}
    </Stack>
  )
}
