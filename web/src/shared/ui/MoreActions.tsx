import { ActionIcon, Menu, Tooltip } from '@mantine/core'
import { IconDots } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { icon } from './theme'

// MoreActions holds the actions that do not earn a button, as Menu.Item children: at the end of a
// table row, or after a page's buttons once there are more than three. On a page it is outlined,
// on a row it is quiet; both are the one control height.
export function MoreActions({ children, place = 'row' }: { children: ReactNode; place?: 'page' | 'row' }) {
  const { t } = useTranslation()
  if (!children) return null
  const label = t('shared.table.actions')
  return (
    <Menu position="bottom-end">
      <Menu.Target>
        <Tooltip label={label}>
          <ActionIcon variant={place === 'page' ? 'default' : 'subtle'} color="gray" size="input-sm" aria-label={label}>
            <IconDots {...icon.button} />
          </ActionIcon>
        </Tooltip>
      </Menu.Target>
      <Menu.Dropdown>{children}</Menu.Dropdown>
    </Menu>
  )
}
