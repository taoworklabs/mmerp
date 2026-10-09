import { expect, test, type Browser, type Page } from '@playwright/test'
import { created, createUser, password, signedInApi } from './api'

// One run: team A (staff and their manager) and team B; a catalogue admin. Quotations with a
// discount above 10 % wait for a manager.
const run = Date.now().toString(36)
const login = { staff: `m14staff_${run}`, manager: `m14mgr_${run}`, other: `m14b_${run}`, catalog: `m14cat_${run}` }
const team = `Kinh doanh A ${run}`

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty M14 ${run}`, legal_name: `Công ty TNHH M14 ${run}` } }))
  const a = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: team } }))
  const b = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Kinh doanh B ${run}` } }))
  await createUser(api, login.staff, [{ role: 'sales.staff', unit: a }])
  await createUser(api, login.manager, [{ role: 'sales.manager', unit: a }])
  await createUser(api, login.other, [{ role: 'sales.staff', unit: b }])
  await createUser(api, login.catalog, [{ role: 'sales.catalog_admin', unit: null }])
  const rule = await api.put('/api/approval-rules/sales.quote', {
    data: {
      steps: [{ condition: { field: 'max_discount', op: 'gt', value: '10' }, approver: { kind: 'role', product: 'sales', role: 'manager' } }],
      max_levels: 3,
      fallback_product: 'sales',
      fallback_role: 'manager',
    },
  })
  expect(rule.status(), await rule.text()).toBe(204)
})

async function as(browser: Browser, user: string, url: string): Promise<Page> {
  const page = await (await browser.newContext()).newPage()
  await page.goto(url)
  await page.getByLabel('Tên đăng nhập').fill(user)
  await page.getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: 'Tài khoản' })).toBeVisible()
  return page
}

const today = () => {
  const d = new Date()
  return `${String(d.getDate()).padStart(2, '0')}/${String(d.getMonth() + 1).padStart(2, '0')}/${d.getFullYear()}`
}

test('customer and item, a quotation approved by the manager, printed, then ordered once', async ({ browser }) => {
  // The catalogue admin adds an item.
  const cat = await as(browser, login.catalog, '/sales/items')
  await cat.getByRole('button', { name: 'Thêm mặt hàng' }).click()
  const dialog = cat.getByRole('dialog')
  await dialog.getByRole('textbox', { name: 'Mã hàng' }).fill(`BUT${run}`)
  await dialog.getByRole('textbox', { name: 'Tên hàng' }).fill(`Bút bi ${run}`)
  await dialog.getByRole('textbox', { name: 'Đơn vị tính' }).fill('hộp')
  await dialog.getByRole('textbox', { name: /^Đơn giá mặc định/ }).fill('10005')
  await dialog.getByRole('combobox', { name: /^Thuế suất GTGT/ }).click()
  await cat.getByRole('option', { name: '8%' }).click()
  await dialog.getByRole('button', { name: 'Lưu' }).click()
  await expect(cat.getByText('Đã lưu mặt hàng.')).toBeVisible()

  // Staff of A adds a customer of A.
  const staff = await as(browser, login.staff, '/sales/customers')
  await staff.getByRole('link', { name: 'Thêm khách hàng' }).click()
  await staff.getByRole('textbox', { name: 'Mã khách hàng' }).fill(`KH${run}`)
  await staff.getByRole('textbox', { name: 'Tên khách hàng' }).fill(`Công ty Ánh Dương ${run}`)
  await staff.getByRole('textbox', { name: 'Mã số thuế' }).fill('0312345678')
  await staff.getByRole('combobox', { name: 'Đơn vị phụ trách' }).click()
  await staff.getByRole('option', { name: team }).click()
  await staff.getByRole('textbox', { name: 'Điều kiện thanh toán' }).fill('Thanh toán trong 30 ngày')
  await staff.getByRole('button', { name: 'Thêm khách hàng' }).click()
  await expect(staff.getByRole('heading', { name: `Công ty Ánh Dương ${run}` })).toBeVisible()

  // A quotation with a 15 % discount: twelve boxes at 10,005.
  await staff.goto('/sales/quotes')
  await staff.getByRole('link', { name: 'Lập báo giá' }).click()
  await staff.getByRole('textbox', { name: 'Ngày báo giá' }).fill(today())
  await staff.getByRole('textbox', { name: 'Hiệu lực đến' }).fill(`31/12/${new Date().getFullYear()}`)
  await staff.getByRole('combobox', { name: 'Đơn vị', exact: true }).click()
  await staff.getByRole('option', { name: team }).click()
  await staff.getByRole('combobox', { name: 'Khách hàng' }).fill(`KH${run}`)
  await staff.getByRole('option', { name: new RegExp(`KH${run}`) }).click()
  await expect(staff.getByRole('textbox', { name: 'Điều kiện thanh toán' })).toHaveValue('Thanh toán trong 30 ngày')
  await staff.getByRole('combobox', { name: 'Mặt hàng (1)' }).fill(`BUT${run}`)
  await staff.getByRole('option', { name: new RegExp(`BUT${run}`) }).click()
  await expect(staff.getByRole('textbox', { name: 'Diễn giải (1)' })).toHaveValue(`Bút bi ${run}`)
  await staff.getByRole('textbox', { name: 'Số lượng (1)' }).fill('12')
  await staff.getByRole('textbox', { name: 'Chiết khấu % (1)' }).fill('15')
  await staff.getByRole('button', { name: 'Tạo bản nháp' }).click()
  await expect(staff.getByRole('heading', { name: /^BG-\d{4}-/ })).toBeVisible()
  // 120,060 less 18,009 = 102,051; VAT 8 % = 8,164.08 → 8,164; total 110,215.
  await expect(staff.getByText('110.215', { exact: true }).first()).toBeVisible()
  const quoteUrl = staff.url()

  await staff.getByRole('button', { name: 'Gửi duyệt' }).click()
  await expect(staff.getByText('Chờ duyệt').first()).toBeVisible()

  // Team B sees nothing of it, even by its address.
  const other = await as(browser, login.other, '/sales/quotes')
  await expect(other.getByText('Chưa có báo giá')).toBeVisible()
  await other.goto(quoteUrl)
  await expect(other.getByRole('link', { name: 'Về danh sách báo giá' })).toBeVisible()

  // The manager approves it from the inbox.
  const manager = await as(browser, login.manager, '/inbox')
  await manager.getByRole('button', { name: /BG-\d{4}-/ }).first().click()
  await expect(manager.getByText(`KH${run} · Công ty Ánh Dương ${run}`)).toBeVisible()
  await manager.getByRole('button', { name: 'Duyệt', exact: true }).click()

  await staff.reload()
  await expect(staff.getByText('Đã duyệt').first()).toBeVisible()

  // Printed to a PDF the staff downloads.
  await staff.getByRole('button', { name: 'In báo giá' }).click()
  const link = staff.getByRole('link', { name: 'Tải file' })
  const href = (await link.getAttribute('href'))!
  const file = await staff.request.get(href)
  expect(file.headers()['content-type']).toBe('application/pdf')
  expect((await file.text()).startsWith('%PDF-')).toBe(true)

  // Turned into an order; asking again gives the same order.
  await staff.getByRole('button', { name: 'Tạo đơn bán hàng' }).click()
  await expect(staff.getByRole('heading', { name: /^DH-\d{4}-/ })).toBeVisible()
  const orderUrl = staff.url()
  await expect(staff.getByText('110.215', { exact: true }).first()).toBeVisible()
  const quoteId = Number(quoteUrl.split('/').pop())
  const again = await staff.request.post(`/api/sales/quotes/${quoteId}/order`)
  expect(again.status()).toBe(201)
  expect(orderUrl.endsWith(`/sales/orders/${((await again.json()) as { id: number }).id}`)).toBe(true)

  // No discount above 10 % on orders' rule: sending confirms it at once.
  await staff.getByRole('button', { name: 'Gửi duyệt' }).click()
  await expect(staff.getByText('Đã xác nhận').first()).toBeVisible()
  await staff.goto(quoteUrl)
  await expect(staff.getByText('Đã lên đơn')).toBeVisible()
  await expect(staff.getByRole('button', { name: 'Tạo đơn bán hàng' })).toHaveCount(0)
})
