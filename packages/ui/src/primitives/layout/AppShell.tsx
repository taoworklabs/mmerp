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
  // Pinned to the navbar's bottom (a rail-collapse toggle); hidden below the drawer breakpoint,
  // where the rest of the navbar's chrome already assumes desktop width.
  navbarFooter?: React.ReactNode
  // Accessible name of the navbar landmark, so it is told apart from other navigation landmarks.
  navbarLabel?: string
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
  navbarFooter,
  navbarLabel,
  className,
  style,
  children,
  ...props
}: AppShellProps) {
  const isMobile = useMediaQuery(mobileQuery)
  // The rail only applies at desktop widths; on mobile the navbar is always the full-width drawer.
  // Without a navbar (a home page) main takes the whole width.
  const effectiveNavbarWidth = !navbar ? 0 : !isMobile && navbarCollapsed ? navbarCollapsedWidth : navbarWidth
  const vars: StyleVars = {
    ...style,
    '--mmerp-appshell-header-height': `${headerHeight}px`,
    '--mmerp-appshell-navbar-width': `${effectiveNavbarWidth}px`,
  }
  return (
    <div className={cx(styles.root, className)} style={vars} data-mobile={isMobile ? '' : undefined} {...props}>
      {header && <header className={styles.header}>{header}</header>}
      {navbar && (
        <nav className={styles.navbar} aria-label={navbarLabel} data-opened={navbarOpened ? '' : undefined} inert={isMobile && !navbarOpened}>
          <div className={styles.navbarContent}>{navbar}</div>
          {navbarFooter && <div className={styles.navbarFooter}>{navbarFooter}</div>}
        </nav>
      )}
      <main className={styles.main}>{children}</main>
    </div>
  )
}
