import { Menu } from '@mantine/core'
import type { Icon as TablerIcon } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { icon as iconSize } from './theme'

// MenuAction is one entry of a "…" menu (MoreActions): the icon at text size, red only for an
// action that loses something.
export function MenuAction({ icon: Icon, danger, onClick, children }: { icon: TablerIcon; danger?: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <Menu.Item color={danger ? 'danger' : undefined} leftSection={<Icon {...iconSize.text} />} onClick={onClick}>
      {children}
    </Menu.Item>
  )
}
