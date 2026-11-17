import { expect, test, type APIRequestContext } from '@playwright/test'
import { created, createUser, ok, password, runId, signedInApi } from './api'
import { as, signedIn, status } from './helpers'

// One run, in order: a company of its own with sales teams A and B, a goods item and a service,
// and a confirmed order of team A: 10 goods at 100 000 đ and 10 %, 2 hours of service at
// 500 000 đ and 8 %. Root plays the accountant: the chart, the five sales rules and the start
// date. The file sets mapping rules for the whole tenant, so it runs alone.
const run = runId()
const login = { a: `stc_a_${run}`, b: `stc_b_${run}` }
const iso = new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Ho_Chi_Minh' })
const from = `${iso.slice(0, 4)}-01-01`
const to = `${iso.slice(0, 4)}-12-31`
let api: APIRequestContext
let staff: APIRequestContext
let c: number
let teamA: number
let order: number
let customerName: string
const acct: Record<string, number> = {}
const inv: { id: number; number: string }[] = []
let receipt: number

const version = async (as: APIRequestContext, path: string) => ((await (await as.get(path)).json()) as { version: number }).version
const move = async (as: APIRequestContext, type: string, path: string, id: number, to: string) =>
  as.post(`/api/documents/${type}/${id}/transitions`, { data: { to, version: await version(as, `${path}/${id}`) } })
const code = async (res: Promise<{ json(): Promise<unknown> }>) => ((await (await res).json()) as { code?: string }).code

type Ledger = { lines: { debit: number; credit: number; doc_number: string; partner_id: number | null }[] }
const ledger = async (account: string) =>
  (await (await api.get(`/api/accounting/ledger?legal_entity_id=${c}&account_id=${acct[account]}&from=${from}&to=${to}`)).json()) as Ledger
const net = async (account: string, number: string) =>
  (await ledger(account)).lines.filter((l) => l.doc_number === number).reduce((n, l) => n + l.debit - l.credit, 0)

test.beforeAll(async ({ baseURL }) => {
  test.setTimeout(120_000)
  api = await signedInApi(baseURL!)
  c = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty TT ${run}`, legal_name: `Công ty TNHH TT ${run}` } }))
  teamA = await created(api.post('/api/org-units', { data: { parent_id: c, kind: 'department', name: `Kinh doanh TT A ${run}` } }))
  const teamB = await created(api.post('/api/org-units', { data: { parent_id: c, kind: 'department', name: `Kinh doanh TT B ${run}` } }))
  await createUser(api, login.a, [
    { role: 'sales.staff', unit: teamA },
    { role: 'partner.editor', unit: teamA },
  ])
  await createUser(api, login.b, [{ role: 'sales.staff', unit: teamB }])
  const me = ((await (await api.get('/api/users')).json()) as { id: number; login: string }[]).find((u) => u.login === 'root')
  if (me) await api.post(`/api/users/${me.id}/roles`, { data: { product: 'accounting', role: 'admin', org_unit_id: null } })

  // Accounts other files made are reused by code.
  const existing = ((await (await api.get('/api/accounting/accounts?page_size=500')).json()) as { items: { id: number; code: string }[] }).items
  const chart: [string, string, string, string?, boolean?][] = [
    ['111', 'Tiền mặt', 'asset'],
    ['112', 'Tiền gửi ngân hàng', 'asset'],
    ['131', 'Phải thu của khách hàng', 'asset', undefined, true],
    ['333', 'Thuế và các khoản phải nộp Nhà nước', 'liability'],
    ['3331', 'Thuế giá trị gia tăng phải nộp', 'liability', '333'],
    ['511', 'Doanh thu bán hàng và cung cấp dịch vụ', 'revenue'],
  ]
  for (const [code, name, type, parent, byPartner] of chart) {
    acct[code] =
      existing.find((a) => a.code === code)?.id ??
      (await created(api.post('/api/accounting/accounts', { data: { code, name, type, parent_id: parent ? acct[parent] : null, active: true, by_partner: !!byPartner } })))
  }
  for (const [kind, account, side] of [
    ['sales_receivable', '131', 'debit'],
    ['sales_revenue', '511', 'credit'],
    ['sales_output_vat', '3331', 'credit'],
    ['sales_cash_receipt', '111', 'debit'],
    ['sales_bank_receipt', '112', 'debit'],
  ] as const) {
    await ok(api.put(`/api/accounting/rules/${kind}`, { data: { account_id: acct[account], side } }))
  }
  await ok(api.put(`/api/accounting/starts/${c}`, { data: { start_date: from } }))

  const goods = await created(api.post('/api/items', { data: { code: `TTH${run}`, name: `Máy lọc nước ${run}`, unit: 'cái', price: 100_000, vat_rate: '10', active: true } }))
  const service = await created(api.post('/api/items', { data: { code: `TTD${run}`, name: `Lắp đặt ${run}`, unit: 'giờ', price: 500_000, vat_rate: '8', active: true } }))
  customerName = `Khách sạn Biển ${run}`
  const customer = await created(
    api.post('/api/sales/customers', {
      data: {
        code: `KHTT${run}`,
        partner: { name: customerName, tax_code: null, address: null, phone: null, email: null, contact_name: null },
        payment_terms: null,
        org_unit_id: teamA,
        active: true,
      },
    }),
  )
  staff = await signedInApi(baseURL!, login.a, password)
  order = await created(
    staff.post('/api/sales/orders', {
      data: {
        request_id: crypto.randomUUID(),
        date: iso,
        org_unit_id: teamA,
        customer_id: customer,
        valid_until: null,
        delivery_date: null,
        payment_terms: null,
        delivery_terms: null,
        note: null,
        lines: [
          { item_id: goods, description: `Máy lọc nước ${run}`, quantity: '10', unit_price: 100_000, discount_percent: '0', vat_rate: '10' },
          { item_id: service, description: `Lắp đặt ${run}`, quantity: '2', unit_price: 500_000, discount_percent: '0', vat_rate: '8' },
        ],
      },
    }),
  )
  await ok(move(staff, 'sales.order', '/api/sales/orders', order, 'posted'))
})

test('the order is invoiced in two parts; a third invoice beyond what is left is refused', async ({ browser }) => {
  const page = await as(browser, login.a, `/sales/orders/${order}`)
  await page.getByRole('link', { name: 'Lập hoá đơn' }).click()
  // Pre-filled with all that is left; this invoice takes part of each line.
  await expect(page.getByRole('textbox', { name: 'Số lượng (1)' })).toHaveValue('10')
  await page.getByRole('textbox', { name: 'Số lượng (1)' }).fill('4')
  await page.getByRole('textbox', { name: 'Số lượng (2)' }).fill('1')
  await page.getByRole('button', { name: 'Tạo bản nháp' }).click()
  const heading = page.getByRole('heading', { name: /^HD-\d{4}-/ })
  await expect(heading).toBeVisible()
  await page.getByRole('button', { name: 'Gửi duyệt' }).click()
  await status(page, 'Đã ghi sổ')
  // 4 × 100 000 + 10 % and 1 × 500 000 + 8 %: 400 000 + 40 000 + 500 000 + 40 000.
  await expect(page.getByText('980.000').first()).toBeVisible()
  inv.push({ id: Number(page.url().split('/').pop()), number: (await heading.textContent())!.trim() })

  await page.goto(`/sales/orders/${order}`)
  await expect(page.getByText('Xuất hoá đơn một phần').first()).toBeVisible()

  const second = await created(
    staff.post('/api/sales/invoices', {
      data: { request_id: crypto.randomUUID(), order_id: order, date: iso, note: null, lines: [{ order_position: 1, quantity: '6' }, { order_position: 2, quantity: '1' }] },
    }),
  )
  await ok(move(staff, 'sales.invoice', '/api/sales/invoices', second, 'posted'))
  inv.push({ id: second, number: ((await (await staff.get(`/api/sales/invoices/${second}`)).json()) as { number: string }).number })
  const third = staff.post('/api/sales/invoices', {
    data: { request_id: crypto.randomUUID(), order_id: order, date: iso, note: null, lines: [{ order_position: 1, quantity: '1' }] },
  })
  expect(await code(third)).toBe('sales_invoice_over_ordered')

  await page.reload()
  await expect(page.getByText('Đã xuất hoá đơn đủ').first()).toBeVisible()
  // The order list filters by invoicing.
  await page.goto(`/sales/orders?invoice_state=full&q=${encodeURIComponent(customerName)}`)
  await expect(page.getByRole('link', { name: /^DH-\d{4}-/ })).toHaveCount(1)
})

test('a posted invoice reaches the ledger: receivable with the customer, revenue and output VAT', async () => {
  await expect.poll(() => net('131', inv[0]!.number), { timeout: 60_000 }).toBe(980_000)
  expect(await net('511', inv[0]!.number)).toBe(-900_000)
  expect(await net('3331', inv[0]!.number)).toBe(-80_000)
  expect((await ledger('131')).lines.find((l) => l.doc_number === inv[0]!.number)?.partner_id).not.toBeNull()
  // 6 × 100 000 + 10 % and 1 × 500 000 + 8 %: 1 200 000.
  await expect.poll(() => net('131', inv[1]!.number), { timeout: 60_000 }).toBe(1_200_000)
  const balances = (await (await api.get(`/api/accounting/partner-balances?legal_entity_id=${c}&account_id=${acct['131']}&from=${from}&to=${to}`)).json()) as {
    rows: { partner_name: string; closing: number }[]
  }
  expect(balances.rows.find((r) => r.partner_name === customerName)?.closing).toBe(2_180_000)
})

test('one receipt settles the first invoice and part of the second; payment shows on invoices and the order', async ({ browser }) => {
  const page = await as(browser, login.a, `/sales/invoices/${inv[0]!.id}`)
  await page.getByRole('link', { name: 'Thu tiền' }).click()
  await expect(page.getByRole('textbox', { name: 'Số thu (1)' })).toHaveValue(/980/)
  await page.getByRole('button', { name: 'Thêm hoá đơn' }).click()
  await page.getByRole('combobox', { name: 'Hoá đơn (2)' }).click()
  await page.getByRole('option', { name: new RegExp(`^${inv[1]!.number}`) }).click()
  await page.getByRole('textbox', { name: 'Số thu (2)' }).fill('200000')
  await page.getByRole('button', { name: 'Tạo bản nháp' }).click()
  await expect(page.getByRole('heading', { name: /^PT-\d{4}-/ })).toBeVisible()
  receipt = Number(page.url().split('/').pop())
  await page.getByRole('button', { name: 'Gửi duyệt' }).click()
  await status(page, 'Đã ghi sổ')

  await page.goto(`/sales/invoices/${inv[0]!.id}`)
  await expect(page.getByText('Đã thu đủ').first()).toBeVisible()
  await page.goto(`/sales/invoices/${inv[1]!.id}`)
  await expect(page.getByText('Thu một phần').first()).toBeVisible()
  await page.goto(`/sales/orders/${order}`)
  await expect(page.getByText('Thu một phần').first()).toBeVisible()
  // The order shows what was received of it, and each line what was invoiced.
  await expect(page.getByText('Đã thu 1.180.000 / 2.180.000 đ')).toBeVisible()
  await expect(page.getByRole('columnheader', { name: 'Đã xuất hoá đơn' })).toBeVisible()

  const number = ((await (await staff.get(`/api/sales/receipts/${receipt}`)).json()) as { number: string }).number
  await expect.poll(() => net('112', number), { timeout: 60_000 }).toBe(1_180_000)
  expect(await net('131', number)).toBe(-1_180_000)
})

test('over-allocation and cancelling out of order are refused; cancelling backwards reverses the lines', async () => {
  const over = staff.post('/api/sales/receipts', {
    data: { request_id: crypto.randomUUID(), date: iso, org_unit_id: teamA, customer_id: await customerOf(), method: 'cash', note: null, allocations: [{ invoice_id: inv[1]!.id, amount: 1_000_001 }] },
  })
  expect(await code(over)).toBe('sales_receipt_over_invoice')
  expect(await code(move(staff, 'sales.invoice', '/api/sales/invoices', inv[0]!.id, 'cancelled'))).toBe('sales_invoice_has_receipt')
  await ok(move(staff, 'sales.receipt', '/api/sales/receipts', receipt, 'cancelled'))
  await ok(move(staff, 'sales.invoice', '/api/sales/invoices', inv[0]!.id, 'cancelled'))
  await expect.poll(() => net('131', inv[0]!.number), { timeout: 60_000 }).toBe(0)
  expect(await net('511', inv[0]!.number)).toBe(0)
})

test("another team's staff sees no invoice or receipt", async () => {
  const b = await signedInApi(test.info().project.use.baseURL!, login.b, password)
  expect(((await (await b.get(`/api/sales/invoices?q=${inv[1]!.number}`)).json()) as { total: number }).total).toBe(0)
  expect(((await (await b.get('/api/sales/receipts')).json()) as { items: { id: number }[] }).items.some((r) => r.id === receipt)).toBe(false)
  expect((await b.get(`/api/sales/invoices/${inv[1]!.id}`)).status()).toBe(404)
  expect((await b.get(`/api/sales/receipts/${receipt}`)).status()).toBe(404)
})

test('with Sales disabled, an invoice still reads, with no write action', async ({ page }) => {
  await signedIn(page, `http://localhost:8091/sales/invoices/${inv[1]!.id}`)
  await expect(page.getByRole('heading', { name: inv[1]!.number })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Thu tiền' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Hủy chứng từ' })).toHaveCount(0)
})

async function customerOf() {
  return ((await (await staff.get(`/api/sales/orders/${order}`)).json()) as { customer: { id: number } }).customer.id
}
