import { expect, test } from '@playwright/test'
import { created, createUser, ok, password, runId, signedInApi } from './api'
import { as, status } from './helpers'

// One run: a company with a sales department and the main warehouse it comes with, a lot-tracked
// item, and a confirmed order for four of it. One account sells and keeps the stock.
const run = runId()
const login = `dlv_${run}`
const code = `GAO${run}`
const lot = `L${run}`
let warehouse: string
let orderId: number

// Today in the tenant time zone, as the API and as the date fields take it.
const iso = new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Ho_Chi_Minh' })
const typed = (d: string) => d.split('-').reverse().join('/')

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(
    api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty GH ${run}`, legal_name: `Công ty TNHH GH ${run}` } }),
  )
  const unit = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Kinh doanh GH ${run}` } }))
  await createUser(api, login, [
    { role: 'sales.staff', unit },
    { role: 'partner.editor', unit },
    { role: 'stock.admin', unit: null },
  ])
  // The company's default warehouse: the delivery form starts from it.
  const { items } = (await (await api.get('/api/stock/warehouses?page_size=100')).json()) as {
    items: { code: string; name: string; legal_entity_id: number; is_default: boolean }[]
  }
  const main = items.find((w) => w.legal_entity_id === company && w.is_default)!
  warehouse = `${main.code} · ${main.name}`
  const item = await created(
    api.post('/api/items', { data: { code, name: `Gạo thơm ${run}`, unit: 'túi', price: 210_000, vat_rate: '5', active: true, tracking: 'lot' } }),
  )
  const customer = await created(
    api.post('/api/sales/customers', {
      data: {
        code: `KHGH${run}`,
        partner: { name: `Nhà hàng Sen ${run}`, tax_code: null, address: null, phone: null, email: null, contact_name: null },
        payment_terms: null,
        org_unit_id: unit,
        active: true,
      },
    }),
  )
  const staff = await signedInApi(baseURL!, login, password)
  orderId = await created(
    staff.post('/api/sales/orders', {
      data: {
        request_id: crypto.randomUUID(),
        date: iso,
        org_unit_id: unit,
        customer_id: customer,
        valid_until: null,
        delivery_date: null,
        payment_terms: null,
        delivery_terms: null,
        note: null,
        lines: [{ item_id: item, description: `Gạo thơm ${run}`, quantity: '4', unit_price: 210_000, discount_percent: '0', vat_rate: '5' }],
      },
    }),
  )
  const { version } = (await (await staff.get(`/api/sales/orders/${orderId}`)).json()) as { version: number }
  await ok(staff.post(`/api/documents/sales.order/${orderId}/transitions`, { data: { to: 'posted', version } }))
})

test('opening stock in a new lot, delivered from the order, leaves the rest on hand and the order delivered', async ({ browser }) => {
  // Opening stock: ten bags in a new lot, by an adjustment.
  const page = await as(browser, login, '/sales/stock/adjustments')
  await page.getByRole('link', { name: 'Lập phiếu điều chỉnh' }).click()
  await page.getByRole('textbox', { name: 'Ngày điều chỉnh' }).fill(typed(iso))
  await page.getByRole('combobox', { name: 'Kho', exact: true }).click()
  await page.getByRole('option', { name: warehouse }).click()
  await page.getByRole('textbox', { name: 'Lý do' }).fill('Tồn đầu kỳ')
  await page.getByRole('combobox', { name: 'Mặt hàng (1)' }).fill(code)
  await page.getByRole('option', { name: new RegExp(code) }).click()
  await page.getByRole('textbox', { name: 'Lô hàng (1)' }).fill(lot)
  await page.getByRole('textbox', { name: 'Hạn sử dụng (1)' }).fill(`31/12/${new Date().getFullYear() + 1}`)
  await page.getByRole('textbox', { name: 'Số lượng (1)' }).fill('10')
  await page.getByRole('button', { name: 'Tạo bản nháp' }).click()
  await expect(page.getByRole('heading', { name: /^DC-\d{4}-/ })).toBeVisible()
  await page.getByRole('button', { name: 'Gửi duyệt' }).click()
  await status(page, 'Đã ghi sổ')

  const onHand = async (quantity: string) => {
    await page.goto(`/sales/stock/on-hand?q=${code}`)
    const row = page.getByRole('table', { name: 'Tồn kho' }).getByRole('row').filter({ hasText: lot })
    await expect(row.getByRole('cell', { name: quantity, exact: true })).toBeVisible()
  }
  await onHand('10')

  // The order offers a delivery; its line comes with the lot already picked.
  await page.goto(`/sales/orders/${orderId}`)
  await page.getByRole('link', { name: 'Tạo phiếu giao hàng' }).click()
  await expect(page.getByRole('combobox', { name: 'Lô hàng (1)' })).toHaveValue(new RegExp(`^${lot}`))
  await expect(page.getByRole('textbox', { name: 'Số lượng (1)' })).toHaveValue('4')
  await page.getByRole('button', { name: 'Tạo bản nháp' }).click()
  await expect(page.getByRole('heading', { name: /^GH-\d{4}-/ })).toBeVisible()
  await page.getByRole('button', { name: 'Gửi duyệt' }).click()
  await status(page, 'Đã ghi sổ')

  await onHand('6')

  // Fully delivered; cancelling the order is refused while the delivery stands.
  await page.goto(`/sales/orders/${orderId}`)
  await expect(page.getByText('Đã giao đủ').first()).toBeVisible()
  await page.getByRole('button', { name: 'Hủy chứng từ' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Hủy chứng từ' }).click()
  await expect(page.getByText('Đơn bán hàng đã có phiếu giao hàng. Hủy phiếu giao hàng trước.')).toBeVisible()
  await page.reload()
  await status(page, 'Đã xác nhận')
})
