import { Tooltip as BaseTooltip } from '@base-ui/react/tooltip'
import type * as React from 'react'
import styles from './Tooltip.module.css'

export type TooltipSide = 'top' | 'bottom' | 'left' | 'right'

export interface TooltipProps {
  label: React.ReactNode
  side?: TooltipSide
  disabled?: boolean
  children: React.ReactElement
}

export function Tooltip({ label, side = 'top', disabled, children }: TooltipProps) {
  return (
    <BaseTooltip.Root disabled={disabled}>
      <BaseTooltip.Trigger render={children} />
      <BaseTooltip.Portal>
        <BaseTooltip.Positioner side={side} sideOffset={6} className={styles.positioner}>
          {/* Base UI leaves tooltips visual-only; role="tooltip" is ours to add. */}
          <BaseTooltip.Popup role="tooltip" className={styles.popup}>
            {label}
          </BaseTooltip.Popup>
        </BaseTooltip.Positioner>
      </BaseTooltip.Portal>
    </BaseTooltip.Root>
  )
}
