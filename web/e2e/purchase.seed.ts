import { expect, test, type APIRequestContext, type APIResponse } from '@playwright/test'
import { created, signedInApi } from './api'
import { admin as root } from './helpers'

// Purchasing on top of the Demo demo company (demo.seed.ts and sales.seed.ts run first): two
// suppliers, one of them a partner that is already a Sales customer; an order of rice by lot,
// laptops by serial and transport, received in part, billed in two invoices (one repriced) and
// paid in part. Root does the work with a tenant-wide Purchasing role. Skipped once orders exist.

async function ok(res: Promise<APIResponse>, what: string) {
  const r = await res
  expect(r.ok(), `${what}: ${r.status()} ${await r.text()}`).toBe(true)
  return r
}
const get = async <T>(api: APIRequestContext, path: string): Promise<T> => (await (await ok(api.get(path), path)).json()) as T
const post = async (api: APIRequestContext, type: string, path: string, id: number) => {
  const { version } = await get<{ version: number }>(api, `${path}/${id}`)
  await ok(api.post(`/api/documents/${type}/${id}/transitions`, { data: { to: 'posted', version } }), `post ${type} ${id}`)
}

// A date months after today in the tenant time zone, so the seeded lot stays unexpired.
const monthsAhead = (months: number) => {
  const d = new Date(new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Ho_Chi_Minh' }))
  d.setUTCMonth(d.getUTCMonth() + months)
  return d.toISOString().slice(0, 10)
}

test('seed Purchasing on the Demo demo company', async ({ baseURL }) => {
  test.setTimeout(120_000)
  const api = await signedInApi(baseURL!)
  if ((await get<{ total: number }>(api, '/api/purchase/orders?page_size=20')).total > 0) test.skip(true, 'Purchasing data already there')

  const users = await get<{ id: number; login: string }[]>(api, '/api/users')
  const me = users.find((u) => u.login === root.login)!.id
  for (const product of ['purchase', 'partner']) {
    const role = await api.post(`/api/users/${me}/roles`, { data: { product, role: product === 'purchase' ? 'staff' : 'editor', org_unit_id: null } })
    expect([201, 409], await role.text()).toContain(role.status())
  }

  const units = await get<{ id: number; name: string; kind: string }[]>(api, '/api/org-units')
  const demo = units.find((u) => u.kind === 'company' && u.name === 'Công ty CP Demo')!.id
  const today = new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Ho_Chi_Minh' })

  // A customer's partner also sells to us; a second supplier is new.
  const partners = (await get<{ items: { id: number; name: string }[] }>(api, '/api/partners?page_size=20&sort=code')).items
  const both = await created(api.post('/api/purchase/suppliers', { data: { partner_id: partners[0]!.id, payment_terms: 'Thanh toán trong 15 ngày', org_unit_id: demo, active: true } }))
  const rice = await created(
    api.post('/api/purchase/suppliers', {
      data: {
        partner: { name: 'Công ty TNHH Lương thực Miền Bắc', tax_code: '0108888888', address: '8 Giải Phóng, Hà Nội', phone: '024 3999 8888', email: null, contact_name: 'Chị Lan' },
        payment_terms: 'Thanh toán trong 30 ngày',
        org_unit_id: demo,
        active: true,
      },
    }),
  )

  const item = async (code: string) =>
    (await get<{ items: { id: number; code: string; name: string }[] }>(api, `/api/purchase/line-items?q=${code}`)).items.find((i) => i.code === code)!
  const [gao, lap, vc] = [await item('GAO-5KG'), await item('LAP-14'), await item('DV-VC')]

  const order = await created(
    api.post('/api/purchase/orders', {
      data: {
        request_id: crypto.randomUUID(),
        date: today,
        org_unit_id: demo,
        supplier_id: rice,
        expected_date: null,
        payment_terms: 'Thanh toán trong 30 ngày',
        note: null,
        lines: [
          { item_id: gao.id, description: gao.name, quantity: '50', unit_price: 190_000, discount_percent: '0', vat_rate: '5' },
          { item_id: lap.id, description: lap.name, quantity: '3', unit_price: 22_000_000, discount_percent: '0', vat_rate: '10' },
          { item_id: vc.id, description: vc.name, quantity: '1', unit_price: 350_000, discount_percent: '0', vat_rate: '8' },
        ],
      },
    }),
  )
  await post(api, 'purchase.order', '/api/purchase/orders', order)
  // A draft order of the other supplier, so its page has something to show.
  await created(
    api.post('/api/purchase/orders', {
      data: {
        request_id: crypto.randomUUID(),
        date: today,
        org_unit_id: demo,
        supplier_id: both,
        expected_date: null,
        payment_terms: null,
        note: null,
        lines: [{ item_id: vc.id, description: vc.name, quantity: '2', unit_price: 350_000, discount_percent: '0', vat_rate: '8' }],
      },
    }),
  )

  const options = await get<{ warehouses: { id: number }[] }>(api, `/api/purchase/orders/${order}/receipt-options`)
  const receipt = await created(
    api.post('/api/purchase/goods-receipts', {
      data: {
        request_id: crypto.randomUUID(),
        order_id: order,
        date: today,
        warehouse_id: options.warehouses[0]!.id,
        note: null,
        lines: [
          { order_position: 1, quantity: '30', lot: { code: 'G2712', expiry: monthsAhead(14) } },
          { order_position: 2, quantity: '1', serial: 'SN-MB-001' },
          { order_position: 2, quantity: '1', serial: 'SN-MB-002' },
        ],
      },
    }),
  )
  await post(api, 'purchase.goods_receipt', '/api/purchase/goods-receipts', receipt)

  const invoice = (no: string, lines: unknown[]) =>
    created(
      api.post('/api/purchase/invoices', {
        data: {
          request_id: crypto.randomUUID(),
          order_id: order,
          date: today,
          supplier_invoice_symbol: '1C26TLM',
          supplier_invoice_no: no,
          supplier_invoice_date: today,
          note: null,
          lines,
        },
      }),
    )
  // The supplier billed the rice at 195,000 đ instead of the 190,000 ordered.
  const first = await invoice('0000451', [
    { order_position: 1, quantity: '30', unit_price: 195_000, discount_percent: '0', vat_rate: '5' },
    { order_position: 2, quantity: '2', unit_price: 22_000_000, discount_percent: '0', vat_rate: '10' },
    { order_position: 3, quantity: '1', unit_price: 350_000, discount_percent: '0', vat_rate: '8' },
  ])
  const second = await invoice('0000452', [{ order_position: 1, quantity: '20', unit_price: 190_000, discount_percent: '0', vat_rate: '5' }])
  await post(api, 'purchase.invoice', '/api/purchase/invoices', first)
  await post(api, 'purchase.invoice', '/api/purchase/invoices', second)

  // One payment settles the first invoice and part of the second.
  const total = (await get<{ totals: { total: number } }>(api, `/api/purchase/invoices/${first}`)).totals.total
  const payment = await created(
    api.post('/api/purchase/payments', {
      data: {
        request_id: crypto.randomUUID(),
        date: today,
        org_unit_id: demo,
        supplier_id: rice,
        method: 'bank',
        note: null,
        allocations: [
          { invoice_id: first, amount: total },
          { invoice_id: second, amount: 1_000_000 },
        ],
      },
    }),
  )
  await post(api, 'purchase.payment', '/api/purchase/payments', payment)
})
