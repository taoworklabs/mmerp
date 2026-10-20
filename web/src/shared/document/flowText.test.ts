import type { TFunction } from 'i18next'
import { describe, expect, it } from 'vitest'
import type { TypeRule } from '@/shared/api/core'
import { newStep, type CondValues } from './flowForm'
import { approverText, condText, stepDetail } from './flowText'

const words: Record<string, string> = {
  'shared.rule.and': ' và ',
  'shared.rule.or': ' hoặc ',
  'shared.rule.op_word.gt': '>',
  'shared.rule.op_word.eq': 'là',
  'shared.rule.op_word.within': 'thuộc',
}
// Known words read as the real text; any other key reads as itself, with its parameters.
const t = ((k: string, p?: Record<string, unknown>) => {
  if (words[k]) return words[k]
  const { defaultValue, ...rest } = p ?? {}
  if (Object.keys(rest).length) return `${k}:${JSON.stringify(rest)}`
  return defaultValue && k.startsWith('label.') ? k.slice(6) : k
}) as unknown as TFunction

const fields: TypeRule['fields'] = [
  { key: 'days', kind: 'number', label: 'label.Số ngày', ops: ['gt'], searchable: false },
  { key: 'kind', kind: 'choice', label: 'label.Loại nghỉ', ops: ['eq'], options: [{ value: 'unpaid', label: 'Không lương' }], searchable: false },
  { key: 'org_unit', kind: 'org_unit', label: 'label.Đơn vị', ops: ['within'], searchable: false },
]
const users = [
  { value: '1', label: 'An (an)' },
  { value: '2', label: 'Bình (binh)' },
  { value: '3', label: 'Cường (cuong)' },
]

describe('approverText', () => {
  it('names the approver', () => {
    expect(approverText(t, newStep(true), [], [])).toBe('shared.rule.box.module')
    expect(approverText(t, { ...newStep(false), role: 'hrm.hr' }, [{ value: 'hrm.hr', label: 'Nhân sự' }], [])).toBe('Nhân sự')
    expect(approverText(t, { ...newStep(false), role: null }, [], [])).toBe('shared.rule.box.pick_role')
    expect(approverText(t, { ...newStep(false), kind: 'user', users: ['1', '2'] }, [], users)).toBe('An, Bình')
    expect(approverText(t, { ...newStep(false), kind: 'user', users: ['1', '2', '3'] }, [], users)).toBe('An, Bình +1')
    expect(approverText(t, { ...newStep(false), kind: 'user', users: [] }, [], users)).toBe('shared.rule.box.pick_user')
  })
})

describe('stepDetail', () => {
  it('details only what differs from the defaults', () => {
    expect(stepDetail(t, newStep(true))).toBe('')
    expect(stepDetail(t, { ...newStep(true), dueHours: '24' })).toBe('shared.rule.box.due:{"hours":"24"}')
    expect(stepDetail(t, { ...newStep(true), approvals: '2' })).toBe('shared.rule.box.approvals_any:{"approvals":"2"}')
    expect(stepDetail(t, { ...newStep(false), kind: 'user', users: ['1', '2', '3'], approvals: '2', dueHours: '8' })).toBe(
      'shared.rule.box.approvals:{"approvals":"2","of":3} · shared.rule.box.due:{"hours":"8"}',
    )
    // One chosen user approving alone needs no count.
    expect(stepDetail(t, { ...newStep(false), kind: 'user', users: ['1'] })).toBe('')
  })
})

describe('condText', () => {
  it('reads a nested condition', () => {
    const cond: CondValues = {
      mode: 'any',
      rows: [
        { group: null, field: 'days', op: 'gt', value: '3' },
        {
          group: { mode: 'all', rows: [{ group: null, field: 'kind', op: 'eq', value: 'unpaid' }, { group: null, field: 'org_unit', op: 'within', value: '' }] },
          field: null,
          op: null,
          value: '',
        },
      ],
    }
    expect(condText(t, cond, fields, () => undefined)).toBe('Số ngày > 3 hoặc (Loại nghỉ là Không lương và …)')
  })

  it('uses the value label for org unit and searched fields', () => {
    const cond: CondValues = { mode: 'all', rows: [{ group: null, field: 'org_unit', op: 'within', value: '7' }] }
    expect(condText(t, cond, fields, (f, v) => (f.key === 'org_unit' && v === '7' ? 'Kế toán' : undefined))).toBe('Đơn vị thuộc Kế toán')
    expect(condText(t, cond, fields, () => undefined)).toBe('Đơn vị thuộc 7')
  })

  it('no rows reads as nothing', () => {
    expect(condText(t, { mode: 'all', rows: [] }, fields, () => undefined)).toBe('')
  })
})
