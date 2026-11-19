import { expect, test, type APIRequestContext } from '@playwright/test'
import { created, createUser, ok, password, runId, signedInApi } from './api'
import { as } from './helpers'

// One run, in order: a company of its own with purchasing teams A and B, a lot item, a serial item
// and a service, a partner that is a Sales customer and becomes a supplier of team A, and an order
// of 10 cartons at 50 000 đ (8 %), 2 purifiers at 3 000 000 đ (10 %) and one transport at
// 500 000 đ (10 %). Root plays the accountant: the chart, the six purchasing rules and the start
// date. The file sets mapping rules for the whole tenant, so it runs alone.
const run = runId()
const login = { a: `pur_a_${run}`, b: `pur_b_${run}` }
const iso = new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Ho_Chi_Minh' })
const from = `${iso.slice(0, 4)}-01-01`
const to = `${iso.slice(0, 4)}-12-31`
let api: APIRequestContext
let staff: APIRequestContext
let c: number
let supplier: number
let partner: number
let order: number
let warehouse: number
const acct: Record<string, number> = {}
const inv: { id: number; number: string; total: number }[] = []

const version = async (as: APIRequestContext, path: string) => ((await (await as.get(path)).json()) as { version: number }).version
const move = async (as: APIRequestContext, type: string, path: string, id: number, to: string) =>
  as.post(`/api/documents/${type}/${id}/transitions`, { data: { to, version: await version(as, `${path}/${id}`) } })
const code = async (res: Promise<{ json(): Promise<unknown> }>) => ((await (await res).json()) as { code?: string }).code
type OrderView = { receipt_state?: string; invoice_state?: string; payment_state?: string; paid?: number }
const orderView = async () => (await (await staff.get(`/api/purchase/orders/${order}`)).json()) as OrderView

type Ledger = { lines: { debit: number; credit: number; doc_number: string; partner_id: number | null }[] }
const net = async (account: string, number: string) =>
  (
    (await (await api.get(`/api/accounting/ledger?legal_entity_id=${c}&account_id=${acct[account]}&from=${from}&to=${to}`)).json()) as Ledger
  ).lines
    .filter((l) => l.doc_number === number)
    .reduce((n, l) => n + l.debit - l.credit, 0)

const receive = (lines: unknown[]) =>
  staff.post('/api/purchase/goods-receipts', { data: { request_id: crypto.randomUUID(), order_id: order, date: iso, warehouse_id: warehouse, note: null, lines } })
const bill = (no: string, lines: unknown[]) =>
  staff.post('/api/purchase/invoices', {
    data: { request_id: crypto.randomUUID(), order_id: order, date: iso, supplier_invoice_symbol: '1C26TAA', supplier_invoice_no: no, supplier_invoice_date: iso, note: null, lines },
  })
const line = (pos: number, qty: string, price: number, vat: string) => ({ order_position: pos, quantity: qty, unit_price: price, discount_percent: '0', vat_rate: vat })

test.beforeAll(async ({ baseURL }) => {
  test.setTimeout(120_000)
  api = await signedInApi(baseURL!)
  c = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty MH ${run}`, legal_name: `Công ty TNHH MH ${run}` } }))
  const teamA = await created(api.post('/api/org-units', { data: { parent_id: c, kind: 'department', name: `Mua hàng A ${run}` } }))
  const teamB = await created(api.post('/api/org-units', { data: { parent_id: c, kind: 'department', name: `Mua hàng B ${run}` } }))
  await createUser(api, login.a, [
    { role: 'purchase.staff', unit: teamA },
    { role: 'partner.viewer', unit: teamA },
  ])
  await createUser(api, login.b, [{ role: 'purchase.staff', unit: teamB }])
  const me = ((await (await api.get('/api/users')).json()) as { id: number; login: string }[]).find((u) => u.login === 'root')
  if (me) await api.post(`/api/users/${me.id}/roles`, { data: { product: 'accounting', role: 'admin', org_unit_id: null } })

  // Accounts other files made are reused by code.
  const existing = ((await (await api.get('/api/accounting/accounts?page_size=500')).json()) as { items: { id: number; code: string }[] }).items
  const chart: [string, string, string, string?, boolean?][] = [
    ['111', 'Tiền mặt', 'asset'],
    ['112', 'Tiền gửi ngân hàng', 'asset'],
    ['133', 'Thuế GTGT được khấu trừ', 'asset'],
    ['1331', 'Thuế GTGT được khấu trừ của hàng hóa, dịch vụ', 'asset', '133'],
    ['156', 'Hàng hóa', 'asset'],
    ['331', 'Phải trả cho người bán', 'liability', undefined, true],
    ['642', 'Chi phí quản lý doanh nghiệp', 'expense'],
  ]
  for (const [code, name, type, parent, byPartner] of chart) {
    acct[code] =
      existing.find((a) => a.code === code)?.id ??
      (await created(api.post('/api/accounting/accounts', { data: { code, name, type, parent_id: parent ? acct[parent] : null, active: true, by_partner: !!byPartner } })))
  }
  for (const [kind, account, side] of [
    ['purchase_goods', '156', 'debit'],
    ['purchase_expense', '642', 'debit'],
    ['purchase_input_vat', '1331', 'debit'],
    ['purchase_payable', '331', 'credit'],
    ['purchase_cash_payment', '111', 'credit'],
    ['purchase_bank_payment', '112', 'credit'],
  ] as const) {
    await ok(api.put(`/api/accounting/rules/${kind}`, { data: { account_id: acct[account], side } }))
  }
  await ok(api.put(`/api/accounting/starts/${c}`, { data: { start_date: from } }))

  const item = (code: string, name: string, unit: string, price: number, vat: string, tracking?: string) =>
    created(api.post('/api/items', { data: { code: `${code}${run}`, name: `${name} ${run}`, unit, price, vat_rate: vat, active: true, ...(tracking ? { tracking } : {}) } }))
  const lotItem = await item('MHL', 'Sữa hộp', 'thùng', 50_000, '8', 'lot')
  const serialItem = await item('MHS', 'Máy lọc nước', 'cái', 3_000_000, '10', 'serial')
  const service = await item('MHD', 'Vận chuyển', 'chuyến', 500_000, '10')

  // Exit criterion 1: a partner that is already a customer becomes a supplier.
  partner = await created(api.post('/api/partners', { data: { name: `Công ty Hai Vai ${run}`, tax_code: null, address: null, phone: null, email: null, contact_name: null } }))
  await created(api.post('/api/sales/customers', { data: { code: `KHMH${run}`, partner_id: partner, payment_terms: null, org_unit_id: teamA, active: true } }))
  supplier = await created(api.post('/api/purchase/suppliers', { data: { partner_id: partner, payment_terms: null, org_unit_id: teamA, active: true } }))

  staff = await signedInApi(baseURL!, login.a, password)
  order = await created(
    staff.post('/api/purchase/orders', {
      data: {
        request_id: crypto.randomUUID(),
        date: iso,
        org_unit_id: teamA,
        supplier_id: supplier,
        expected_date: null,
        payment_terms: null,
        note: null,
        lines: [
          { item_id: lotItem, description: `Sữa hộp ${run}`, quantity: '10', unit_price: 50_000, discount_percent: '0', vat_rate: '8' },
          { item_id: serialItem, description: `Máy lọc nước ${run}`, quantity: '2', unit_price: 3_000_000, discount_percent: '0', vat_rate: '10' },
          { item_id: service, description: `Vận chuyển ${run}`, quantity: '1', unit_price: 500_000, discount_percent: '0', vat_rate: '10' },
        ],
      },
    }),
  )
  await ok(move(staff, 'purchase.order', '/api/purchase/orders', order, 'posted'))
  warehouse = ((await (await staff.get(`/api/purchase/orders/${order}/receipt-options`)).json()) as { warehouses: { id: number }[] }).warehouses[0]!.id
})

test('two goods receipts bring the stock lines in by lot and serial; a third beyond the order is refused', async ({ browser }) => {
  const first = await created(receive([{ order_position: 1, quantity: '6', lot: { code: `L1${run}`, expiry: to } }, { order_position: 2, quantity: '1', serial: `S1${run}` }]))
  await ok(move(staff, 'purchase.goods_receipt', '/api/purchase/goods-receipts', first, 'posted'))
  expect((await orderView()).receipt_state).toBe('partial')

  const page = await as(browser, login.a, `/purchase/orders/${order}`)
  await expect(page.getByText('Nhận một phần')).toBeVisible()

  expect(await code(receive([{ order_position: 1, quantity: '5', lot: { code: `L2${run}` } }]))).toBe('purchase_receipt_over_order')
  const second = await created(receive([{ order_position: 1, quantity: '4', lot: { code: `L2${run}` } }, { order_position: 2, quantity: '1', serial: `S2${run}` }]))
  await ok(move(staff, 'purchase.goods_receipt', '/api/purchase/goods-receipts', second, 'posted'))
  expect((await orderView()).receipt_state).toBe('full')
})

test('a repriced supplier invoice books goods, expense and input VAT against the payable; its paper is booked once', async () => {
  // 4 cartons at 52 000 (ordered at 50 000), both purifiers, the transport.
  const id = await created(bill('0000123', [line(1, '4', 52_000, '8'), line(2, '2', 3_000_000, '10'), line(3, '1', 500_000, '10')]))
  await ok(move(staff, 'purchase.invoice', '/api/purchase/invoices', id, 'posted'))
  const v = (await (await staff.get(`/api/purchase/invoices/${id}`)).json()) as { number: string; totals: { total: number } }
  expect(v.totals.total).toBe(7_374_640)
  inv.push({ id, number: v.number, total: v.totals.total })
  expect((await orderView()).invoice_state).toBe('partial')

  // The ledger gets it once the accounting job has run.
  await expect.poll(() => net('331', v.number), { timeout: 30_000 }).toBe(-7_374_640)
  expect(await net('156', v.number)).toBe(6_208_000)
  expect(await net('642', v.number)).toBe(500_000)
  expect(await net('1331', v.number)).toBe(666_640)

  const dup = await created(bill('0000123', [line(1, '1', 50_000, '8')]))
  expect(await code(move(staff, 'purchase.invoice', '/api/purchase/invoices', dup, 'posted'))).toBe('purchase_invoice_duplicate')
  // The refused draft still holds its carton; deleting it frees the quantity.
  await ok(staff.delete(`/api/documents/purchase.invoice/${dup}?version=${await version(staff, `/api/purchase/invoices/${dup}`)}`))
  const rest = await created(bill('0000124', [line(1, '6', 50_000, '8')]))
  await ok(move(staff, 'purchase.invoice', '/api/purchase/invoices', rest, 'posted'))
  const r = (await (await staff.get(`/api/purchase/invoices/${rest}`)).json()) as { number: string; totals: { total: number } }
  inv.push({ id: rest, number: r.number, total: r.totals.total })
  expect((await orderView()).invoice_state).toBe('full')
})

test('one payment settles one invoice and part of the other; over-allocating and cancelling out of order are refused', async () => {
  const [a, b] = inv as [(typeof inv)[number], (typeof inv)[number]]
  const unit = ((await (await staff.get(`/api/purchase/invoices/${a.id}`)).json()) as { org_unit_id: number }).org_unit_id
  const body = (allocations: unknown[]) => ({ request_id: crypto.randomUUID(), date: iso, org_unit_id: unit, supplier_id: supplier, method: 'bank', note: null, allocations })
  expect(await code(staff.post('/api/purchase/payments', { data: body([{ invoice_id: a.id, amount: a.total + 1 }]) }))).toBe('purchase_payment_over_invoice')
  const id = await created(staff.post('/api/purchase/payments', { data: body([{ invoice_id: a.id, amount: a.total }, { invoice_id: b.id, amount: 1_000 }]) }))
  await ok(move(staff, 'purchase.payment', '/api/purchase/payments', id, 'posted'))
  const state = async (i: number) => ((await (await staff.get(`/api/purchase/invoices/${i}`)).json()) as { payment_state: string }).payment_state
  expect(await state(a.id)).toBe('paid')
  expect(await state(b.id)).toBe('partial')
  expect(await orderView()).toMatchObject({ payment_state: 'partial', paid: a.total + 1_000 })

  expect(await code(move(staff, 'purchase.invoice', '/api/purchase/invoices', a.id, 'cancelled'))).toBe('purchase_invoice_has_payment')
  await ok(move(staff, 'purchase.payment', '/api/purchase/payments', id, 'cancelled'))
  expect(await state(a.id)).toBe('unpaid')
})

test('another team sees none of the documents', async ({ baseURL }) => {
  const other = await signedInApi(baseURL!, login.b, password)
  expect((await other.get(`/api/purchase/orders/${order}`)).status()).toBe(404)
  expect((await other.get(`/api/purchase/invoices/${inv[0]!.id}`)).status()).toBe(404)
  const list = (await (await other.get('/api/purchase/orders')).json()) as { total: number }
  expect(list.total).toBe(0)
})
