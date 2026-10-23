import { expect, test, type APIRequestContext } from '@playwright/test'
import { created, employee, signedInApi, runId } from './api'
import { signedIn } from './helpers'

// Guards that keep a user's work or a failure from vanishing without a word: unsaved edits
// before submitting, an invalid timesheet cell, a delete the server refused.
const run = runId()
let api: APIRequestContext
let company: number
let unit: number

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty G ${run}`, legal_name: `Công ty TNHH G ${run}` } }))
  unit = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng G ${run}` } }))
})

test('submitting a quotation with unsaved edits asks to save first and keeps the edits', async ({ page }) => {
  const item = await created(api.post('/api/items', { data: { code: `G${run}`, name: `Giấy A4 ${run}`, unit: 'ram', price: 70000, vat_rate: '10', active: true } }))
  const customer = await created(
    api.post('/api/sales/customers', {
      data: { code: `KG${run}`, partner: { name: `Khách G ${run}`, tax_code: null, address: null, phone: null, email: null, contact_name: null }, payment_terms: null, org_unit_id: unit, active: true },
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

test('an invalid timesheet cell is named and the footer leads to it', async ({ page }) => {
  const code = `G${run}E1`
  await created(api.post('/api/hrm/employees', { data: employee(code, unit) }))
  const sheet = await created(api.post('/api/hrm/timesheets', { data: { org_unit_id: unit, month: '2026-04' } }))

  await signedIn(page, `/hrm/timesheets/${sheet}`)
  const cell = page.getByRole('textbox', { name: `Nhân viên ${code}, 01/04/2026` })
  await cell.fill('2')
  await page.getByRole('button', { name: 'Lưu' }).click()

  await expect(page.getByText('Có ô không hợp lệ: mỗi ô chỉ nhận 0,5 hoặc 1.')).toBeVisible()
  await page.getByRole('button', { name: 'Lưu' }).focus()
  await page.getByRole('button', { name: 'Đến trường tiếp theo' }).click()
  await expect(cell).toBeFocused()
})

test('a holiday the server refuses to delete says so', async ({ page }) => {
  const put = await api.put(`/api/hrm/calendars/${company}/holidays/2026-12-24`, { data: { name: `Nghỉ G ${run}` } })
  expect(put.status(), await put.text()).toBe(204)
  await page.route('**/api/hrm/calendars/*/holidays/*', (route) =>
    route.request().method() === 'DELETE' ? route.fulfill({ status: 500, contentType: 'application/json', body: '{"code":"internal_error"}' }) : route.continue(),
  )

  await signedIn(page, `/hrm/work-calendar?legal_entity=${company}&year=2026`)
  await page.getByRole('row', { name: new RegExp(`Nghỉ G ${run}`) }).getByRole('button', { name: 'Thao tác khác' }).click()
  await page.getByRole('menuitem', { name: 'Xóa' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Xóa' }).click()

  await expect(page.getByRole('alert').filter({ hasText: 'Máy chủ gặp lỗi.' })).toBeVisible()
})
