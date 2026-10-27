import { ButtonBase } from '../../internal/ButtonBase.tsx'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { RenderProp } from '../../internal/render.ts'
import type { StyleVars } from '../../internal/style.ts'
import { Loader } from '../feedback/Loader.tsx'
import styles from './Button.module.css'

export type ButtonVariant = 'filled' | 'light' | 'subtle' | 'default' | 'outline'
export type ButtonColor = 'primary' | 'gray' | 'danger' | 'success' | 'warning'

// Shared with ActionIcon: both pick their palette from the same variant x colour grid.
export function buttonColorVars(variant: ButtonVariant, color: ButtonColor): StyleVars {
  switch (variant) {
    case 'filled':
      return {
        '--mmerp-btn-bg': `var(--mmerp-color-${color}-6)`,
        '--mmerp-btn-bg-hover': `var(--mmerp-color-${color}-7)`,
        '--mmerp-btn-color': 'var(--mmerp-color-body)',
        '--mmerp-btn-border': 'transparent',
      }
    case 'light':
      return {
        '--mmerp-btn-bg': `var(--mmerp-color-${color}-light)`,
        '--mmerp-btn-bg-hover': `var(--mmerp-color-${color}-light-hover)`,
        '--mmerp-btn-color': `var(--mmerp-color-${color}-light-color)`,
        '--mmerp-btn-border': 'transparent',
      }
    case 'subtle':
      return {
        '--mmerp-btn-bg': 'transparent',
        '--mmerp-btn-bg-hover': `var(--mmerp-color-${color}-light)`,
        '--mmerp-btn-color': `var(--mmerp-color-${color}-light-color)`,
        '--mmerp-btn-border': 'transparent',
      }
    case 'outline':
      return {
        '--mmerp-btn-bg': 'transparent',
        '--mmerp-btn-bg-hover': `var(--mmerp-color-${color}-light)`,
        '--mmerp-btn-color': `var(--mmerp-color-${color}-6)`,
        '--mmerp-btn-border': `var(--mmerp-color-${color}-6)`,
      }
    case 'default':
      return {
        '--mmerp-btn-bg': 'var(--mmerp-color-elevation-surface)',
        '--mmerp-btn-bg-hover': 'var(--mmerp-color-elevation-surface-sunken)',
        '--mmerp-btn-color': 'var(--mmerp-color-text)',
        '--mmerp-btn-border': 'var(--mmerp-color-border)',
      }
  }
}

export interface ButtonProps extends Omit<React.ComponentPropsWithRef<'button'>, 'color'> {
  variant?: ButtonVariant
  color?: ButtonColor
  size?: 'xs' | 'sm'
  leftSection?: React.ReactNode
  rightSection?: React.ReactNode
  loading?: boolean
  fullWidth?: boolean
  render?: RenderProp
}

function ButtonRoot({
  variant = 'filled',
  color = 'primary',
  size = 'sm',
  leftSection,
  rightSection,
  loading,
  fullWidth,
  disabled,
  className,
  style,
  render,
  children,
  ...props
}: ButtonProps) {
  const vars: StyleVars = { ...style, ...buttonColorVars(variant, color) }
  return (
    <ButtonBase
      render={render}
      disabled={disabled || loading}
      aria-busy={loading ? true : undefined}
      data-mmerp-control=""
      data-size={size}
      data-full-width={fullWidth ? '' : undefined}
      className={cx(styles.root, className)}
      style={vars}
      {...props}
    >
      {loading ? (
        <span aria-hidden="true" className={styles.loader}>
          <Loader size={size} label="" />
        </span>
      ) : (
        leftSection
      )}
      {children}
      {!loading && rightSection}
    </ButtonBase>
  )
}

// Joins its buttons edge to edge, like a split button or a chip's main button and its clear button.
function Group({ className, ...props }: React.ComponentPropsWithRef<'div'>) {
  return <div role="group" className={cx(styles.group, className)} {...props} />
}

export const Button = Object.assign(ButtonRoot, { Group })
