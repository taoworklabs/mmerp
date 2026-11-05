import { ButtonBase } from '../../internal/ButtonBase.tsx'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { RenderProp } from '../../internal/render.ts'
import type { StyleVars } from '../../internal/style.ts'
import { Loader } from '../feedback/Loader.tsx'
import { buttonToneVars, type ButtonTone, type ButtonVariant } from './Button.tsx'
import styles from './ActionIcon.module.css'

export interface ActionIconProps extends Omit<React.ComponentPropsWithRef<'button'>, 'color'> {
  variant?: ButtonVariant
  tone?: ButtonTone
  size?: 'xs' | 'sm'
  loading?: boolean
  render?: RenderProp
  'aria-label': string
}

export function ActionIcon({
  variant = 'filled',
  tone = 'primary',
  size = 'sm',
  loading,
  disabled,
  className,
  style,
  render,
  children,
  ...props
}: ActionIconProps) {
  const vars: StyleVars = { ...style, ...buttonToneVars(variant, tone) }
  return (
    <ButtonBase
      render={render}
      disabled={disabled || loading}
      // Loading must not blur a focused button (native disabled does); clicks are still blocked.
      focusableWhenDisabled={!disabled}
      aria-busy={loading ? true : undefined}
      data-mmerp-control=""
      data-size={size}
      className={cx(styles.root, className)}
      style={vars}
      {...props}
    >
      {loading ? (
        <span aria-hidden="true">
          <Loader size={size} label="" />
        </span>
      ) : (
        children
      )}
    </ButtonBase>
  )
}
