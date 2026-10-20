import { createContext, useContext, useState, type ReactNode } from 'react'
import { get, useFormContext, useFormState } from 'react-hook-form'
import type { TypeRule } from '@/shared/api/core'
import type { Option } from '@/shared/ui/form'
import { Notice } from '@/shared/ui/Notice'
import { parentOf, type BranchValues, type ElementValues, type Selection } from '../flowForm'

export type EditorCtx = { rule: TypeRule; roleOptions: Option[]; userOptions: Option[]; readOnly: boolean }

type FlowEditorValue = {
  ctx: EditorCtx
  selected: Selection
  select: (s: Selection) => void
  // Bumped by every structural edit, so the side panel remounts on paths that now mean something else.
  rev: number
  insert: (list: string, at: number, item: ElementValues | BranchValues) => void
  remove: (path: string) => void
  move: (path: string, by: -1 | 1) => void
}

const FlowEditorContext = createContext<FlowEditorValue | null>(null)

export function useFlowEditor() {
  const v = useContext(FlowEditorContext)
  if (!v) throw new Error('useFlowEditor outside FlowEditor')
  return v
}

// FlowEditor holds the editor's edits: whole lists are rewritten, so the diagram (which reads the
// form's values) and the panel (keyed by rev) both follow; the selection follows what it pointed at.
export function FlowEditor({ ctx, selected, select, children }: { ctx: EditorCtx; selected: Selection; select: (s: Selection) => void; children: ReactNode }) {
  const { getValues, setValue } = useFormContext()
  const [rev, setRev] = useState(0)
  const write = (list: string, items: unknown[]) => {
    setValue(list, items, { shouldDirty: true })
    setRev((r) => r + 1)
  }
  const itemsOf = (list: string) => [...((getValues(list) as unknown[] | undefined) ?? [])]
  const value: FlowEditorValue = {
    ctx,
    selected,
    select,
    rev,
    insert(list, at, item) {
      const items = itemsOf(list)
      items.splice(at, 0, item)
      write(list, items)
      select(`${list}.${at}`)
    },
    remove(path) {
      const [list, i] = parentOf(path)
      const items = itemsOf(list)
      items.splice(i, 1)
      write(list, items)
      select(null)
    },
    move(path, by) {
      const [list, i] = parentOf(path)
      const items = itemsOf(list)
      const j = i + by
      if (j < 0 || j >= items.length) return
      ;[items[i], items[j]] = [items[j], items[i]]
      write(list, items)
      select(`${list}.${j}`)
    },
  }
  return <FlowEditorContext.Provider value={value}>{children}</FlowEditorContext.Provider>
}

// IssueAlert shows the complaint about one element (or the settings) at the top of its panel.
export function IssueAlert({ name }: { name: string }) {
  const { errors } = useFormState({ name })
  const issue: string | undefined = get(errors, name)?.message
  if (!issue) return null
  return <Notice tone="danger">{issue}</Notice>
}
