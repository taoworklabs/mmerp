import { expect, test, type APIRequestContext, type Browser } from '@playwright/test'
import { created, createUser, employee, password, signedInApi } from './api'

// One run: a company of its own with department P and employee M6E01 under contract, March
// timesheet posted. pay runs payrolls; dir approves them.
const run = Date.now().toString(36)
const company = `Công ty M6 ${run}`
const login = { pay: `m6pay_${run}`, dir: `m6dir_${run}` }
let api: APIRequestContext
let emp: number

async function ok(res: Promise<{ status(): number; text(): Promise<string> }>) {
  const r = await res
  expect([200, 204], await r.text()).toContain(r.status())
}

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  const c = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: company } }))
  const p = await created(api.post('/api/org-units', { data: { parent_id: c, kind: 'department', name: `Phòng P ${run}` } }))
  emp = await created(api.post('/api/hrm/employees', { data: employee(`M6E${run}`, p, { hire_date: '2026-01-05' }) }))
  await createUser(api, login.pay, [
    { role: 'hrm.payroll', unit: c },
    { role: 'hrm.hr', unit: c },
  ])
  const dir = await createUser(api, login.dir, [{ role: 'hrm.payroll', unit: null }])
  // Contracts, timesheets and overtime post at once here; payrolls go to dir.
  for (const type of ['hrm.contract', 'hrm.timesheet', 'hrm.overtime_request']) await api.delete(`/api/approval-rules/${type}`)
  await ok(api.put('/api/approval-rules/hrm.payroll', { data: { steps: [{ approver: { kind: 'user', user_id: dir } }], max_levels: 1, fallback_product: 'hrm', fallback_role: 'payroll' } }))

  const kind = await created(api.post('/api/hrm/contract-types', { data: { name: `Không thời hạn ${run}`, fixed_term: false, active: true } }))
  const contract = await created(
    api.post('/api/hrm/contracts', { data: { employee_id: emp, contract_type_id: kind, start_date: '2026-01-01', end_date: null, terms: { salary: 22_000_000, lines: [] } } }),
  )
  await ok(api.post(`/api/documents/hrm.contract/${contract}/transitions`, { data: { to: 'posted', version: 1 } }))
  const ts = await created(api.post('/api/hrm/timesheets', { data: { org_unit_id: p, month: '2026-03' } }))
  const lines = []
  for (let d = 1; d <= 31; d++) {
    const date = `2026-03-${String(d).padStart(2, '0')}`
    const wd = new Date(`${date}T00:00:00Z`).getUTCDay()
    if (wd !== 0 && wd !== 6) lines.push({ employee_id: emp, date, days: '1' })
  }
  await ok(api.put(`/api/hrm/timesheets/${ts}`, { data: { version: 1, month: '2026-03', lines } }))
  await ok(api.post(`/api/documents/hrm.timesheet/${ts}/transitions`, { data: { to: 'posted', version: 2 } }))
})

async function as(browser: Browser, user: string, url: string) {
  const page = await (await browser.newContext()).newPage()
  await page.goto(url)
  await page.getByLabel('Tên đăng nhập').fill(user)
  await page.getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: 'Tài khoản' })).toBeVisible()
  return page
}

test('payroll: computed by a job, refused while sources changed, computed again, approved, exported', async ({ browser }) => {
  const pay = await as(browser, login.pay, '/hrm/payrolls')
  await pay.getByRole('button', { name: 'Lập bảng lương' }).click()
  const create = pay.getByRole('dialog')
  await create.getByRole('combobox', { name: 'Pháp nhân' }).click()
  await pay.getByRole('option', { name: company }).click()
  await create.getByRole('button', { name: /^Kỳ/ }).click()
  await pay.getByRole('button', { name: 'Th03' }).click()
  await create.getByRole('button', { name: 'Lập và tính lương' }).click()
  await expect(pay.getByRole('heading', { name: /^BL-2026-/ })).toBeVisible()
  const lines = pay.getByRole('table', { name: 'Lương từng nhân viên' })
  await expect(lines.getByRole('row', { name: new RegExp(`^M6E${run}`) })).toContainText('22.000.000')
  const number = (await pay.getByRole('heading', { name: /^BL-2026-/ }).textContent())!

  // An overtime request approved since: warned, and sending is refused until computed again.
  const ot = await created(api.post('/api/hrm/overtimes', { data: { employee_id: emp, date: '2026-03-16', day_kind: 'weekday', day_hours: '2', night_hours: '0', reason: null } }))
  await ok(api.post(`/api/documents/hrm.overtime_request/${ot}/transitions`, { data: { to: 'posted', version: 1 } }))
  await pay.reload()
  await expect(pay.getByText('Dữ liệu nguồn đã thay đổi', { exact: true })).toBeVisible()
  await pay.getByRole('button', { name: 'Gửi duyệt' }).click()
  await expect(pay.getByRole('alert').filter({ hasText: 'rồi gửi lại' })).toBeVisible()
  await pay.getByRole('button', { name: 'Tính lại' }).click()
  await expect(pay.getByText('Dữ liệu nguồn đã thay đổi', { exact: true })).toHaveCount(0)
  await expect(lines.getByRole('row', { name: new RegExp(`^M6E${run}`) })).toContainText('375.000')

  await pay.getByRole('button', { name: 'Gửi duyệt' }).click()
  await expect(pay.getByRole('main').getByText('Chờ duyệt', { exact: true }).first()).toBeVisible()
  const dir = await as(browser, login.dir, '/inbox')
  await dir.getByRole('button', { name: new RegExp(number) }).click()
  await expect(dir.getByText('Tổng chi phí lương')).toBeVisible()
  await dir.getByRole('button', { name: 'Duyệt', exact: true }).click()
  await expect(dir.getByText('Đã duyệt.')).toBeVisible()
  await pay.reload()
  await expect(pay.getByRole('main').getByText('Đã chốt', { exact: true }).first()).toBeVisible()

  await pay.getByRole('button', { name: 'Xuất Excel' }).click()
  const download = pay.waitForEvent('download')
  await pay.getByRole('link', { name: 'Tải file' }).click()
  expect((await download).suggestedFilename()).toBe(`${number}.xlsx`)
})
