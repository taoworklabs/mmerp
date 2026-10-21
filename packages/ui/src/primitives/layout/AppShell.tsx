import type * as React from 'react'
import { useMediaQuery } from '../../hooks/useMediaQuery.ts'
import { cx } from '../../internal/cx.ts'
import type { StyleVars } from '../../internal/style.ts'
import styles from './AppShell.module.css'

export interface AppShellProps extends Omit<React.ComponentPropsWithRef<'div'>, 'children'> {
  header?: React.ReactNode
  headerHeight?: number
  navbar?: React.ReactNode
  navbarWidth?: number
  navbarOpened?: boolean
  navbarCollapsed?: boolean
  navbarCollapsedWidth?: number
  children: React.ReactNode
}

// Below this width the navbar leaves the grid and becomes an off-canvas overlay.
const mobileQuery = '(max-width: 63.99em)'

export function AppShell({
  header,
  headerHeight = 48,
  navbar,
  navbarWidth = 240,
  navbarOpened,
  navbarCollapsed,
  navbarCollapsedWidth = 56,
  className,
  style,
  children,
  ...props
}: AppShellProps) {
  const isMobile = useMediaQuery(mobileQuery)
  // The rail only applies at desktop widths; on mobile the navbar is always the full-width drawer.
  const effectiveNavbarWidth = !isMobile && navbarCollapsed ? navbarCollapsedWidth : navbarWidth
  const vars: StyleVars = {
    ...style,
    '--mmerp-appshell-header-height': `${headerHeight}px`,
    '--mmerp-appshell-navbar-width': `${effectiveNavbarWidth}px`,
  }
  return (
    <div className={cx(styles.root, className)} style={vars} data-mobile={isMobile ? '' : undefined} {...props}>
      {header && <header className={styles.header}>{header}</header>}
      {navbar && (
        <nav className={styles.navbar} data-opened={navbarOpened ? '' : undefined} inert={isMobile && !navbarOpened}>
          {navbar}
        </nav>
      )}
      <main className={styles.main}>{children}</main>
    </div>
  )
}
