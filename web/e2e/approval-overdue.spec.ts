import { expect, test, type APIRequestContext, type Browser } from '@playwright/test'
import { created, createUser, employee, password, signedInApi, runId } from './api'
import { admin } from './helpers'

// One run: an employee whose manager approves leave within a due time. The leave rule is
// tenant-wide, so this file runs alone and puts back what it found.
const run = runId()
const login = { e: `aoe_${run}`, manager: `aomgr_${run}` }
const url = '/hrm/approval-rules/hrm.leave_request'
let api: APIRequestContext
let original: unknown = null
let leaveType: number

// Dates are the tenant's, not the runner's: near midnight the two differ by a day.
const tenantToday = () => new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Ho_Chi_Minh' }).format(new Date())
const addDays = (iso: string, n: number) => new Date(Date.parse(`${iso}T00:00:00Z`) + n * 86_400_000).toISOString().slice(0, 10)
// workdayAhead is the first Monday to Friday at least n days after the tenant's today.
const workdayAhead = (n: number) => {
  let d = addDays(tenantToday(), n)
  while ([0, 6].includes(new Date(`${d}T00:00:00Z`).getUTCDay())) d = addDays(d, 1)
  return d
}

const moduleStep = (due?: number) => ({ step: { approver: { kind: 'module' }, approvals: 1, rejections: 1, ...(due ? { due_hours: due } : {}) } })
const putRule = async (due?: number) => {
  const res = await api.put('/api/approval-rules/hrm.leave_request', { data: { flow: [moduleStep(due)], max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' } })
  expect(res.status(), await res.text()).toBe(204)
}

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  const rules = (await (await api.get('/api/approval-rules?product=hrm')).json()) as { doc_type: string; action: string; rule: unknown }[]
  original = rules.find((r) => r.doc_type === 'hrm.leave_request' && r.action === 'post')?.rule ?? null
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty AO ${run}` } }))
  const unit = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng AO ${run}` } }))
  for (const l of [login.e, login.manager]) await createUser(api, l, [])
  const manager = await created(api.post('/api/hrm/employees', { data: employee(`AOM${run}`, unit, { user_login: login.manager }) }))
  await created(api.post('/api/hrm/employees', { data: employee(`AOE${run}`, unit, { user_login: login.e, manager_id: manager }) }))
  leaveType = await created(api.post('/api/hrm/leave-types', { data: { name: `Nghỉ AO ${run}`, deducts_balance: false, paid: true, active: true } }))
})

test.afterAll(async () => {
  // beforeAll may have failed before `api` was signed in; nothing to restore then.
  if (!api) return
  const res = original ? await api.put('/api/approval-rules/hrm.leave_request', { data: original }) : await api.delete('/api/approval-rules/hrm.leave_request')
  expect(res.status(), await res.text()).toBe(204)
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

test('an administrator gives a step a due time and it stays after reload', async ({ browser }) => {
  await putRule()
  const page = await as(browser, admin.login, url)
  await page.getByRole('main').getByRole('button', { name: 'Bước duyệt: Quản lý trực tiếp' }).click()
  const due = page.getByRole('complementary').getByRole('textbox', { name: /^Hạn duyệt \(giờ\)/ })
  await expect(due).toHaveValue('')
  await due.fill('24')
  await page.getByRole('button', { name: 'Lưu' }).click()
  await expect(page.getByText('Đã lưu quy trình duyệt.')).toBeVisible()
  await page.reload()
  // The box reads the due time without opening it.
  await page.getByRole('main').getByRole('button', { name: 'Bước duyệt: Quản lý trực tiếp' }).click()
  await expect(page.getByRole('main').getByText('hạn 24 giờ')).toBeVisible()
  await expect(page.getByRole('complementary').getByRole('textbox', { name: /^Hạn duyệt \(giờ\)/ })).toHaveValue('24')
  const rules = (await (await api.get('/api/approval-rules?product=hrm')).json()) as { doc_type: string; action: string; rule: { flow: unknown[] } | null }[]
  expect(rules.find((r) => r.doc_type === 'hrm.leave_request' && r.action === 'post')?.rule?.flow).toEqual([moduleStep(24)])
})

test('a waiting step shows when it is due', async ({ baseURL, browser }) => {
  await putRule(24)
  const asE = await signedInApi(baseURL!, login.e, password)
  const day = workdayAhead(5)
  const leave = await created(asE.post('/api/hrm/leaves', { data: { leave_type_id: leaveType, start_date: day, end_date: day, days: '1', reason: null } }))
  const sent = await asE.post(`/api/documents/hrm.leave_request/${leave}/transitions`, { data: { to: 'posted', version: 1 } })
  expect(sent.status(), await sent.text()).toBe(200)

  const m = await as(browser, login.manager, `/hrm/leaves/${leave}`)
  await expect(m.getByText(/^Hạn: /)).toBeVisible()
  await expect(m.getByText('Quá hạn', { exact: true })).toHaveCount(0)
})
