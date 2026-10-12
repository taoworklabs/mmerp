import { expect, test, type APIRequestContext } from '@playwright/test'
import { created, employee, signedInApi } from './api'
import { signedIn } from './helpers'

// Guards that keep a user's work or a failure from vanishing without a word: unsaved edits
// before submitting, an invalid timesheet cell, a delete the server refused.
const run = Date.now().toString(36)
let api: APIRequestContext
let company: number
let unit: number

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty G ${run}`, legal_name: `Công ty TNHH G ${run}` } }))
  unit = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng G ${run}` } }))
})

test('submitting a quotation with unsaved edits asks to save first and keeps the edits', async ({ page }) => {
  const item = await created(api.post('/api/sales/items', { data: { code: `G${run}`, name: `Giấy A4 ${run}`, unit: 'ram', price: 70000, vat_rate: '10', active: true } }))
  const customer = await created(
    api.post('/api/sales/customers', {
      data: { code: `KG${run}`, name: `Khách G ${run}`, tax_code: null, address: null, phone: null, email: null, contact_name: null, payment_terms: null, org_unit_id: unit, active: true },
    }),
  )
  const quote = await created(
    api.post('/api/sales/quotes', {
      data: {
        request_id: crypto.randomUUID(),
        date: '2026-10-01',
        org_unit_id: unit,
        customer_id: customer,
        valid_until: '2026-12-31',
        delivery_date: null,
        payment_terms: null,
        delivery_terms: null,
        note: null,
        lines: [{ item_id: item, description: `Giấy A4 ${run}`, quantity: '2', unit_price: 70000, discount_percent: '0', vat_rate: '10' }],
      },
    }),
  )

  await signedIn(page, `/sales/quotes/${quote}`)
  const terms = page.getByRole('textbox', { name: 'Điều kiện giao hàng' })
  await terms.fill('Giao trong 3 ngày')
  await page.getByRole('button', { name: 'Gửi duyệt' }).click()

  await expect(page.getByRole('alert').filter({ hasText: 'Lưu thay đổi trước khi gửi duyệt hoặc rút lại.' })).toBeVisible()
  await expect(terms).toHaveValue('Giao trong 3 ngày')
  const saved = (await (await api.get(`/api/sales/quotes/${quote}`)).json()) as { status: string }
  expect(saved.status).toBe('draft')
})
