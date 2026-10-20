import { Children } from 'react'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Breadcrumbs.module.css'

export interface BreadcrumbsProps extends Omit<React.ComponentPropsWithRef<'nav'>, 'children'> {
  'aria-label': string
  separator?: React.ReactNode
  children: React.ReactNode
}

export function Breadcrumbs({ separator = '/', className, children, ...props }: BreadcrumbsProps) {
  const items = Children.toArray(children)
  return (
    <nav className={cx(styles.root, className)} {...props}>
      <ol className={styles.list}>
        {items.map((item, index) => {
          const isLast = index === items.length - 1
          return (
            <li key={index} className={styles.item} aria-current={isLast ? 'page' : undefined}>
              {item}
              {!isLast && (
                <span className={styles.separator} aria-hidden="true">
                  {separator}
                </span>
              )}
            </li>
          )
        })}
      </ol>
    </nav>
  )
}
