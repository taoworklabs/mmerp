import { Button as BaseButton } from '@base-ui/react/button'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { RenderProp } from '../../internal/render.ts'
import type { StyleVars } from '../../internal/style.ts'
import { Loader } from '../feedback/Loader.tsx'
import { buttonColorVars, type ButtonColor, type ButtonVariant } from './Button.tsx'
import styles from './ActionIcon.module.css'

export interface ActionIconProps extends Omit<React.ComponentPropsWithRef<'button'>, 'color'> {
  variant?: ButtonVariant
  color?: ButtonColor
  size?: 'xs' | 'sm'
  loading?: boolean
  render?: RenderProp
  'aria-label': string
}

export function ActionIcon({
  variant = 'filled',
  color = 'primary',
  size = 'sm',
  loading,
  disabled,
  className,
  style,
  render,
  children,
  ...props
}: ActionIconProps) {
  const vars: StyleVars = { ...style, ...buttonColorVars(variant, color) }
  return (
    <BaseButton
      render={render as React.ComponentProps<typeof BaseButton>['render']}
      disabled={disabled || loading}
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
    </BaseButton>
  )
}
