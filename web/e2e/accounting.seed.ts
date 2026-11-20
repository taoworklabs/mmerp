import { expect, test, type APIResponse } from '@playwright/test'
import { created, signedInApi } from './api'
import { admin as root } from './helpers'

// Accounting on top of the Demo demo company (demo.seed.ts, sales.seed.ts and purchase.seed.ts run
// first): the Circular 133 chart template (chart, mapping rules, closing accounts, statement forms),
// a start date per company, an opening-balance entry with two customers and a supplier, then "post
// pending" so the seeded payrolls and documents reach the ledger. Skipped once accounts exist.

async function ok(res: Promise<APIResponse>, what: string) {
  const r = await res
  expect(r.ok(), `${what}: ${r.status()} ${await r.text()}`).toBe(true)
}

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

  await ok(api.post('/api/accounting/templates/vn-tt133/apply'), 'apply the Circular 133 template')
  const id = Object.fromEntries((await list<{ id: number; code: string }>('/api/accounting/accounts')).map((a) => [a.code, a.id]))

  const units = (await (await api.get('/api/org-units')).json()) as { id: number; name: string; kind: string }[]
  const companies = units.filter((u) => u.kind === 'company')
  for (const c of companies) await ok(api.put(`/api/accounting/starts/${c.id}`, { data: { start_date: '2026-01-01' } }), `start ${c.name}`)

  // Opening balances of Demo: 500 million in the bank, what two customers owe and what a
  // supplier is owed, against owner capital.
  const demo = companies.find((c) => c.name === 'Công ty CP Demo')!.id
  const partners = ((await (await api.get('/api/partners?page_size=20&sort=code')).json()) as { items: { id: number }[] }).items
  expect(partners.length, 'Sales seeded its customers').toBeGreaterThanOrEqual(2)
  const [first, second] = partners as [{ id: number }, { id: number }]
  const supplier = await created(
    api.post('/api/partners', { data: { name: 'Công ty TNHH Văn phòng phẩm Hồng Hà', tax_code: '0101234567', address: null, phone: null, email: null, contact_name: null } }),
  )
  const entry = await created(
    api.post('/api/accounting/journal-entries', {
      data: {
        request_id: crypto.randomUUID(),
        date: '2026-07-01',
        legal_entity_id: demo,
        description: 'Số dư đầu kỳ',
        lines: [
          { account_id: id['1121'], debit: 500_000_000, credit: 0 },
          { account_id: id['131'], partner_id: first.id, debit: 80_000_000, credit: 0 },
          { account_id: id['131'], partner_id: second.id, debit: 45_000_000, credit: 0 },
          { account_id: id['331'], partner_id: supplier, debit: 0, credit: 60_000_000 },
          { account_id: id['4111'], debit: 0, credit: 565_000_000 },
        ],
      },
    }),
  )
  await ok(api.post(`/api/documents/accounting.journal_entry/${entry}/transitions`, { data: { to: 'posted', version: 1 } }), 'post opening entry')

  // The payrolls, invoices, receipts and payments were posted before Accounting existed; queue them for the ledger.
  for (const c of companies) await ok(api.post('/api/accounting/pending/post', { data: { legal_entity_id: c.id } }), `post pending ${c.name}`)
})
