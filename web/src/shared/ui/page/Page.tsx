import { Anchor, Breadcrumbs, Flex, Group, Stack, Text, Title } from '@mantine/core'
import { IconChevronRight } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { Link } from 'react-router'
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
  actions?: ReactNode
  // Templates with their own rule under the header (RecordPage's tabs) turn the header's off.
  divider?: boolean
}

// pageFrame is the padding around a page, shared with the page-level skeleton.
export const pageFrame = { px: { base: 'md', md: 'xl' }, py: 'lg' } as const

// readingWidth is the width of a side panel (approval, attachments, discussion) shown full width as a tab.
export const readingWidth = 480

// Page is the frame every page template builds on: header (breadcrumbs, title, at most one
// primary action), then content.
export function Page({ children, ...header }: PageHeaderProps & { children: ReactNode }) {
  return (
    <Stack gap="lg" {...pageFrame}>
      <PageHeader {...header} />
      {children}
    </Stack>
  )
}

function PageHeader({ title, breadcrumbs, description, leading, actions, divider = true }: PageHeaderProps) {
  return (
    <Stack gap="xs" className={divider ? classes.header : undefined}>
      {breadcrumbs && breadcrumbs.length > 0 && (
        <Breadcrumbs separator={<IconChevronRight {...icon.text} />} separatorMargin="xxs">
          {breadcrumbs.map((c) => (
            <Anchor key={c.to} component={Link} to={c.to} size="sm" c="dimmed">
              {c.label}
            </Anchor>
          ))}
        </Breadcrumbs>
      )}
      {/* Narrow screens stack the actions under the title instead of squeezing them. */}
      <Flex direction={{ base: 'column', sm: 'row' }} justify="space-between" align={{ base: 'stretch', sm: 'center' }} gap="sm">
        <Group gap="md" wrap="nowrap" miw={0}>
          {leading}
          <Stack gap={0} miw={0}>
            <Title order={1} lineClamp={1}>
              {title}
            </Title>
            {description && (
              <Text size="sm" c="dimmed" component="div">
                {description}
              </Text>
            )}
          </Stack>
        </Group>
        {actions}
      </Flex>
    </Stack>
  )
}
