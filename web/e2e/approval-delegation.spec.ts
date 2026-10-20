import { expect, test, type APIRequestContext, type Browser } from '@playwright/test'
import { created, createUser, employee, password, signedInApi, runId, toFlow } from './api'
import { admin } from './helpers'

// One run: an employee whose manager is away, and a colleague of HR who approves in the
// manager's place. The leave request's rule is tenant-wide, so this file runs alone and puts back
// what it found.
const run = runId()
const login = { e: `ade_${run}`, manager: `admgr_${run}`, colleague: `adcol_${run}`, other: `adoth_${run}` }
let api: APIRequestContext
let original: unknown = null
let leaveType: number

// Dates are the tenant's, not the runner's: near midnight the two differ by a day.
const tenantToday = () => new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Ho_Chi_Minh' }).format(new Date())
const addDays = (iso: string, n: number) => new Date(Date.parse(`${iso}T00:00:00Z`) + n * 86_400_000).toISOString().slice(0, 10)
const vn = (iso: string) => iso.split('-').reverse().join('/')
const inDays = (n: number) => vn(addDays(tenantToday(), n))
// workdayAhead is the first Monday to Friday at least n days after the tenant's today.
const workdayAhead = (n: number) => {
  let d = addDays(tenantToday(), n)
  while ([0, 6].includes(new Date(`${d}T00:00:00Z`).getUTCDay())) d = addDays(d, 1)
  return d
}

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  const rules = (await (await api.get('/api/approval-rules?product=hrm')).json()) as { doc_type: string; rule: unknown }[]
  original = rules.find((r) => r.doc_type === 'hrm.leave_request')?.rule ?? null
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty AD ${run}` } }))
  const unit = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng AD ${run}` } }))
  for (const l of [login.e, login.manager, login.other]) await createUser(api, l, [])
  // A delegate votes only on what they may view: HR of the unit sees its leave requests.
  await createUser(api, login.colleague, [{ role: 'hrm.hr', unit }])
  const manager = await created(api.post('/api/hrm/employees', { data: employee(`ADM${run}`, unit, { user_login: login.manager }) }))
  await created(api.post('/api/hrm/employees', { data: employee(`ADE${run}`, unit, { user_login: login.e, manager_id: manager }) }))
  const rule = await api.put('/api/approval-rules/hrm.leave_request', {
    data: { flow: toFlow([{ approver: { kind: 'module' } }]), max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' },
  })
  expect(rule.status(), await rule.text()).toBe(204)
  leaveType = await created(api.post('/api/hrm/leave-types', { data: { name: `Nghỉ AD ${run}`, deducts_balance: false, paid: true, active: true } }))
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
  await expect(page.getByRole('button', { name: 'Tài khoản' })).toBeVisible()
  return page
}

test('an absent manager’s delegate approves a leave request', async ({ browser, baseURL }) => {
  // The manager reaches delegations from the inbox and delegates for the next days.
  const m = await as(browser, login.manager, '/inbox')
  await m.getByRole('link', { name: 'Ủy quyền duyệt' }).click()
  await expect(m).toHaveURL(/\/delegations$/)
  await m.getByRole('button', { name: 'Tạo ủy quyền' }).click()
  const dialog = m.getByRole('dialog')
  // Only a manager of everyone's delegations picks whom it is for.
  await expect(dialog.getByRole('combobox', { name: 'Người ủy quyền' })).toHaveCount(0)
  await dialog.getByRole('combobox', { name: /^Người được ủy quyền/ }).click()
  await m.getByRole('option', { name: login.colleague, exact: true }).click()
  await dialog.getByRole('textbox', { name: /^Đến ngày/ }).fill(inDays(3))
  await dialog.getByRole('button', { name: 'Tạo ủy quyền' }).click()
  await expect(m.getByText('Đã tạo ủy quyền.')).toBeVisible()
  await expect(m.getByRole('row', { name: new RegExp(`${login.colleague}.*Đang hiệu lực`) })).toBeVisible()

  // The employee sends a leave request.
  const asE = await signedInApi(baseURL!, login.e, password)
  const day = workdayAhead(5)
  const leave = await created(asE.post('/api/hrm/leaves', { data: { leave_type_id: leaveType, start_date: day, end_date: day, days: '1', reason: null } }))
  const sent = await asE.post(`/api/documents/hrm.leave_request/${leave}/transitions`, { data: { to: 'posted', version: 1 } })
  expect(sent.status(), await sent.text()).toBe(200)

  // The colleague finds it in their inbox, marked as for the manager, and approves it.
  const c = await as(browser, login.colleague, '/inbox')
  const row = c.getByRole('button', { name: new RegExp(`${login.e}.*thay ${login.manager}`) })
  await row.click()
  await c.getByRole('button', { name: 'Duyệt', exact: true }).click()
  await expect(c.getByText('Đã duyệt.')).toBeVisible()

  // The panel, in the leave page's side column, says who voted and in whose place.
  const e = await as(browser, login.e, `/hrm/leaves/${leave}`)
  await expect(e.getByText(`${login.colleague} thay ${login.manager}`)).toBeVisible()
})

test('the delegation form refuses delegating to oneself', async ({ browser }) => {
  const page = await as(browser, login.other, '/delegations')
  await page.getByRole('button', { name: 'Tạo ủy quyền' }).first().click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('combobox', { name: /^Người được ủy quyền/ }).click()
  await page.getByRole('option', { name: login.other, exact: true }).click()
  await dialog.getByRole('textbox', { name: /^Đến ngày/ }).fill(inDays(2))
  await dialog.getByRole('button', { name: 'Tạo ủy quyền' }).click()
  await expect(dialog.getByRole('combobox', { name: /^Người được ủy quyền/ })).toHaveAccessibleDescription(/Phải khác người ủy quyền\./)
})

test('an administrator creates a delegation for another user', async ({ browser }) => {
  const page = await as(browser, admin.login, '/delegations')
  await page.getByRole('button', { name: 'Tạo ủy quyền' }).first().click()
  const dialog = page.getByRole('dialog')
  // Managers see logins next to names.
  await dialog.getByRole('combobox', { name: 'Người ủy quyền' }).click()
  await page.getByRole('option', { name: `${login.other} (${login.other})`, exact: true }).click()
  await dialog.getByRole('combobox', { name: /^Người được ủy quyền/ }).click()
  await page.getByRole('option', { name: `${login.e} (${login.e})`, exact: true }).click()
  await dialog.getByRole('textbox', { name: /^Từ ngày/ }).fill(inDays(1))
  await dialog.getByRole('textbox', { name: /^Đến ngày/ }).fill(inDays(4))
  await dialog.getByRole('button', { name: 'Tạo ủy quyền' }).click()
  await expect(page.getByText('Đã tạo ủy quyền.')).toBeVisible()
  const row = page.getByRole('row', { name: new RegExp(`${login.other}.*${login.e}.*Sắp tới`) })
  await expect(row).toBeVisible()

  // A manager's confirm names whose delegation it is; cancelled here, the other user deletes it.
  await row.getByRole('button', { name: 'Thao tác khác' }).click()
  await page.getByRole('menuitem', { name: 'Xóa ủy quyền' }).click()
  await expect(page.getByRole('dialog').getByText(`Xóa ủy quyền của ${login.other} cho ${login.e}`)).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Quay lại' }).click()

  // The other user sees it as their own, and may take it back.
  const other = await as(browser, login.other, '/delegations')
  const own = other.getByRole('row', { name: new RegExp(`${login.e}.*Sắp tới`) })
  await own.getByRole('button', { name: 'Thao tác khác' }).click()
  await other.getByRole('menuitem', { name: 'Xóa ủy quyền' }).click()
  await other.getByRole('dialog').getByRole('button', { name: 'Xóa ủy quyền' }).click()
  await expect(other.getByText('Đã xóa ủy quyền.')).toBeVisible()
  await expect(own).toHaveCount(0)
})
