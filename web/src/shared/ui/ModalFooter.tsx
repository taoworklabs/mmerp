import { Group } from '@mmerp/ui'
import type { ReactNode } from 'react'
import classes from './ModalFooter.module.css'

// ModalFooter holds a modal's buttons, right-aligned in a band across the modal under a rule.
export function ModalFooter({ children }: { children: ReactNode }) {
  return (
    <Group justify="end" className={classes.footer}>
      {children}
    </Group>
  )
}
