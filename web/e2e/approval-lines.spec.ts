import { expect, test, type APIRequestContext, type Browser, type Locator, type Page } from '@playwright/test'
import { created, createUser, password, signedInApi, runId } from './api'
import { admin } from './helpers'

// One run: a sales team's staff and manager, a special item and a plain one. The post rule of
// quotations is tenant-wide, so this file runs alone and puts back what it found.
const run = runId()
const login = { staff: `alstaff_${run}`, manager: `almgr_${run}` }
const special = { code: `TH-${run}`, name: 'Thép đặc chủng' }
let api: APIRequestContext
let staffApi: APIRequestContext
let ids: { unit: number; customer: number; special: number; plain: number }
let originalRule: unknown

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  const rules = (await (await api.get('/api/approval-rules?product=sales')).json()) as { doc_type: string; action: string; rule: unknown }[]
  originalRule = rules.find((r) => r.doc_type === 'sales.quote' && r.action === 'post')?.rule ?? null
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty AL ${run}`, legal_name: `Công ty TNHH AL ${run}` } }))
  const unit = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Kinh doanh AL ${run}` } }))
  await createUser(api, login.staff, [{ role: 'sales.staff', unit }])
  await createUser(api, login.manager, [{ role: 'sales.manager', unit }])
  const item = (code: string, name: string) => created(api.post('/api/sales/items', { data: { code, name, unit: 'kg', price: 50_000, vat_rate: '10', active: true } }))
  const customer = await created(
    api.post('/api/sales/customers', {
      data: { code: `KHAL${run}`, name: `Công ty Hòa Phát ${run}`, tax_code: null, address: null, phone: null, email: null, contact_name: null, payment_terms: null, org_unit_id: unit, active: true },
    }),
  )
  ids = { unit, customer, special: await item(special.code, special.name), plain: await item(`AL${run}`, `Thép thường ${run}`) }
  staffApi = await signedInApi(baseURL!, login.staff, password)
})

test.afterAll(async () => {
  // beforeAll may have failed before `api` was signed in; nothing to restore then.
  if (!api) return
  const res = originalRule ? await api.put('/api/approval-rules/sales.quote', { data: originalRule }) : await api.delete('/api/approval-rules/sales.quote')
  expect(res.status(), await res.text()).toBe(204)
})

async function as(browser: Browser, user: string, path: string) {
  const page = await (await browser.newContext()).newPage()
  await page.goto(path)
  await page.getByLabel('Tên đăng nhập').fill(user)
  await page.getByLabel(/^Mật khẩu/).fill(user === admin.login ? admin.password : password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: 'Tài khoản' })).toBeVisible()
  return page
}

async function pick(scope: Locator, page: Page, field: string, option: string) {
  await scope.getByRole('combobox', { name: field, exact: true }).click()
  await page.getByRole('option', { name: option, exact: true }).click()
}

// sent makes a quotation of the staff with one line of the item and sends it.
async function sent(item: number) {
  const d = new Date()
  const id = await created(
    staffApi.post('/api/sales/quotes', {
      data: {
        request_id: crypto.randomUUID(),
        date: `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`,
        org_unit_id: ids.unit,
        customer_id: ids.customer,
        valid_until: `${d.getFullYear()}-12-31`,
        delivery_date: null,
        payment_terms: null,
        delivery_terms: null,
        note: null,
        lines: [{ item_id: item, description: `Thép ${run}`, quantity: '2', unit_price: 50_000, discount_percent: '0', vat_rate: '10' }],
      },
    }),
  )
  const res = await staffApi.post(`/api/documents/sales.quote/${id}/transitions`, { data: { to: 'posted', version: 1 } })
  expect(res.status(), await res.text()).toBe(200)
  return (await (await staffApi.get(`/api/sales/quotes/${id}`)).json()) as { status: string; number: string }
}

test('a quote selling the special item goes to the manager', async ({ browser }) => {
  await api.delete('/api/approval-rules/sales.quote')
  const page = await as(browser, admin.login, '/sales/approval-rules/sales.quote')
  const form = page.getByRole('main')
  // A choose-one split replaces the single step the editor starts with.
  await form.getByRole('button', { name: 'Thêm', exact: true }).last().click()
  await page.getByRole('menuitem', { name: 'Rẽ nhánh chọn một' }).click()
  await form.getByRole('region', { name: 'Bước duyệt' }).first().getByRole('button', { name: 'Thao tác khác' }).click()
  await page.getByRole('menuitem', { name: 'Xóa bước' }).click()

  const branch = form.getByRole('region', { name: 'Rẽ nhánh chọn một' }).getByRole('region', { name: 'Nhánh 1' })
  await branch.getByRole('button', { name: 'Thêm điều kiện' }).click()
  await pick(branch, page, 'Trường', 'Mặt hàng')
  // The only comparison of a set field is filled in.
  await expect(branch.getByRole('combobox', { name: 'So sánh' })).toHaveValue('Có')
  await branch.getByRole('combobox', { name: /^Giá trị/ }).fill('TH-')
  await page.getByRole('option', { name: `${special.code} · ${special.name}`, exact: true }).click()
  await pick(branch, page, 'Vai trò', 'Quản lý bán hàng')
  await pick(form, page, 'Vai trò dự phòng', 'Quản lý bán hàng')
  await page.getByRole('button', { name: 'Lưu' }).click()
  await expect(page.getByText('Đã lưu quy tắc duyệt.')).toBeVisible()

  const rules = (await (await api.get('/api/approval-rules?product=sales')).json()) as { doc_type: string; action: string; rule: { flow: unknown } | null }[]
  const flow = rules.find((r) => r.doc_type === 'sales.quote' && r.action === 'post')?.rule?.flow
  expect(JSON.stringify(flow)).toMatch(new RegExp(`"field":"items","op":"has","value":"${ids.special}"`))

  // The saved item reads in full before any search.
  await page.reload()
  await expect(branch.getByRole('combobox', { name: /^Giá trị/ })).toHaveValue(`${special.code} · ${special.name}`)

  const held = await sent(ids.special)
  expect(held.status).toBe('pending_approval')
  const manager = await as(browser, login.manager, '/inbox')
  await expect(manager.getByRole('button', { name: new RegExp(held.number) }).first()).toBeVisible()

  const plain = await sent(ids.plain)
  expect(plain.status).toBe('posted')
})
