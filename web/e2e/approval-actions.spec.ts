import { expect, test, type APIRequestContext, type Browser, type Page } from '@playwright/test'
import { created, createUser, password, signedInApi, runId, toFlow } from './api'
import { admin } from './helpers'

// One run: a sales team's staff and manager. The cancel rule of orders and the create_order rule
// of quotations are tenant-wide, so this file runs alone and puts back what it found.
const run = runId()
const login = { staff: `aastaff_${run}`, manager: `aamgr_${run}` }
const touched = [
  ['sales.order', 'cancel'],
  ['sales.quote', 'create_order'],
] as const
let api: APIRequestContext
let staffApi: APIRequestContext
let ids: { unit: number; customer: number; item: number }
const original = new Map<string, unknown>()

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  const rules = (await (await api.get('/api/approval-rules?product=sales')).json()) as { doc_type: string; action: string; rule: unknown }[]
  for (const [type, action] of touched) original.set(`${type}:${action}`, rules.find((r) => r.doc_type === type && r.action === action)?.rule ?? null)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty AA ${run}`, legal_name: `Công ty TNHH AA ${run}` } }))
  const unit = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Kinh doanh AA ${run}` } }))
  await createUser(api, login.staff, [{ role: 'sales.staff', unit }, { role: 'partner.editor', unit }])
  await createUser(api, login.manager, [{ role: 'sales.manager', unit }, { role: 'partner.editor', unit }])
  const item = await created(api.post('/api/items', { data: { code: `AA${run}`, name: `Giấy A4 ${run}`, unit: 'ram', price: 78_000, vat_rate: '10', active: true } }))
  const customer = await created(
    api.post('/api/sales/customers', {
      data: { code: `KHAA${run}`, partner: { name: `Công ty Bình Minh ${run}`, tax_code: null, address: null, phone: null, email: null, contact_name: null }, payment_terms: null, org_unit_id: unit, active: true },
    }),
  )
  ids = { unit, customer, item }
  staffApi = await signedInApi(baseURL!, login.staff, password)
})

test.afterAll(async () => {
  // beforeAll may have failed before `api` was signed in; nothing to restore then.
  if (!api) return
  for (const [type, action] of touched) {
    const rule = original.get(`${type}:${action}`)
    const res = rule ? await api.put(`/api/approval-rules/${type}?action=${action}`, { data: rule }) : await api.delete(`/api/approval-rules/${type}?action=${action}`)
    expect(res.status(), await res.text()).toBe(204)
  }
})

async function as(browser: Browser, user: string, path: string) {
  const page = await (await browser.newContext()).newPage()
  await page.goto(path)
  await page.getByLabel('Tên đăng nhập').fill(user)
  await page.getByLabel(/^Mật khẩu/).fill(user === admin.login ? admin.password : password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: /^Tài khoản:/ })).toBeVisible()
  return page
}

const managerRule = { flow: toFlow([{ approver: { kind: 'role', product: 'sales', role: 'manager' } }]), max_levels: 3, fallback_product: 'sales', fallback_role: 'manager' }

async function putRule(type: string, action: string) {
  const res = await api.put(`/api/approval-rules/${type}?action=${action}`, { data: managerRule })
  expect(res.status(), await res.text()).toBe(204)
}

// posted makes a quotation or order of the staff, without a discount, and posts it.
async function posted(kind: 'quotes' | 'orders') {
  const d = new Date()
  const year = d.getFullYear()
  const id = await created(
    staffApi.post(`/api/sales/${kind}`, {
      data: {
        request_id: crypto.randomUUID(),
        date: `${year}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`,
        org_unit_id: ids.unit,
        customer_id: ids.customer,
        valid_until: kind === 'quotes' ? `${year}-12-31` : null,
        delivery_date: null,
        payment_terms: null,
        delivery_terms: null,
        note: null,
        lines: [{ item_id: ids.item, description: `Giấy A4 ${run}`, quantity: '2', unit_price: 78_000, discount_percent: '0', vat_rate: '10' }],
      },
    }),
  )
  const type = kind === 'quotes' ? 'sales.quote' : 'sales.order'
  const res = await staffApi.post(`/api/documents/${type}/${id}/transitions`, { data: { to: 'posted', version: 1 } })
  expect(res.status(), await res.text()).toBe(200)
  const doc = (await (await staffApi.get(`/api/sales/${kind}/${id}`)).json()) as { status: string; number: string }
  expect(doc.status).toBe('posted')
  return { id, number: doc.number }
}

// managerApproves opens the document from the manager's inbox and approves the request it waits for.
async function managerApproves(browser: Browser, number: string, request: string) {
  const page = await as(browser, login.manager, '/inbox')
  await page.getByRole('button', { name: new RegExp(number) }).first().click()
  await expect(page.getByText(request)).toBeVisible()
  await page.getByRole('button', { name: 'Duyệt', exact: true }).click()
  await expect(page.getByText('Đã duyệt.')).toBeVisible()
}

async function orderRules() {
  const rules = (await (await api.get('/api/approval-rules?product=sales')).json()) as { doc_type: string; action: string; rule: unknown }[]
  const of = (action: string) => rules.find((r) => r.doc_type === 'sales.order' && r.action === action)?.rule ?? null
  return { post: of('post'), cancel: of('cancel') }
}

const status = (page: Page, text: string) => expect(page.getByRole('main').getByText(text, { exact: true }).first()).toBeVisible()

test('cancelling a posted order waits for the manager', async ({ browser }) => {
  await api.delete('/api/approval-rules/sales.order?action=cancel')
  const order = await posted('orders')
  const before = await orderRules()

  // The administrator sets the cancel rule of orders in the editor.
  const admin_ = await as(browser, admin.login, '/sales/approval-rules')
  await admin_.getByRole('link', { name: 'Đơn bán hàng · Hủy chứng từ' }).click()
  await expect(admin_).toHaveURL(/\/sales\/approval-rules\/sales\.order\/cancel$/)
  // No rule yet: one step, approved by the sales managers.
  await admin_.getByRole('main').getByRole('button', { name: 'Thêm vào đây' }).click()
  await admin_.getByRole('menuitem', { name: /^Bước duyệt/ }).click()
  const panel = admin_.getByRole('complementary')
  await panel.getByRole('combobox', { name: 'Vai trò', exact: true }).click()
  await admin_.getByRole('option', { name: 'Quản lý bán hàng', exact: true }).click()
  await admin_.getByRole('button', { name: 'Cài đặt chung' }).click()
  await panel.getByRole('combobox', { name: 'Không tìm được người duyệt thì chuyển cho' }).click()
  await admin_.getByRole('option', { name: 'Quản lý bán hàng', exact: true }).click()
  await admin_.getByRole('button', { name: 'Lưu' }).click()
  await expect(admin_.getByText('Đã lưu quy tắc duyệt.')).toBeVisible()
  // Only the cancel rule changed; the order's post rule is its own.
  const after = await orderRules()
  expect(after.cancel).toBeTruthy()
  expect(after.post).toEqual(before.post)

  // Staff cancel it: the order stays confirmed and shows the request.
  const staff = await as(browser, login.staff, `/sales/orders/${order.id}`)
  await staff.getByRole('button', { name: 'Hủy chứng từ' }).click()
  await staff.getByRole('dialog').getByRole('button', { name: 'Hủy chứng từ' }).click()
  await expect(staff.getByText(`Đã gửi yêu cầu duyệt cho ${order.number}.`)).toBeVisible()
  await status(staff, 'Đã xác nhận')
  await expect(staff.getByRole('button', { name: 'Hủy chứng từ' })).toHaveCount(0)
  // On a wide screen the approval panel sits in the activity tab.
  await staff.getByRole('tab', { name: 'Hoạt động' }).click()
  await expect(staff.getByText('Yêu cầu hủy chứng từ')).toBeVisible()

  await managerApproves(browser, order.number, 'Yêu cầu hủy chứng từ')
  await staff.reload()
  await status(staff, 'Đã hủy')
})

test('a cancel request can be withdrawn', async ({ browser }) => {
  await putRule('sales.order', 'cancel')
  const order = await posted('orders')
  const res = await staffApi.post(`/api/documents/sales.order/${order.id}/transitions`, { data: { to: 'cancelled', version: 2 } })
  expect(res.status(), await res.text()).toBe(200)
  expect(((await res.json()) as { held: boolean }).held).toBe(true)

  const staff = await as(browser, login.staff, `/sales/orders/${order.id}?tab=approval`)
  await expect(staff.getByText('Yêu cầu hủy chứng từ')).toBeVisible()
  await staff.getByRole('button', { name: 'Rút yêu cầu' }).click()
  await staff.getByRole('dialog').getByRole('button', { name: 'Rút yêu cầu' }).click()
  await expect(staff.getByText('Đã rút yêu cầu.')).toBeVisible()
  await expect(staff.getByRole('button', { name: 'Hủy chứng từ' })).toBeVisible()
  await status(staff, 'Đã xác nhận')
})

test('creating an order from a quote waits for approval', async ({ browser }) => {
  await putRule('sales.quote', 'create_order')
  const quote = await posted('quotes')

  const staff = await as(browser, login.staff, `/sales/quotes/${quote.id}`)
  await staff.getByRole('button', { name: 'Tạo đơn bán hàng' }).click()
  await expect(staff.getByText(`Đã gửi yêu cầu duyệt cho ${quote.number}.`)).toBeVisible()
  await expect(staff).toHaveURL(new RegExp(`/sales/quotes/${quote.id}$`))
  await expect(staff.getByRole('button', { name: 'Tạo đơn bán hàng' })).toHaveCount(0)
  expect(((await (await staffApi.get(`/api/sales/quotes/${quote.id}`)).json()) as { order: unknown }).order ?? null).toBeNull()

  await managerApproves(browser, quote.number, 'Yêu cầu duyệt: Tạo đơn bán hàng')
  await staff.reload()
  await expect(staff.getByRole('link', { name: /^Đơn DH-\d{4}-/ })).toBeVisible()
})
