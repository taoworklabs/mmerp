import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { created, createUser, employee, password, signedInApi, runId, toFlow } from './api'
import { admin, signInAs, status } from './helpers'

// One run: department P with manager M and employees E and E2 (both reporting to M), E3
// reporting to B, an HR user of P, all with accounts. Every leave request needs the direct manager.
const run = runId()
const login = { m: `m_${run}`, e: `e_${run}`, e2: `e2_${run}`, e3: `e3_${run}`, b: `b_${run}` }
let ids: { e: number; e2: number; e3: number; annual: number }
let api: APIRequestContext

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty nghỉ ${run}` } }))
  // Balances here are granted by hand; the hourly accrual job must not add to them mid-run.
  expect((await api.put(`/api/legal-entities/${company}/settings/hrm.leave_accrual`, { data: { value: 'manual' } })).status()).toBe(204)
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
    data: { flow: toFlow([{ approver: { kind: 'module' } }]), max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' },
  })
  expect(rule.status(), await rule.text()).toBe(204)
  ids = { e, e2, e3, annual }
})

// fillLeave fills the leave form for a request from `from` to `to` (dd/mm/yyyy); the days
// are the working days between them.
async function fillLeave(page: Page, from: string, to: string) {
  await page.getByRole('combobox', { name: 'Loại nghỉ' }).click()
  await page.getByRole('option', { name: `Phép năm ${run}` }).click()
  await page.getByRole('textbox', { name: /^Từ ngày/ }).fill(from)
  await page.getByRole('textbox', { name: /^Đến ngày/ }).fill(to)
  await page.getByRole('textbox', { name: /^Lý do/ }).click()
}

async function newLeave(page: Page, from: string, to: string) {
  await page.goto('/me/requests/leave/new')
  await fillLeave(page, from, to)
  await page.getByRole('button', { name: 'Tạo bản nháp' }).click()
  await expect(page.getByRole('heading', { name: /^NP-2026-/ })).toBeVisible()
  return Number(new URL(page.url()).pathname.split('/').pop())
}

// sendViaApi creates and sends a request as another user, for setting up a waiting approval.
async function sendViaApi(baseURL: string, user: string, start: string, end: string) {
  const as = await signedInApi(baseURL, user, password)
  const id = await created(as.post('/api/hrm/leaves', { data: { leave_type_id: ids.annual, start_date: start, end_date: end, reason: null } }))
  const res = await as.post(`/api/documents/hrm.leave_request/${id}/transitions`, { data: { to: 'posted', version: 1 } })
  expect(res.status(), await res.text()).toBe(200)
  return id
}


test('HR grants the start-of-year leave balance through the UI', async ({ page }) => {
  await signInAs(page, admin.login, `/hrm/employees/${ids.e}?tab=balances`)
  await expect(page.getByText('Chưa cấp quỹ phép.')).toBeVisible()
  await page.getByRole('button', { name: 'Cấp, điều chỉnh quỹ phép' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('textbox', { name: /^Năm/ }).fill('2026')
  await dialog.getByRole('textbox', { name: /^Số ngày cộng thêm/ }).fill('12')
  await dialog.getByRole('textbox', { name: /^Lý do/ }).fill('Cấp phép năm 2026')
  await dialog.getByRole('button', { name: 'Lưu điều chỉnh' }).click()
  await expect(page.getByRole('table', { name: 'Quỹ phép' }).getByRole('row').filter({ hasText: /2026[\s\S]*12/ })).toBeVisible()
})

test('an employee sends, withdraws and sends again; the manager approves one request and rejects another', async ({ browser }) => {
  const e = await (await browser.newContext()).newPage()
  await signInAs(e, login.e, '/me/requests')
  const first = await newLeave(e, '10/03/2026', '11/03/2026')
  await status(e, 'Nháp')
  await e.getByRole('button', { name: 'Gửi duyệt' }).click()
  await status(e, 'Chờ duyệt')
  await e.getByRole('button', { name: 'Rút lại' }).click()
  // Withdrawing undoes the approvals so far, so it asks first.
  await e.getByRole('dialog').getByRole('button', { name: 'Rút lại' }).click()
  await status(e, 'Nháp')
  await e.getByRole('button', { name: 'Gửi duyệt' }).click()
  await status(e, 'Chờ duyệt')
  const second = await newLeave(e, '11/05/2026', '11/05/2026')
  await e.getByRole('button', { name: 'Gửi duyệt' }).click()
  await status(e, 'Chờ duyệt')

  const m = await (await browser.newContext()).newPage()
  await signInAs(m, login.m, '/inbox')
  await m.getByRole('button', { name: new RegExp(`NP-2026-.*`) }).first().waitFor()
  // Approve the first from the inbox.
  await m.goto(`/me/requests/leave/${first}`)
  await m.getByRole('button', { name: 'Duyệt', exact: true }).click()
  await status(m, 'Đã duyệt')
  // Reject the second, with a reason.
  await m.goto(`/me/requests/leave/${second}`)
  await m.getByRole('button', { name: 'Từ chối' }).click()
  await m.getByRole('dialog').getByRole('textbox', { name: /^Lý do/ }).fill('Trùng lịch dự án')
  await m.getByRole('dialog').getByRole('button', { name: 'Từ chối' }).click()
  await status(m, 'Nháp')
  await expect(m.getByText('Lý do: Trùng lịch dự án')).toBeVisible()
})

test('two tabs editing the same draft: the second save gets the reload dialog', async ({ browser }) => {
  const ctx = await browser.newContext()
  const one = await ctx.newPage()
  await signInAs(one, login.e, '/me/requests')
  const id = await newLeave(one, '10/06/2026', '10/06/2026')
  const two = await ctx.newPage()
  await two.goto(`/me/requests/leave/${id}`)
  // A draft reads as values; its card turns into the form on Edit.
  await two.getByRole('button', { name: 'Sửa Thông tin nghỉ' }).click()
  await expect(two.getByRole('textbox', { name: /^Số ngày/ })).toHaveValue('1')

  await one.getByRole('button', { name: 'Sửa Thông tin nghỉ' }).click()
  await one.getByRole('textbox', { name: /^Đến ngày/ }).fill('11/06/2026')
  await one.getByRole('textbox', { name: /^Lý do/ }).click()
  await one.getByRole('button', { name: 'Lưu' }).click()
  await expect(one.getByText('Thông tin nghỉ')).toBeVisible()

  await two.getByRole('textbox', { name: /^Đến ngày/ }).fill('12/06/2026')
  await two.getByRole('textbox', { name: /^Lý do/ }).click()
  await two.getByRole('button', { name: 'Lưu' }).click()
  await expect(two.getByRole('dialog', { name: 'Dữ liệu đã thay đổi' })).toBeVisible()
  await two.getByRole('button', { name: 'Tải lại' }).click()
  await expect(two.getByRole('region', { name: 'Thông tin nghỉ' }).getByText('2 ngày làm việc')).toBeVisible()
})

test('approving from the inbox refreshes the leave balance on the HRM screen', async ({ page, baseURL }) => {
  // Friday 10 and Monday 13 July: two working days.
  await sendViaApi(baseURL!, login.e3, '2026-07-10', '2026-07-13')
  await signInAs(page, login.b, `/hrm/employees/${ids.e3}?tab=balances`)
  const balances = page.getByRole('table', { name: 'Quỹ phép' })
  await expect(balances.getByRole('row').filter({ hasText: /2026[\s\S]*12/ })).toBeVisible()

  await page.getByRole('link', { name: /Hộp duyệt/ }).click()
  await page.getByRole('button', { name: new RegExp(`NP-2026-.*`) }).first().click()
  await page.getByRole('button', { name: 'Duyệt', exact: true }).click()
  await expect(page.getByText('Đã duyệt.')).toBeVisible()

  // Back in the app, without reloading: the balance is already fresh.
  await expect(page.getByRole('heading', { name: 'Hộp duyệt' })).toBeVisible()
  await page.goBack()
  await expect(balances.getByRole('row').filter({ hasText: /2026[\s\S]*10/ })).toBeVisible()
})

test('at 375px an employee sends a request and the manager approves it', async ({ browser }) => {
  const small = { viewport: { width: 375, height: 800 } }
  const e = await (await browser.newContext(small)).newPage()
  await signInAs(e, login.e2, '/me/requests')
  await e.getByRole('link', { name: 'Tạo đơn nghỉ' }).click()
  await fillLeave(e, '10/08/2026', '10/08/2026')
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

// The form counts the days from the calendar as soon as both dates are known, half days
// included, and a range of nothing but rest days is refused. 2026-08-10 to 14 is a whole
// working week; 2026-08-08 to 09 is a weekend.
test('the form computes the days of the range, half days included', async ({ page }) => {
  await signInAs(page, login.e3, '/me/requests/leave/new')
  await page.getByRole('combobox', { name: 'Loại nghỉ' }).click()
  await page.getByRole('option', { name: `Phép năm ${run}` }).click()
  await page.getByRole('textbox', { name: /^Từ ngày/ }).fill('10/08/2026')
  await page.getByRole('textbox', { name: /^Đến ngày/ }).fill('14/08/2026')
  // A date field hands its value to the form on blur, so move off it before reading the count.
  await page.getByRole('textbox', { name: /^Lý do/ }).click()
  await expect(page.getByText('Khoảng ngày đã chọn có 5 ngày làm việc.')).toBeVisible()
  const days = page.getByRole('textbox', { name: /^Số ngày/ })
  await expect(days).toHaveValue('5')
  await page.getByRole('checkbox', { name: 'Ngày cuối chỉ nghỉ buổi sáng' }).check()
  await expect(days).toHaveValue('4,5')
  await page.getByRole('button', { name: 'Tạo bản nháp' }).click()
  await expect(page.getByRole('heading', { name: /^NP-2026-/ })).toBeVisible()
  await expect(page.getByRole('main').getByText('4,5 ngày làm việc').first()).toBeVisible()

  // One day: a single half-day box.
  await page.goto('/me/requests/leave/new')
  await page.getByRole('textbox', { name: /^Từ ngày/ }).fill('17/08/2026')
  await page.getByRole('textbox', { name: /^Đến ngày/ }).fill('17/08/2026')
  await page.getByRole('textbox', { name: /^Lý do/ }).click()
  await expect(page.getByRole('checkbox', { name: 'Nghỉ nửa ngày' })).toBeVisible()
  await expect(page.getByRole('checkbox', { name: 'Ngày cuối chỉ nghỉ buổi sáng' })).toHaveCount(0)

  // A range of nothing but rest days has no leave to take.
  await page.getByRole('combobox', { name: 'Loại nghỉ' }).click()
  await page.getByRole('option', { name: `Phép năm ${run}` }).click()
  await page.getByRole('textbox', { name: /^Từ ngày/ }).fill('08/08/2026')
  await page.getByRole('textbox', { name: /^Đến ngày/ }).fill('09/08/2026')
  await page.getByRole('textbox', { name: /^Lý do/ }).click()
  await expect(page.getByText('Khoảng ngày đã chọn có 0 ngày làm việc.')).toBeVisible()
  await page.getByRole('button', { name: 'Tạo bản nháp' }).click()
  await expect(page.getByText(/đều là ngày nghỉ hoặc ngày lễ/)).toBeVisible()
})

// A setting whose change has side effects says so where it is changed.
test('the accrual setting explains what switching it does', async ({ page }) => {
  await signInAs(page, admin.login, '/admin/settings')
  await page.getByRole('button', { name: 'Cộng phép năm' }).click()
  await expect(page.getByRole('dialog').getByText(/chồng lên số đã cấp tay/)).toBeVisible()
})
