import { ActionIcon, Anchor, Box, Breadcrumbs, Flex, Group, Stack, Text, Title, Tooltip } from '@mantine/core'
import { IconArrowLeft, IconChevronRight } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useLocation, useNavigate } from 'react-router'
import { icon } from '../theme'
import classes from './Page.module.css'

export type Crumb = { label: string; to: string }

export type PageHeaderProps = {
  title: string
  // Where this page sits: links to the pages above it, the page itself excluded.
  breadcrumbs?: Crumb[]
  // One line under the title: what the list holds, or the record's key facts.
  description?: ReactNode
  // Shown left of the title, e.g. a person's avatar.
  leading?: ReactNode
  // Right of the title: the record's status badge.
  status?: ReactNode
  actions?: ReactNode
  // Under the title, over the tabs: the record's key facts (KeyFacts).
  summary?: ReactNode
  // A Tabs.List (inside a Tabs around the page) that sits on the header's rule.
  tabs?: ReactNode
  // No rule under the header: the content draws its own frame.
  plain?: boolean
  // One record's page (DetailPage, DocumentPage): the breadcrumb ends with the page itself and a
  // back button leads the title.
  withBack?: boolean
  // The breadcrumb ends with the page itself, without the back button: a record page whose
  // breadcrumb already leads back.
  current?: boolean
}

// pageFrame is the padding around a page, shared with the page-level skeleton.
export const pageFrame = { px: { base: 'md', md: 'xl', lg: 'xxl' }, py: 'lg' } as const

// readingWidth is the width of a side panel (approval, attachments) shown full width as a tab.
export const readingWidth = 480

// Page is the frame every page template builds on: header (breadcrumbs, title, at most one
// primary action), then content. A record's or a form's page is centred at the width it stops at
// (centered: the content width, or the given one), its header too, so both share a left edge;
// lists and dashboards use the whole width.
export function Page({ children, className, centered, ...header }: PageHeaderProps & { children: ReactNode; className?: string; centered?: boolean | string }) {
  if (centered)
    return (
      <Box {...pageFrame} className={className}>
        <Stack gap="lg" w="100%" maw={centered === true ? 'var(--mmerp-content-width)' : centered} mx="auto">
          <PageHeader {...header} />
          {children}
        </Stack>
      </Box>
    )
  return (
    <Stack gap="lg" {...pageFrame} className={className}>
      <PageHeader {...header} />
      {children}
    </Stack>
  )
}

function PageHeader({ title, breadcrumbs, description, leading, status, actions, summary, tabs, plain, withBack, current }: PageHeaderProps) {
  const parent = breadcrumbs?.at(-1)
  return (
    <Stack gap="xs" className={classes.header} data-tabs={tabs ? true : undefined} data-plain={plain || undefined}>
      {breadcrumbs && breadcrumbs.length > 0 && (
        <Breadcrumbs separator={<IconChevronRight {...icon.text} />} separatorMargin="xxs">
          {breadcrumbs.map((c) => (
            <Anchor key={c.to} component={Link} to={c.to} size="sm" c="dimmed">
              {c.label}
            </Anchor>
          ))}
          {(withBack || current) && (
            <Text size="sm" fw={500} aria-current="page" truncate>
              {title}
            </Text>
          )}
        </Breadcrumbs>
      )}
      {/* Narrow screens stack the actions under the title instead of squeezing them. */}
      <Flex direction={{ base: 'column', sm: 'row' }} justify="space-between" align={{ base: 'stretch', sm: 'center' }} gap="sm">
        <Group gap="md" wrap="nowrap" miw={0}>
          {withBack && parent && <BackButton to={parent.to} />}
          {leading}
          <Stack gap={0} miw={0}>
            <Group gap="sm" wrap="nowrap" miw={0}>
              <Title order={1} lineClamp={1}>
                {title}
              </Title>
              {status}
            </Group>
            {description && (
              <Text size="sm" c="dimmed" component="div">
                {description}
              </Text>
            )}
          </Stack>
        </Group>
        {actions}
      </Flex>
      {summary}
      {tabs && <div className={classes.tabs}>{tabs}</div>}
    </Stack>
  )
}

// BackButton returns to where the user came from, the list with its filters and page; opened
// directly (a link from elsewhere, a new tab), it goes to the page above. A real link, so it also
// opens in a new tab.
function BackButton({ to }: { to: string }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const label = t('shared.page.back')
  return (
    <Tooltip label={label}>
      <ActionIcon
        component={Link}
        to={to}
        variant="default"
        radius={1000}
        size="input-sm"
        aria-label={label}
        onClick={(e) => {
          // The first entry of the session has the default key: there is nothing in the app to go back to.
          if (location.key === 'default' || e.metaKey || e.ctrlKey || e.shiftKey) return
          e.preventDefault()
          void navigate(-1)
        }}
      >
        <IconArrowLeft {...icon.button} />
      </ActionIcon>
    </Tooltip>
  )
}
