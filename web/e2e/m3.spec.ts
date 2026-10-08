import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { created, createUser, employee, password, signedInApi } from './api'
import { admin } from './helpers'

// One run: department P with manager M and employees E and E2 (both reporting to M), E3
// reporting to B, an HR user of P, all with accounts. Every leave request needs the direct manager.
const run = Date.now().toString(36)
const login = { m: `m_${run}`, e: `e_${run}`, e2: `e2_${run}`, e3: `e3_${run}`, b: `b_${run}` }
let ids: { e: number; e2: number; e3: number; annual: number }
let api: APIRequestContext

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty nghỉ ${run}` } }))
  const p = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng P ${run}` } }))
  for (const l of [login.m, login.e, login.e2, login.e3]) await createUser(api, l, [])
  await createUser(api, login.b, [{ role: 'hrm.hr', unit: p }])
  const m = await created(api.post('/api/hrm/employees', { data: employee(`M${run}`, p, { user_login: login.m }) }))
  const b = await created(api.post('/api/hrm/employees', { data: employee(`B${run}`, p, { user_login: login.b }) }))
  const e = await created(api.post('/api/hrm/employees', { data: employee(`E${run}`, p, { user_login: login.e, manager_id: m }) }))
  const e2 = await created(api.post('/api/hrm/employees', { data: employee(`E2${run}`, p, { user_login: login.e2, manager_id: m }) }))
  const e3 = await created(api.post('/api/hrm/employees', { data: employee(`E3${run}`, p, { user_login: login.e3, manager_id: b }) }))
  const annual = await created(api.post('/api/hrm/leave-types', { data: { name: `Phép năm ${run}`, deducts_balance: true, paid: true, active: true } }))
  for (const id of [e2, e3]) {
    const res = await api.post(`/api/hrm/employees/${id}/leave-balances/2026/adjustments`, { data: { delta: '12', reason: 'đầu năm' } })
    expect(res.status()).toBe(204)
  }
  const rule = await api.put('/api/approval-rules/hrm.leave_request', {
    data: { steps: [{ approver: { kind: 'module' } }], max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' },
  })
  expect(rule.status(), await rule.text()).toBe(204)
  ids = { e, e2, e3, annual }
})

async function signInAs(page: Page, user: string, url: string) {
  await page.goto(url)
  await page.getByLabel('Tên đăng nhập').fill(user)
  await page.getByLabel(/^Mật khẩu/).fill(user === admin.login ? admin.password : password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: 'Tài khoản' })).toBeVisible()
}

// fillLeave fills the leave form for a request of `days` days starting on the 10th of `month`.
async function fillLeave(page: Page, month: string, days: string) {
  await page.getByRole('combobox', { name: 'Loại nghỉ' }).click()
  await page.getByRole('option', { name: `Phép năm ${run}` }).click()
  await page.getByRole('textbox', { name: /^Số ngày/ }).fill(days)
  await page.getByRole('textbox', { name: /^Từ ngày/ }).fill(`10/${month}/2026`)
  await page.getByRole('textbox', { name: /^Đến ngày/ }).fill(`14/${month}/2026`)
  await page.getByRole('textbox', { name: /^Lý do/ }).click()
}

async function newLeave(page: Page, month: string, days: string) {
  await page.goto('/hrm/leaves/new')
  await fillLeave(page, month, days)
  await page.getByRole('button', { name: 'Tạo bản nháp' }).click()
  await expect(page.getByRole('heading', { name: /^NP-2026-/ })).toBeVisible()
  return Number(new URL(page.url()).pathname.split('/').pop())
}

// sendViaApi creates and sends a request as another user, for setting up a waiting approval.
async function sendViaApi(baseURL: string, user: string, month: string, days: string) {
  const as = await signedInApi(baseURL, user, password)
  const id = await created(
    as.post('/api/hrm/leaves', { data: { leave_type_id: ids.annual, start_date: `2026-${month}-10`, end_date: `2026-${month}-14`, days, reason: null } }),
  )
  const res = await as.post(`/api/documents/hrm.leave_request/${id}/transitions`, { data: { to: 'posted', version: 1 } })
  expect(res.status(), await res.text()).toBe(204)
  return id
}

const status = (page: Page, text: string) => expect(page.getByRole('main').getByText(text, { exact: true }).first()).toBeVisible()

test('HR grants the start-of-year leave balance through the UI', async ({ page }) => {
  await signInAs(page, admin.login, `/hrm/employees/${ids.e}?tab=balances`)
  await expect(page.getByText('Chưa cấp quỹ phép.')).toBeVisible()
  await page.getByRole('button', { name: 'Cấp, điều chỉnh quỹ phép' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('textbox', { name: /^Năm/ }).fill('2026')
  await dialog.getByRole('textbox', { name: /^Số ngày cộng thêm/ }).fill('12')
  await dialog.getByRole('textbox', { name: /^Lý do/ }).fill('Cấp phép năm 2026')
  await dialog.getByRole('button', { name: 'Lưu điều chỉnh' }).click()
  await expect(page.getByRole('table', { name: 'Quỹ phép' }).getByRole('row', { name: /2026.*12/ })).toBeVisible()
})

test('an employee sends, withdraws and sends again; the manager approves one request and rejects another', async ({ browser }) => {
  const e = await (await browser.newContext()).newPage()
  await signInAs(e, login.e, '/hrm/leaves')
  const first = await newLeave(e, '03', '2')
  await status(e, 'Nháp')
  await e.getByRole('button', { name: 'Gửi duyệt' }).click()
  await status(e, 'Chờ duyệt')
  await e.getByRole('button', { name: 'Rút lại' }).click()
  await status(e, 'Nháp')
  await e.getByRole('button', { name: 'Gửi duyệt' }).click()
  await status(e, 'Chờ duyệt')
  const second = await newLeave(e, '05', '1')
  await e.getByRole('button', { name: 'Gửi duyệt' }).click()
  await status(e, 'Chờ duyệt')

  const m = await (await browser.newContext()).newPage()
  await signInAs(m, login.m, '/inbox')
  await m.getByRole('button', { name: new RegExp(`NP-2026-.*`) }).first().waitFor()
  // Approve the first from the inbox.
  await m.goto(`/hrm/leaves/${first}`)
  await m.getByRole('button', { name: 'Duyệt', exact: true }).click()
  await status(m, 'Đã duyệt')
  // Reject the second, with a reason.
  await m.goto(`/hrm/leaves/${second}`)
  await m.getByRole('button', { name: 'Từ chối' }).click()
  await m.getByRole('dialog').getByRole('textbox', { name: /^Lý do/ }).fill('Trùng lịch dự án')
  await m.getByRole('dialog').getByRole('button', { name: 'Từ chối' }).click()
  await status(m, 'Nháp')
  await expect(m.getByText('Lý do: Trùng lịch dự án')).toBeVisible()
})

test('two tabs editing the same draft: the second save gets the reload dialog', async ({ browser }) => {
  const ctx = await browser.newContext()
  const one = await ctx.newPage()
  await signInAs(one, login.e, '/hrm/leaves')
  const id = await newLeave(one, '06', '1')
  const two = await ctx.newPage()
  await two.goto(`/hrm/leaves/${id}`)
  await expect(two.getByRole('textbox', { name: /^Số ngày/ })).toHaveValue('1')

  await one.getByRole('textbox', { name: /^Số ngày/ }).fill('2')
  await one.getByRole('button', { name: 'Lưu' }).click()
  await expect(one.getByText('Thông tin nghỉ')).toBeVisible()

  await two.getByRole('textbox', { name: /^Số ngày/ }).fill('3')
  await two.getByRole('button', { name: 'Lưu' }).click()
  await expect(two.getByRole('dialog', { name: 'Dữ liệu đã thay đổi' })).toBeVisible()
  await two.getByRole('button', { name: 'Tải lại' }).click()
  await expect(two.getByRole('textbox', { name: /^Số ngày/ })).toHaveValue('2')
})

test('approving from the inbox refreshes the leave balance on the HRM screen', async ({ page, baseURL }) => {
  await sendViaApi(baseURL!, login.e3, '07', '2')
  await signInAs(page, login.b, `/hrm/employees/${ids.e3}?tab=balances`)
  const balances = page.getByRole('table', { name: 'Quỹ phép' })
  await expect(balances.getByRole('row', { name: /2026.*12/ })).toBeVisible()

  await page.getByRole('link', { name: /Hộp duyệt/ }).click()
  await page.getByRole('button', { name: new RegExp(`NP-2026-.*`) }).first().click()
  await page.getByRole('button', { name: 'Duyệt', exact: true }).click()
  await expect(page.getByText('Đã duyệt.')).toBeVisible()

  // Back in the app, without reloading: the balance is already fresh.
  await expect(page.getByRole('heading', { name: 'Hộp duyệt' })).toBeVisible()
  await page.goBack()
  await expect(balances.getByRole('row', { name: /2026.*10/ })).toBeVisible()
})

test('at 375px an employee sends a request and the manager approves it', async ({ browser }) => {
  const small = { viewport: { width: 375, height: 800 } }
  const e = await (await browser.newContext(small)).newPage()
  await signInAs(e, login.e2, '/hrm/leaves')
  await e.getByRole('link', { name: 'Tạo đơn nghỉ' }).click()
  await fillLeave(e, '08', '1')
  await e.getByRole('button', { name: 'Tạo bản nháp' }).click()
  await expect(e.getByRole('heading', { name: /^NP-2026-/ })).toBeVisible()
  const number = (await e.getByRole('heading', { name: /^NP-2026-/ }).textContent()) ?? ''
  await e.getByRole('button', { name: 'Gửi duyệt' }).click()
  await status(e, 'Chờ duyệt')

  const m = await (await browser.newContext(small)).newPage()
  await signInAs(m, login.m, '/inbox')
  await m.getByRole('button', { name: new RegExp(number) }).click()
  await m.getByRole('button', { name: 'Duyệt', exact: true }).click()
  await expect(m.getByText('Đã duyệt.')).toBeVisible()
  // Back returns to the list, where the item is gone.
  await expect(m.getByRole('button', { name: new RegExp(number) })).toHaveCount(0)

  await e.reload()
  await status(e, 'Đã duyệt')
})
