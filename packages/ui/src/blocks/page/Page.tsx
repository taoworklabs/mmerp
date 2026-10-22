import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Page.module.css'

export type PageVariant = 'full' | 'centered' | 'detail' | 'fill' | 'list'

export interface PageProps extends React.ComponentPropsWithRef<'div'> {
  variant?: PageVariant
  header: React.ReactNode
  children: React.ReactNode
  // "list" only: the filter bar under the header, and the bar pinned at the bottom (paging).
  filters?: React.ReactNode
  footer?: React.ReactNode
}

// The frame every page builds on: header, then content, with the padding and gap shared by every
// product screen. "centered"/"detail" stop at a fixed width so the header and content share a
// left edge; "fill" instead fills the AppShell main, its content scrolling in Page.Content while
// the header stays put. "list" fills it too: header and filters at the page's margins, then the
// table edge to edge (the only part that scrolls), then the footer pinned at the bottom.
function PageRoot({ variant = 'full', header, filters, footer, children, className, ...props }: PageProps) {
  if (variant === 'list') {
    return (
      <div className={cx(styles.list, className)} {...props}>
        <div className={styles.listHead}>
          {header}
          {filters}
        </div>
        <div className={styles.listBody}>{children}</div>
        {footer && <div className={styles.listFoot}>{footer}</div>}
      </div>
    )
  }
  const body = (
    <>
      {header}
      {children}
    </>
  )
  if (variant === 'centered' || variant === 'detail') {
    return (
      <div className={cx(styles.root, className)} {...props}>
        <div className={cx(styles.inner, variant === 'detail' && styles.detail)}>{body}</div>
      </div>
    )
  }
  return (
    <div className={cx(styles.root, variant === 'fill' && styles.fill, className)} {...props}>
      {body}
    </div>
  )
}

// The scrolling area of a "fill" page; place it inside children alongside nothing else.
// tabIndex makes the scroll region itself reachable by keyboard, not just the controls inside it.
function PageContent({ className, tabIndex = 0, ...props }: React.ComponentPropsWithRef<'div'>) {
  return <div className={cx(styles.content, className)} tabIndex={tabIndex} {...props} />
}

export const Page = Object.assign(PageRoot, { Content: PageContent })
