import { Box, Button, Flex, Group, NavLink, Stack, Text, VisuallyHidden } from '@mantine/core'
import { useMediaQuery } from '@mantine/hooks'
import { IconArrowLeft } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { icon } from '../theme'
import classes from './InboxPage.module.css'
import { Page, type PageHeaderProps } from './Page'

type Props = Omit<PageHeaderProps, 'actions' | 'leading'> & {
  list: ReactNode
  // The selected item's preview and approval panel; null when nothing is selected.
  detail: ReactNode | null
  onBack: () => void
}

// InboxPage: waiting items on the left, the selected one on the right, split by a rule
// rather than boxed. Below 1024px it is
// list then detail, full screen each, with a back button; the selection lives on the URL,
// so the browser's Back returns to the list too.
export function InboxPage({ list, detail, onBack, ...header }: Props) {
  const { t } = useTranslation()
  const wide = useMediaQuery('(min-width: 64em)', true)
  if (!wide)
    return (
      <Page {...header}>
        {detail ? (
          <>
            <Box>
              <Button variant="subtle" leftSection={<IconArrowLeft {...icon.button} />} onClick={onBack}>
                {t('shared.inbox.back')}
              </Button>
            </Box>
            {detail}
          </>
        ) : (
          list
        )}
      </Page>
    )
  return (
    <Page {...header}>
      <Flex align="stretch">
        <Box className={classes.list}>{list}</Box>
        <Box flex={1} miw={0} pl="xl">
          {detail ?? (
            <Text c="dimmed" ta="center" py="xl">
              {t('shared.inbox.pick')}
            </Text>
          )}
        </Box>
      </Flex>
    </Page>
  )
}

// InboxList holds the inbox's items as flat rows, without bullets; framed, as a page's feed
// (notifications), they sit in one frame ruled apart, like a table inside a page.
export function InboxList({ label, framed = false, children }: { label: string; framed?: boolean; children: ReactNode }) {
  return (
    <Stack gap={framed ? 0 : 'xxxs'} component="ul" className={framed ? `${classes.items} ${classes.framed}` : classes.items} aria-label={label}>
      {children}
    </Stack>
  )
}

// InboxGroup heads the rows that follow it in a framed list, e.g. a day.
export function InboxGroup({ label }: { label: string }) {
  return (
    <Text component="li" size="xs" fw={600} c="dimmed" className={classes.group}>
      {label}
    </Text>
  )
}

// InboxRow is one item of a feed (waiting approvals, notifications); the selected one is
// tinted, an unread one is bold, marked with a dot and announced as such. icon leads the row
// (what kind of item it is), aside ends it (when it happened).
export function InboxRow({
  active,
  unread = false,
  onClick,
  label,
  description,
  icon: lead,
  aside,
}: {
  active: boolean
  unread?: boolean
  onClick: () => void
  label: ReactNode
  description: string
  icon?: ReactNode
  aside?: ReactNode
}) {
  const { t } = useTranslation()
  return (
    <li>
      <NavLink
        component="button"
        active={active}
        onClick={onClick}
        classNames={{ label: unread ? classes.unread : undefined }}
        leftSection={lead}
        rightSection={
          (aside || unread) && (
            <Group gap="sm" wrap="nowrap">
              {aside}
              <span className={classes.dot} data-unread={unread || undefined} aria-hidden />
            </Group>
          )
        }
        label={
          <>
            {unread && <VisuallyHidden>{t('shared.inbox.unread')}: </VisuallyHidden>}
            {label}
          </>
        }
        description={description}
      />
    </li>
  )
}
