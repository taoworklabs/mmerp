import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './TableOfContents.module.css'

export interface TableOfContentsItem {
  id: string
  label: React.ReactNode
  depth?: 1 | 2 | 3
}

export interface TableOfContentsProps extends Omit<React.ComponentPropsWithRef<'nav'>, 'children'> {
  items: TableOfContentsItem[]
  activeId?: string
  'aria-label': string
}

export function TableOfContents({ items, activeId, className, ...props }: TableOfContentsProps) {
  return (
    <nav className={cx(styles.root, className)} {...props}>
      <ul className={styles.list}>
        {items.map((item) => (
          <li key={item.id} className={styles.item} data-depth={item.depth ?? 1}>
            <a
              href={`#${item.id}`}
              className={styles.link}
              data-active={item.id === activeId ? '' : undefined}
              aria-current={item.id === activeId ? 'location' : undefined}
            >
              {item.label}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  )
}
