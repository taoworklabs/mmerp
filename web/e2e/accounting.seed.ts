import { expect, test, type APIResponse } from '@playwright/test'
import { created, signedInApi } from './api'
import { admin as root } from './helpers'

// Accounting on top of the Demo demo company (demo.seed.ts runs first): a small chart of
// accounts, the payroll mapping rules, a start date per company, an opening-balance entry,
// then "post pending" so the seeded payrolls reach the ledger. Skipped once accounts exist.

async function ok(res: Promise<APIResponse>, what: string) {
  const r = await res
  expect(r.ok(), `${what}: ${r.status()} ${await r.text()}`).toBe(true)
}

type Type = 'asset' | 'liability' | 'equity' | 'expense'
// [code, name, type, parent code]
const chart: [string, string, Type, string?][] = [
  ['111', 'Tiền mặt', 'asset'],
  ['112', 'Tiền gửi ngân hàng', 'asset'],
  ['333', 'Thuế và các khoản phải nộp Nhà nước', 'liability'],
  ['3335', 'Thuế thu nhập cá nhân', 'liability', '333'],
  ['334', 'Phải trả người lao động', 'liability'],
  ['338', 'Phải trả, phải nộp khác', 'liability'],
  ['411', 'Vốn đầu tư của chủ sở hữu', 'equity'],
  ['642', 'Chi phí quản lý doanh nghiệp', 'expense'],
]
// [kind, account code, side]
const rules: [string, string, 'debit' | 'credit'][] = [
  ['salary_expense', '642', 'debit'],
  ['employer_insurance_expense', '642', 'debit'],
  ['salary_payable', '334', 'credit'],
  ['insurance_payable', '338', 'credit'],
  ['pit_payable', '3335', 'credit'],
]

test('seed Accounting on the Demo demo company', async ({ baseURL }) => {
  test.setTimeout(120_000)
  const api = await signedInApi(baseURL!)
  const list = async <T>(path: string) => ((await (await api.get(path)).json()) as { items: T[] }).items
  if ((await list('/api/accounting/accounts')).length > 0) test.skip(true, 'Accounting data already there')

  // The administrator account does the work, so it needs an Accounting role of its own.
  const users = (await (await api.get('/api/users')).json()) as { id: number; login: string }[]
  const me = users.find((u) => u.login === root.login)!.id
  const role = await api.post(`/api/users/${me}/roles`, { data: { product: 'accounting', role: 'admin', org_unit_id: null } })
  expect([201, 409], await role.text()).toContain(role.status())

  const id: Record<string, number> = {}
  for (const [code, name, type, parent] of chart) {
    id[code] = await created(api.post('/api/accounting/accounts', { data: { code, name, type, parent_id: parent ? id[parent] : null, active: true } }))
  }
  for (const [kind, code, side] of rules) {
    await ok(api.put(`/api/accounting/rules/${kind}`, { data: { account_id: id[code], side } }), `rule ${kind}`)
  }

  const units = (await (await api.get('/api/org-units')).json()) as { id: number; name: string; kind: string }[]
  const companies = units.filter((u) => u.kind === 'company')
  for (const c of companies) await ok(api.put(`/api/accounting/starts/${c.id}`, { data: { start_date: '2026-01-01' } }), `start ${c.name}`)

  // Opening balances of Demo: 500 million in the bank against owner capital.
  const demo = companies.find((c) => c.name === 'Công ty CP Demo')!.id
  const entry = await created(
    api.post('/api/accounting/journal-entries', {
      data: {
        request_id: crypto.randomUUID(),
        date: '2026-01-01',
        legal_entity_id: demo,
        description: 'Số dư đầu kỳ',
        lines: [
          { account_id: id['112'], debit: 500_000_000, credit: 0 },
          { account_id: id['411'], debit: 0, credit: 500_000_000 },
        ],
      },
    }),
  )
  await ok(api.post(`/api/documents/accounting.journal_entry/${entry}/transitions`, { data: { to: 'posted', version: 1 } }), 'post opening entry')

  // The payrolls were posted before Accounting existed; queue them for the ledger.
  for (const c of companies) await ok(api.post('/api/accounting/pending/post', { data: { legal_entity_id: c.id } }), `post pending ${c.name}`)
})
