import { Collapsible } from '@base-ui/react/collapsible'
import { useRender } from '@base-ui/react/use-render'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import type { RenderProp } from '../../internal/render.ts'
import styles from './NavLink.module.css'

export interface NavLinkProps extends Omit<React.ComponentPropsWithRef<'a'>, 'children'> {
  label: React.ReactNode
  leftSection?: React.ReactNode
  rightSection?: React.ReactNode
  description?: React.ReactNode
  active?: boolean
  compact?: boolean
  render?: RenderProp
  children?: React.ReactNode
  defaultOpened?: boolean
}

function ChevronIcon() {
  return (
    <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true" className={styles.chevron}>
      <path d="M2 4l3 3 3-3" stroke="currentColor" strokeWidth="1.4" fill="none" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function NavLinkRoot({
  label,
  leftSection,
  rightSection,
  description,
  active,
  compact,
  render,
  children,
  defaultOpened,
  className,
  ...props
}: NavLinkProps) {
  const body = (
    <>
      {leftSection && <span className={styles.section}>{leftSection}</span>}
      <span className={styles.body}>
        <span className={styles.label}>{label}</span>
        {description && <span className={styles.description}>{description}</span>}
      </span>
      {rightSection && <span className={styles.section}>{rightSection}</span>}
    </>
  )

  if (children) {
    return (
      <Collapsible.Root defaultOpen={defaultOpened} className={styles.group}>
        <Collapsible.Trigger
          className={cx(styles.root, className)}
          data-active={active ? '' : undefined}
          data-compact={compact ? '' : undefined}
          {...(props as React.ComponentPropsWithRef<'button'>)}
        >
          {body}
          <ChevronIcon />
        </Collapsible.Trigger>
        <Collapsible.Panel className={styles.panel}>{children}</Collapsible.Panel>
      </Collapsible.Root>
    )
  }

  return useRender({
    defaultTagName: 'a',
    render,
    props: {
      className: cx(styles.root, className),
      'data-active': active ? '' : undefined,
      'data-compact': compact ? '' : undefined,
      children: body,
      ...props,
    },
  })
}

export type NavLinkGroupLabelProps = React.ComponentPropsWithRef<'p'>

// GroupLabel heads a group of nav items: the caption a quiet nav group needs, indented to sit
// above its links rather than flush with the nav's own edge.
function GroupLabel({ className, ...props }: NavLinkGroupLabelProps) {
  return <p className={cx(styles.groupLabel, className)} {...props} />
}

export const NavLink = Object.assign(NavLinkRoot, { GroupLabel })
