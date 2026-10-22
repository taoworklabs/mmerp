import { Avatar as BaseAvatar } from '@base-ui/react/avatar'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { StyleVars } from '../../internal/style.ts'
import type { ColorName, DecorativeColor } from '../../tokens/tokens.ts'
import styles from './Avatar.module.css'

export interface AvatarProps extends Omit<React.ComponentPropsWithRef<'span'>, 'color'> {
  src?: string
  alt: string
  size?: number
  color?: ColorName | DecorativeColor
}

export function Avatar({ src, alt, size = 30, color = 'gray', className, style, children, ...props }: AvatarProps) {
  const vars: StyleVars = {
    ...style,
    '--mmerp-avatar-size': `${size}px`,
    '--mmerp-avatar-bg': `var(--mmerp-color-${color}-light)`,
    '--mmerp-avatar-color': `var(--mmerp-color-${color}-light-color)`,
  }
  return (
    <BaseAvatar.Root className={cx(styles.root, className)} style={vars} {...props}>
      {src && <BaseAvatar.Image src={src} alt={alt} className={styles.image} />}
      <BaseAvatar.Fallback className={styles.fallback}>{children}</BaseAvatar.Fallback>
    </BaseAvatar.Root>
  )
}
