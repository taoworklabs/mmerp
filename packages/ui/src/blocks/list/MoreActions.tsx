import type * as React from 'react'
import { DotsIcon } from '../../internal/icons.tsx'
import { ActionIcon } from '../../primitives/input/ActionIcon.tsx'
import { Menu } from '../../primitives/overlay/Menu.tsx'
import { Tooltip } from '../../primitives/overlay/Tooltip.tsx'

export interface MoreActionsProps {
  label: string
  // A table row's actions are quiet; a page's own sit beside its other buttons and need an edge.
  place?: 'row' | 'page'
  children: React.ReactNode
}

// MoreActions holds the actions that do not earn a button of their own: at the end of a table row,
// or after a page's buttons once there are more than a few.
export function MoreActions({ label, place = 'row', children }: MoreActionsProps) {
  if (!children) return null
  return (
    <Menu.Root>
      <Tooltip label={label}>
        <Menu.Trigger
          render={
            <ActionIcon variant={place === 'page' ? 'default' : 'subtle'} color="gray" aria-label={label}>
              <DotsIcon />
            </ActionIcon>
          }
        />
      </Tooltip>
      <Menu.Popup align="end">{children}</Menu.Popup>
    </Menu.Root>
  )
}
