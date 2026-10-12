import { Button, Drawer, Group, Stack } from '@mantine/core'
import { IconArrowRight } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useSearchParams } from 'react-router'
import { readingWidth } from './page/Page'
import { icon } from './theme'

// SidePanel previews one record of a list without leaving it: read-only, with a link to the full
// page where it is edited. The list keeps the open record in its URL (?peek=), so Back closes it.
export function SidePanel({ title, href, opened, onClose, children }: { title: string; href: string; opened: boolean; onClose: () => void; children: ReactNode }) {
  const { t } = useTranslation()
  return (
    // Not modal: the list stays usable beside it, so clicking another row switches the panel.
    <Drawer
      opened={opened}
      onClose={onClose}
      position="right"
      size={readingWidth}
      title={title}
      withOverlay={false}
      lockScroll={false}
      trapFocus={false}
      closeOnClickOutside={false}
      closeButtonProps={{ 'aria-label': t('shared.modal.close') }}
    >
      <Stack gap="md">
        <Group>
          <Button component={Link} to={href} variant="default" rightSection={<IconArrowRight {...icon.button} />}>
            {t('shared.panel.open')}
          </Button>
        </Group>
        {children}
      </Stack>
    </Drawer>
  )
}

// usePeek is the list's open side-panel record, kept on the URL as ?peek=<id>; opening adds a
// history entry, so the browser's Back closes the panel.
export function usePeek() {
  const [search, setSearch] = useSearchParams()
  const id = search.get('peek')
  const open = (key: string | number) =>
    setSearch((prev) => {
      const next = new URLSearchParams(prev)
      next.set('peek', String(key))
      return next
    })
  const close = () =>
    setSearch((prev) => {
      const next = new URLSearchParams(prev)
      next.delete('peek')
      return next
    })
  return { id, open, close }
}
