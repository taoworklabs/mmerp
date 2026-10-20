import { createContext, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import { get, useFormContext, useFormState, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import type { TypeRule } from '@/shared/api/core'
import type { Option } from '@/shared/ui/form'
import { Notice } from '@/shared/ui/Notice'
import { parentOf, type BranchValues, type ElementValues, type RuleValues, type Selection } from '../flowForm'
import { flowProblems, type Problem } from '../flowProblems'

export type EditorCtx = { rule: TypeRule; roleOptions: Option[]; userOptions: Option[]; readOnly: boolean }

type FlowEditorValue = {
  ctx: EditorCtx
  selected: Selection
  select: (s: Selection) => void
  // Bumped by every structural edit and by a discard, so the side panel remounts on paths that now
  // mean something else.
  rev: number
  insert: (list: string, at: number, item: ElementValues | BranchValues) => void
  remove: (path: string) => void
  move: (path: string, by: -1 | 1) => void
  // The rule's problems now, by the input that fixes each, and by the element they mark.
  problemOf: (field: string) => Problem | undefined
  problemAt: (at: string) => Problem | undefined
  // True once after an insert: the panel that opens then takes the focus.
  takeFocus: () => boolean
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
  const { getValues, setValue, watch } = useFormContext()
  const [rev, setRev] = useState(0)
  const focus = useRef(false)
  const values = useWatch() as RuleValues
  const problems = flowProblems(values)
  const byField = new Map(problems.flatMap((p) => (p.field ? [[p.field, p] as const] : [])))
  const byAt = new Map(problems.map((p) => [p.at, p]))
  // A reset (Discard, or the reload after a save) names no field; the open panel starts over.
  useEffect(() => {
    const sub = watch((_, { name }) => {
      if (!name) setRev((r) => r + 1)
    })
    return () => sub.unsubscribe()
  }, [watch])
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
      focus.current = true
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
    problemOf: (field) => byField.get(field),
    problemAt: (at) => byAt.get(at),
    takeFocus() {
      const was = focus.current
      focus.current = false
      return was
    },
  }
  return <FlowEditorContext.Provider value={value}>{children}</FlowEditorContext.Provider>
}

// useFieldProblem is the message to show under one input of the panel: the rule's problem that this
// input fixes, once the input was left or a save was tried, as the form's own checks do. afterSave
// waits for a save: a condition row's remove button sits level with its inputs, and a message
// appearing on blur would move it from under the pointer mid-click.
export function useFieldProblem(name: string, afterSave = false): string | undefined {
  const { t } = useTranslation()
  const { problemOf } = useFlowEditor()
  const { touchedFields, submitCount } = useFormState({ name })
  const problem = problemOf(name)
  if (!problem || (submitCount === 0 && (afterSave || !get(touchedFields, name)))) return undefined
  return t(`shared.rule.reason.${problem.code}`)
}

// IssueAlert shows the complaint about one element (or the settings) at the top of its panel,
// unless an input of the panel already shows it.
export function IssueAlert({ name }: { name: string }) {
  const { errors } = useFormState({ name })
  const { problemAt } = useFlowEditor()
  const issue: string | undefined = get(errors, name)?.message
  if (!issue || problemAt(name)?.field) return null
  return <Notice tone="danger">{issue}</Notice>
}
