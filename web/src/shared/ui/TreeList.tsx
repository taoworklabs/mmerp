import { Group, Text, UnstyledButton, cx } from '@mmerp/ui'
import { IconChevronDown, IconChevronRight } from '@tabler/icons-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { MoreActions } from './MoreActions'
import { icon } from './theme'
import classes from './TreeList.module.css'

export type TreeNode<T> = { key: string | number; item: T; children: TreeNode<T>[] }

type Props<T> = {
  label: string // accessible name of the tree
  nodes: TreeNode<T>[]
  render: (item: T) => ReactNode
  // Quiet facts after the name (kind, code). Below 1024px they, and the count a folded branch
  // hides, are left out: the name needs the room.
  meta?: (item: T) => ReactNode
  // Secondary actions of a node, as menu items of the "…" menu at its end.
  menu?: (item: T) => ReactNode
  // Text of the fold button, for screen readers.
  toggleLabel: (item: T, folded: boolean) => string
}

// TreeList shows a hierarchy as indented rows joined by guide lines; every branch folds.
// All branches start open.
export function TreeList<T>({ label, nodes, ...rest }: Props<T>) {
  const [folded, setFolded] = useState<ReadonlySet<string | number>>(new Set())
  const toggle = (key: string | number) =>
    setFolded((f) => {
      const next = new Set(f)
      if (!next.delete(key)) next.add(key)
      return next
    })
  return (
    <ul className={classes.tree} aria-label={label}>
      {nodes.map((n) => (
        <Node key={n.key} node={n} folded={folded} toggle={toggle} {...rest} />
      ))}
    </ul>
  )
}

function Node<T>({ node, folded, toggle, render, meta, menu, toggleLabel }: Omit<Props<T>, 'label' | 'nodes'> & { node: TreeNode<T>; folded: ReadonlySet<string | number>; toggle: (key: string | number) => void }) {
  const { t } = useTranslation()
  const leaf = node.children.length === 0
  const closed = folded.has(node.key)
  return (
    <li className={classes.node} data-leaf={leaf || undefined}>
      <div className={classes.row}>
        {leaf ? (
          <span className={classes.toggle} aria-hidden />
        ) : (
          <UnstyledButton className={classes.toggle} aria-expanded={!closed} aria-label={toggleLabel(node.item, closed)} onClick={() => toggle(node.key)}>
            {closed ? <IconChevronRight {...icon.text} /> : <IconChevronDown {...icon.text} />}
          </UnstyledButton>
        )}
        <Group gap="sm" wrap={false} className={classes.label} fill>
          {render(node.item)}
          {meta && (
            <Text size="sm" tone="dimmed" truncate className={classes.meta}>
              {meta(node.item)}
            </Text>
          )}
          {closed && (
            <Text size="sm" tone="dimmed" className={cx(classes.meta, classes.fixed)}>
              {t('shared.tree.hidden', { count: node.children.length })}
            </Text>
          )}
        </Group>
        {menu && <MoreActions>{menu(node.item)}</MoreActions>}
      </div>
      {!leaf && !closed && (
        <ul className={classes.group}>
          {node.children.map((c) => (
            <Node key={c.key} node={c} folded={folded} toggle={toggle} render={render} meta={meta} menu={menu} toggleLabel={toggleLabel} />
          ))}
        </ul>
      )}
    </li>
  )
}
