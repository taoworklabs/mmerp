import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Page.module.css'

export type PageVariant = 'full' | 'centered' | 'detail' | 'fill'

export interface PageProps extends React.ComponentPropsWithRef<'div'> {
  variant?: PageVariant
  header: React.ReactNode
  children: React.ReactNode
}

// The frame every page builds on: header, then content, with the padding and gap shared by every
// product screen. "centered"/"detail" stop at a fixed width so the header and content share a
// left edge; "fill" instead fills the AppShell main, its content scrolling in Page.Content while
// the header stays put.
function PageRoot({ variant = 'full', header, children, className, ...props }: PageProps) {
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
