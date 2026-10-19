import type { Flow, FlowCondition, FlowElement } from '@/shared/api/core'

// One step of the chain editor: its approver and an optional condition on an approval field.
export type StepValues = { kind: string | null; role: string | null; user: string | null; field: string | null; op: string | null; value: string }

type StepDef = NonNullable<FlowElement['step']>

// splitRole parses a "product.role" select value.
export const splitRole = (v: string) => {
  const [product = '', ...rest] = v.split('.')
  return { product, role: rest.join('.') }
}

function toStep(s: StepValues): FlowElement {
  const approver: StepDef['approver'] =
    s.kind === 'role' ? { kind: 'role', ...splitRole(s.role ?? '') } : s.kind === 'user' ? { kind: 'user', user_ids: [Number(s.user)] } : { kind: 'module' }
  return { step: { approver, approvals: 1, rejections: 1 } }
}

// toFlow writes a chain as a flow: a conditional step becomes a choose-one split with one branch.
export function toFlow(steps: StepValues[]): Flow {
  return steps.map((s) =>
    s.field && s.op
      ? { split: { mode: 'one', branches: [{ condition: { field: s.field, op: s.op as FlowCondition['op'], value: s.value }, body: [toStep(s)] }], otherwise: [] } }
      : toStep(s),
  )
}

function fromStep(st: StepDef | undefined, cond?: FlowCondition): StepValues | null {
  if (!st || st.approvals !== 1 || st.rejections !== 1) return null
  const a = st.approver
  if (a.kind === 'user' && a.user_ids?.length !== 1) return null
  return {
    kind: a.kind,
    role: a.kind === 'role' ? `${a.product}.${a.role}` : null,
    user: a.kind === 'user' ? String(a.user_ids?.[0]) : null,
    field: cond?.field ?? null,
    op: cond?.op ?? null,
    value: cond?.value ?? '',
  }
}

// fromFlow reads back a flow toFlow could have written, or null for any other shape
// (counts above one, parallel splits, several branches, nested conditions), which the chain editor cannot show.
export function fromFlow(flow: Flow): StepValues[] | null {
  const out: StepValues[] = []
  for (const e of flow) {
    let s: StepValues | null = null
    if (e.step) s = fromStep(e.step)
    else if (e.split?.mode === 'one' && e.split.branches.length === 1 && !e.split.otherwise?.length) {
      const b = e.split.branches[0]
      const c = b?.condition
      if (b?.body.length === 1 && c?.field && !c.all?.length && !c.any?.length) s = fromStep(b.body[0]?.step, c)
    }
    if (!s) return null
    out.push(s)
  }
  return out
}
