import type { TFunction } from 'i18next'
import type { TypeRule } from '@/shared/api/core'
import type { Option } from '@/shared/ui/form'
import type { CondValues, ElementValues } from './flowForm'

type Field = TypeRule['fields'][number]

// A user option reads "Name (login)"; a box has room for the name only.
const nameOf = (label: string) => label.replace(/ \([^)]*\)$/, '')

// approverText says who approves a step, as its box on the diagram shows it.
export function approverText(t: TFunction, el: ElementValues, roles: Option[], users: Option[]): string {
  if (el.kind === 'role') return roles.find((o) => o.value === el.role)?.label ?? t('shared.rule.box.pick_role')
  if (el.kind === 'user') {
    if (el.users.length === 0) return t('shared.rule.box.pick_user')
    const names = el.users.map((u) => nameOf(users.find((o) => o.value === u)?.label ?? u))
    return names.length > 2 ? `${names.slice(0, 2).join(', ')} +${names.length - 2}` : names.join(', ')
  }
  return t('shared.rule.box.module')
}

// stepDetail is a step box's second line: only what differs from one approval and no due time.
export function stepDetail(t: TFunction, el: ElementValues): string {
  const parts: string[] = []
  if (el.kind === 'user' && el.users.length > 1) parts.push(t('shared.rule.box.approvals', { approvals: el.approvals, of: el.users.length }))
  else if (el.kind !== 'user' && Number(el.approvals) > 1) parts.push(t('shared.rule.box.approvals_any', { approvals: el.approvals }))
  if (el.dueHours) parts.push(t('shared.rule.box.due', { hours: el.dueHours }))
  return parts.join(' · ')
}

// ValueLabel names a condition's value where the value itself is an id (an org unit, a searched option).
export type ValueLabel = (field: Field, value: string) => string | undefined

// condText reads a condition back in words: "Số ngày > 3 hoặc (Loại nghỉ là Không lương và …)".
export function condText(t: TFunction, cond: CondValues, fields: Field[], valueLabel: ValueLabel): string {
  return cond.rows
    .map((r) => {
      if (r.group) return `(${condText(t, r.group, fields, valueLabel)})`
      const f = fields.find((x) => x.key === r.field)
      if (!f || !r.op || r.value === '') return '…'
      const value = (f.kind === 'choice' ? f.options?.find((o) => o.value === r.value)?.label : valueLabel(f, r.value)) ?? r.value
      return `${t(f.label, { defaultValue: f.key })} ${t(`shared.rule.op_word.${r.op}`)} ${value}`
    })
    .join(t(cond.mode === 'all' ? 'shared.rule.and' : 'shared.rule.or'))
}
