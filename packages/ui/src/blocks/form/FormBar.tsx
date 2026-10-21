import type * as React from 'react'
import { useCallback, useMemo, useState } from 'react'
import { cx } from '../../internal/cx.ts'
import { mergeRefs } from '../../internal/mergeRefs.ts'
import { Group } from '../../primitives/layout/Group.tsx'
import styles from './FormLayout.module.css'

export interface FormBarProps extends React.ComponentPropsWithRef<'div'> {
  children: React.ReactNode
  // Sits left of the buttons (an unsaved indicator, an error count); the kit only places it.
  status?: React.ReactNode
}

// FormBar is a form's footer: its buttons on the right, with status beside them. On a page it
// sticks to the bottom of the screen, casting a shadow while the form runs on under it.
function FormBarRoot({ children, status, className, ref, ...props }: FormBarProps) {
  const [stuck, setStuck] = useState(false)
  // The bar sits 1px past the screen's edge while pinned, so it is then never fully in view.
  const pinned = useCallback((el: HTMLDivElement | null) => {
    if (!el) return
    const io = new IntersectionObserver(([entry]) => setStuck((entry?.intersectionRatio ?? 1) < 1), { threshold: 1 })
    io.observe(el)
    return () => io.disconnect()
  }, [])
  // Memoised so the merged ref keeps its identity across renders: without it, every render would
  // detach and reattach pinned, tearing down and recreating the IntersectionObserver each time.
  const mergedRef = useMemo(() => mergeRefs(ref, pinned), [ref, pinned])
  return (
    <Group ref={mergedRef} justify="end" gap="lg" wrap={false} className={cx(styles.bar, className)} data-stuck={stuck || undefined} {...props}>
      {status}
      <Group gap="sm" wrap={false}>
        {children}
      </Group>
    </Group>
  )
}

export type FormBarUnsavedDotProps = React.ComponentPropsWithRef<'span'>

// UnsavedDot marks a status line as holding unsaved edits; the text beside it is the caller's.
function UnsavedDot({ className, ...props }: FormBarUnsavedDotProps) {
  return <span aria-hidden className={cx(styles.dot, className)} {...props} />
}

export const FormBar = Object.assign(FormBarRoot, { UnsavedDot })
