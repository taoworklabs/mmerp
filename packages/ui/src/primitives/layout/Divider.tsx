import { Separator } from '@base-ui/react/separator'
import { cx } from '../../internal/cx.ts'
import styles from './Divider.module.css'

export interface DividerProps extends Omit<Separator.Props, 'className'> {
  className?: string
}

// Base UI's Separator already carries render, orientation and the data-orientation attribute our CSS reads.
export function Divider({ className, ...props }: DividerProps) {
  return <Separator className={cx(styles.root, className)} {...props} />
}
