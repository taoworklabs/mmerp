import { expect, test, type APIRequestContext } from '@playwright/test'
import { created, createUser, employee, ok, password, runId, signedInApi, toFlow } from './api'
import { signedIn } from './helpers'

// One run, in order: a company of its own with one employee under contract and a March
// timesheet. pay posts the payroll, dir approves it; root plays the accountant. The tests walk
// the "done" list: chart and rules, a payroll posted before its rules exist and later posted
// from the pending list, its cancellation, a manual entry, and the trial balance. The file
// sets mapping rules for the whole tenant, so it runs alone.
const run = runId()
const login = { pay: `acpay_${run}`, dir: `acdir_${run}` }
const from = '2026-01-01'
const to = '2026-12-31'
let api: APIRequestContext
let pay: APIRequestContext
let c: number
let dept: number
let payroll: { id: number; number: string }
const acct: Record<string, number> = {}

const version = async (as: APIRequestContext, path: string) => ((await (await as.get(path)).json()) as { version: number }).version
const move = async (as: APIRequestContext, type: string, path: string, id: number, to: string) =>
  ok(as.post(`/api/documents/${type}/${id}/transitions`, { data: { to, version: await version(as, `${path}/${id}`) } }))

type Ledger = { opening: number; closing: number; lines: { debit: number; credit: number; date: string; org_unit_id: number | null; doc_number: string }[] }
const ledger = async (code: string): Promise<Ledger> =>
  (await api.get(`/api/accounting/ledger?legal_entity_id=${c}&account_id=${acct[code]}&from=${from}&to=${to}`)).json() as Promise<Ledger>
const sum = (ls: Ledger['lines'], side: 'debit' | 'credit') => ls.reduce((n, l) => n + l[side], 0)

test.beforeAll(async ({ baseURL }) => {
  test.setTimeout(120_000)
  api = await signedInApi(baseURL!)
  c = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty AC ${run}` } }))
  dept = await created(api.post('/api/org-units', { data: { parent_id: c, kind: 'department', name: `Phòng AC ${run}` } }))
  const emp = await created(api.post('/api/hrm/employees', { data: employee(`ACE${run}`, dept, { hire_date: '2026-01-05' }) }))
  await createUser(api, login.pay, [
    { role: 'hrm.payroll', unit: c },
    { role: 'hrm.hr', unit: c },
  ])
  const dir = await createUser(api, login.dir, [{ role: 'hrm.payroll', unit: null }])
  const me = ((await (await api.get('/api/users')).json()) as { id: number; login: string }[]).find((u) => u.login === 'root')
  if (me) await api.post(`/api/users/${me.id}/roles`, { data: { product: 'accounting', role: 'admin', org_unit_id: null } })
  for (const type of ['hrm.contract', 'hrm.timesheet', 'hrm.overtime_request']) await api.delete(`/api/approval-rules/${type}`)
  await ok(api.put('/api/approval-rules/hrm.payroll', { data: { flow: toFlow([{ approver: { kind: 'user', user_ids: [dir] } }]), max_levels: 1, fallback_product: 'hrm', fallback_role: 'payroll' } }))

  const kind = await created(api.post('/api/hrm/contract-types', { data: { name: `Không thời hạn AC ${run}`, fixed_term: false, active: true } }))
  const contract = await created(
    api.post('/api/hrm/contracts', { data: { employee_id: emp, contract_type_id: kind, start_date: '2026-01-01', end_date: null, terms: { salary: 22_000_000, lines: [] } } }),
  )
  await ok(api.post(`/api/documents/hrm.contract/${contract}/transitions`, { data: { to: 'posted', version: 1 } }))
  const ts = await created(api.post('/api/hrm/timesheets', { data: { org_unit_id: dept, month: '2026-03' } }))
  const lines = []
  for (let d = 1; d <= 31; d++) {
    const date = `2026-03-${String(d).padStart(2, '0')}`
    const wd = new Date(`${date}T00:00:00Z`).getUTCDay()
    if (wd !== 0 && wd !== 6) lines.push({ employee_id: emp, date, days: '1' })
  }
  await ok(api.put(`/api/hrm/timesheets/${ts}`, { data: { version: 1, month: '2026-03', lines } }))
  await ok(api.post(`/api/documents/hrm.timesheet/${ts}/transitions`, { data: { to: 'posted', version: 2 } }))
  pay = await signedInApi(baseURL!, login.pay, password)
})

test('the chart of accounts is set up and shown; no rule is set yet', async ({ page }) => {
  const chart: [string, string, string, string?][] = [
    ['111', 'Tiền mặt', 'asset'],
    ['112', 'Tiền gửi ngân hàng', 'asset'],
    ['333', 'Thuế và các khoản phải nộp Nhà nước', 'liability'],
    ['3335', 'Thuế thu nhập cá nhân', 'liability', '333'],
    ['334', 'Phải trả người lao động', 'liability'],
    ['338', 'Phải trả, phải nộp khác', 'liability'],
    ['411', 'Vốn đầu tư của chủ sở hữu', 'equity'],
    ['642', 'Chi phí quản lý doanh nghiệp', 'expense'],
  ]
  const existing = ((await (await api.get('/api/accounting/accounts')).json()) as { items: { id: number; code: string }[] }).items
  for (const [code, name, type, parent] of chart) {
    acct[code] =
      existing.find((a) => a.code === code)?.id ??
      (await created(api.post('/api/accounting/accounts', { data: { code, name, type, parent_id: parent ? acct[parent] : null, active: true } })))
  }
  await ok(api.put(`/api/accounting/starts/${c}`, { data: { start_date: from } }))

  const rules = ((await (await api.get('/api/accounting/rules')).json()) as { items: { account_id: number | null }[] }).items
  expect(rules.filter((r) => r.account_id !== null)).toHaveLength(0)

  await signedIn(page, '/accounting/accounts')
  await expect(page.getByRole('heading', { name: 'Hệ thống tài khoản' })).toBeVisible()
  await expect(page.getByText('Chi phí quản lý doanh nghiệp')).toBeVisible()
  await page.goto('/accounting/rules')
  await expect(page.getByRole('heading', { name: 'Quy tắc định khoản' })).toBeVisible()
  await expect(page.getByText('Chi phí lương')).toBeVisible()
})

test('a payroll posted before its rules exist waits for mapping, then reaches the ledger on "post pending"', async ({ page }) => {
  test.setTimeout(180_000)
  const res = await pay.post('/api/hrm/payrolls', { data: { legal_entity_id: c, month: '2026-03' } })
  expect(res.status(), await res.text()).toBe(201)
  const { id, job_id } = (await res.json()) as { id: number; job_id: number }
  await expect.poll(async () => ((await (await pay.get(`/api/jobs/${job_id}`)).json()) as { state: string }).state, { timeout: 60_000 }).toBe('completed')
  payroll = { id, number: ((await (await pay.get(`/api/hrm/payrolls/${id}`)).json()) as { number: string }).number }

  // HR is never blocked: the payroll posts after its approval with no rule at all.
  await move(pay, 'hrm.payroll', '/api/hrm/payrolls', id, 'posted')
  const dir = await signedInApi(test.info().project.use.baseURL!, login.dir, password)
  const inbox = (await (await dir.get('/api/approvals/inbox')).json()) as { items: { instance_id: number; doc_id: number; step: number }[] }
  const item = inbox.items.find((i) => i.doc_id === id)!
  await ok(dir.post(`/api/approvals/${item.instance_id}/approve`, { data: { step: item.step } }))

  const pending = async () =>
    ((await (await api.get(`/api/accounting/pending?legal_entity_id=${c}`)).json()) as { items: { doc_id: number; reason: string }[] }).items.find((d) => d.doc_id === id)
  await expect.poll(async () => (await pending())?.reason, { timeout: 60_000 }).toBe('unmapped')
  expect((await ledger('642')).lines).toHaveLength(0)

  // The accountant sets the five rules, then posts what is pending from the screen.
  for (const [kind, code, side] of [
    ['salary_expense', '642', 'debit'],
    ['employer_insurance_expense', '642', 'debit'],
    ['salary_payable', '334', 'credit'],
    ['insurance_payable', '338', 'credit'],
    ['pit_payable', '3335', 'credit'],
  ] as const) {
    await ok(api.put(`/api/accounting/rules/${kind}`, { data: { account_id: acct[code], side } }))
  }
  await signedIn(page, `/accounting/pending?legal_entity_id=${c}`)
  await expect(page.getByText(payroll.number)).toBeVisible()
  await expect(page.getByText('Chờ quy tắc định khoản')).toBeVisible()
  await page.getByRole('button', { name: 'Ghi sổ chứng từ chờ' }).click()

  // The entry is balanced, dated with the payroll, on the salary expense account, split by department.
  await expect.poll(async () => (await ledger('642')).lines.length, { timeout: 60_000 }).toBeGreaterThan(0)
  const expense = await ledger('642')
  expect(sum(expense.lines, 'debit')).toBeGreaterThan(0)
  for (const l of expense.lines) {
    expect(l.doc_number).toBe(payroll.number)
    expect(l.date.startsWith('2026-03')).toBe(true)
    expect(l.org_unit_id).toBe(c)
  }
  await expect.poll(async () => (await pending()) === undefined).toBe(true)
  const trial = (await (await api.get(`/api/accounting/trial-balance?legal_entity_id=${c}&from=${from}&to=${to}`)).json()) as { total_debit: number; total_credit: number }
  expect(trial.total_debit).toBeGreaterThan(0)
  expect(trial.total_debit).toBe(trial.total_credit)

  await page.goto(`/accounting/ledger?legal_entity_id=${c}&account_id=${acct['642']}&from=${from}&to=${to}`)
  await expect(page.getByRole('table', { name: 'Sổ cái' }).getByText(payroll.number).first()).toBeVisible()
})

test('cancelling the payroll adds a reversing entry and keeps the original', async () => {
  await move(pay, 'hrm.payroll', '/api/hrm/payrolls', payroll.id, 'cancelled')
  await expect.poll(async () => (await ledger('642')).closing, { timeout: 60_000 }).toBe(0)
  const lines = (await ledger('642')).lines
  expect(lines.length).toBeGreaterThanOrEqual(2)
  expect(sum(lines, 'debit')).toBe(sum(lines, 'credit'))
  expect(sum(lines, 'debit')).toBeGreaterThan(0)
})

test('a manual journal entry is entered, posted and cancelled; the trial balance stays equal', async ({ page }) => {
  const entry = await created(
    api.post('/api/accounting/journal-entries', {
      data: {
        request_id: crypto.randomUUID(),
        date: '2026-04-10',
        legal_entity_id: c,
        description: `Góp vốn ${run}`,
        lines: [
          { account_id: acct['112'], debit: 40_000_000, credit: 0 },
          { account_id: acct['411'], debit: 0, credit: 40_000_000 },
        ],
      },
    }),
  )
  await move(api, 'accounting.journal_entry', '/api/accounting/journal-entries', entry, 'posted')
  expect((await ledger('112')).closing).toBe(40_000_000)

  await signedIn(page, `/accounting/trial-balance?legal_entity_id=${c}&from=${from}&to=${to}`)
  await expect(page.getByText('Tổng cộng')).toBeVisible()
  const trial = (await (await api.get(`/api/accounting/trial-balance?legal_entity_id=${c}&from=${from}&to=${to}`)).json()) as { total_debit: number; total_credit: number }
  expect(trial.total_debit).toBe(trial.total_credit)

  await move(api, 'accounting.journal_entry', '/api/accounting/journal-entries', entry, 'cancelled')
  expect((await ledger('112')).closing).toBe(0)
})
