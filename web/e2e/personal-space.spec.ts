import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { created, createUser, employee, password, signedInApi, runId } from './api'

// One run: two companies, each with one employee under contract and a posted March payroll.
// The first employee is linked to a no-role account and has a draft April timesheet.
const run = runId()
const me = `ps_me_${run}`
let api: APIRequestContext
let payroll: number
let otherPayroll: number
let net: number

async function ok(res: Promise<{ status(): number; text(): Promise<string> }>) {
  const r = await res
  expect([200, 204], await r.text()).toContain(r.status())
}

// company builds a company, one employee under contract, a posted March timesheet and a
// posted payroll; it returns the unit, the employee and the payroll.
async function company(name: string, userLogin: string | null) {
  const c = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name } }))
  const unit = await created(api.post('/api/org-units', { data: { parent_id: c, kind: 'department', name: `${name} P` } }))
  const emp = await created(api.post('/api/hrm/employees', { data: employee(`PS${run}${userLogin ? 'A' : 'B'}`, unit, { user_login: userLogin }) }))
  const kind = await created(api.post('/api/hrm/contract-types', { data: { name: `Không thời hạn ${name}`, fixed_term: false, active: true } }))
  const contract = await created(
    api.post('/api/hrm/contracts', { data: { employee_id: emp, contract_type_id: kind, start_date: '2026-01-01', end_date: null, terms: { salary: 22_000_000, lines: [] } } }),
  )
  await ok(api.post(`/api/documents/hrm.contract/${contract}/transitions`, { data: { to: 'posted', version: 1 } }))
  const ts = await created(api.post('/api/hrm/timesheets', { data: { org_unit_id: unit, month: '2026-03' } }))
  const lines = []
  for (let d = 1; d <= 31; d++) {
    const date = `2026-03-${String(d).padStart(2, '0')}`
    const wd = new Date(`${date}T00:00:00Z`).getUTCDay()
    if (wd !== 0 && wd !== 6) lines.push({ employee_id: emp, date, days: '1' })
  }
  await ok(api.put(`/api/hrm/timesheets/${ts}`, { data: { version: 1, month: '2026-03', lines } }))
  await ok(api.post(`/api/documents/hrm.timesheet/${ts}/transitions`, { data: { to: 'posted', version: 2 } }))
  const { id, job_id } = await (await api.post('/api/hrm/payrolls', { data: { legal_entity_id: c, month: '2026-03' } })).json()
  await expect.poll(async () => (await (await api.get(`/api/jobs/${job_id}`)).json()).state, { timeout: 30_000 }).toBe('completed')
  const doc = await (await api.get(`/api/hrm/payrolls/${id}`)).json()
  await ok(api.post(`/api/documents/hrm.payroll/${id}/transitions`, { data: { to: 'posted', version: doc.version } }))
  return { unit, emp, payroll: id as number, net: doc.lines[0].net as number }
}

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  // Payrolls post at once here (m6 sets its own rule again).
  await api.delete('/api/approval-rules/hrm.payroll')
  await createUser(api, me, [])
  const mine = await company(`Công ty PS ${run}`, me)
  payroll = mine.payroll
  net = mine.net
  otherPayroll = (await company(`Công ty PS khác ${run}`, null)).payroll
  // A draft April timesheet: the worked day is not closed.
  const ts = await created(api.post('/api/hrm/timesheets', { data: { org_unit_id: mine.unit, month: '2026-04' } }))
  await ok(api.put(`/api/hrm/timesheets/${ts}`, { data: { version: 1, month: '2026-04', lines: [{ employee_id: mine.emp, date: '2026-04-15', days: '1' }] } }))
})

async function signInAs(page: Page, url: string) {
  await page.goto(url)
  await page.getByLabel('Tên đăng nhập').fill(me)
  await page.getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: /^Tài khoản:/ })).toBeVisible()
}

const overflow = (page: Page) => () => page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth)
const money = (n: number) => new Intl.NumberFormat('vi-VN').format(n)

test('the personal space: tile, days, payslip with its PDF, and another payroll is not found', async ({ page }) => {
  await signInAs(page, '/')
  const main = page.getByRole('main')
  await expect(main.getByRole('link', { name: 'Cá nhân' })).toBeVisible()
  await expect(main.getByRole('link')).toHaveCount(1)

  await page.goto('/me/days?mode=month&date=2026-04-01')
  await expect(page.getByRole('heading', { name: 'Chấm công' })).toBeVisible()
  await expect(page.getByText('chưa chốt', { exact: true })).toBeVisible()

  await page.goto('/me/payslips')
  await expect(page.getByRole('heading', { name: 'Phiếu lương' })).toBeVisible()
  await expect(page.getByRole('main')).toContainText(money(net))
  await page.getByRole('link', { name: /3/ }).first().click()
  await expect(page).toHaveURL(new RegExp(`/me/payslips/${payroll}$`))
  await expect(page.getByRole('main')).toContainText(money(net))

  await page.getByRole('button', { name: 'In phiếu lương' }).click()
  const download = page.waitForEvent('download')
  await page.getByRole('link', { name: 'Tải file' }).click()
  expect((await download).suggestedFilename()).toMatch(/\.pdf$/)

  await page.goto(`/me/payslips/${otherPayroll}`)
  await expect(page.getByText(/Không tìm thấy/i).first()).toBeVisible()
})

test('on a phone the days and the payslip do not overflow sideways', async ({ browser }) => {
  const ctx = await browser.newContext({ viewport: { width: 375, height: 800 }, locale: 'vi-VN' })
  const page = await ctx.newPage()
  await signInAs(page, '/me/days?mode=month&date=2026-04-01')
  await expect(page.getByText('chưa chốt', { exact: true })).toBeVisible()
  await expect.poll(overflow(page)).toBe(false)
  await page.goto(`/me/payslips/${payroll}`)
  await expect(page.getByRole('main')).toContainText(money(net))
  await expect.poll(overflow(page)).toBe(false)
})

test('with no product enabled the tile and payslips still read, and there is no request button', async ({ page }) => {
  await signInAs(page, 'http://localhost:8091/')
  await expect(page.getByRole('main').getByRole('link', { name: 'Cá nhân' })).toBeVisible()
  await page.goto('http://localhost:8091/me/payslips')
  await expect(page.getByRole('main')).toContainText(money(net))
  await page.goto('http://localhost:8091/me/requests')
  await expect(page.getByRole('heading', { name: 'Nghỉ phép & tăng ca' })).toBeVisible()
  await expect(page.getByRole('link', { name: /^Tạo đơn/ })).toHaveCount(0)
})

test('without an HRM role the HRM area sends home; a direct manager sees their report’s request in the team list', async ({ page, browser, baseURL }) => {
  await signInAs(page, '/hrm/leaves')
  await expect(page).toHaveURL(/\/$/)

  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty PS nhóm ${run}` } }))
  const unit = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng PS nhóm ${run}` } }))
  const login = { boss: `ps_boss_${run}`, staff: `ps_staff_${run}` }
  for (const l of [login.boss, login.staff]) await createUser(api, l, [])
  const boss = await created(api.post('/api/hrm/employees', { data: employee(`PSM${run}`, unit, { user_login: login.boss }) }))
  await created(api.post('/api/hrm/employees', { data: employee(`PSS${run}`, unit, { user_login: login.staff, manager_id: boss }) }))
  const type = await created(api.post('/api/hrm/leave-types', { data: { name: `Nghỉ PS ${run}`, deducts_balance: false, paid: true, active: true } }))
  const asStaff = await signedInApi(baseURL!, login.staff, password)
  const leave = await created(asStaff.post('/api/hrm/leaves', { data: { leave_type_id: type, start_date: '2026-06-03', end_date: '2026-06-03', days: '1', reason: null } }))
  const { number } = (await (await asStaff.get(`/api/hrm/leaves/${leave}`)).json()) as { number: string }

  const m = await (await browser.newContext()).newPage()
  await m.goto('/me/team/leaves')
  await m.getByLabel('Tên đăng nhập').fill(login.boss)
  await m.getByLabel(/^Mật khẩu/).fill(password)
  await m.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(m.getByRole('heading', { name: 'Đơn nghỉ phép của nhóm' })).toBeVisible()
  await expect(m.getByRole('navigation').getByRole('group', { name: 'Nhóm của tôi' }).getByRole('link', { name: 'Đơn nghỉ phép' })).toBeVisible()
  await expect(m.getByRole('main').getByRole('link', { name: number })).toHaveAttribute('href', `/me/requests/leave/${leave}`)
})
