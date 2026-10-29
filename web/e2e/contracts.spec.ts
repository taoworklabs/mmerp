import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { created, createUser, employee, password, signedInApi, runId, toFlow } from './api'
import { admin, as, status } from './helpers'

// One run: department Q with manager M; employees E, E2 and E3 report to M. HR users of Q: hr
// (no salaries) and pay (with the payroll role); dir approves contracts.
const run = runId()
const login = { m: `ctm_${run}`, e: `cte_${run}`, hr: `cthr_${run}`, pay: `ctpay_${run}`, dir: `ctdir_${run}` }
const kinds = { open: `Không xác định thời hạn ${run}`, fixed: `Có thời hạn 12 tháng ${run}` }
let ids: { q: number; m: number; e: number; e2: number; e3: number; open: number; fixed: number }
let api: APIRequestContext
let dir: APIRequestContext

const iso = (d: Date) => d.toISOString().slice(0, 10)
const inDays = (n: number) => iso(new Date(Date.now() + n * 86_400_000))
const vn = (s: string) => s.split('-').reverse().join('/')

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty CT ${run}` } }))
  const q = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng Q ${run}` } }))
  for (const l of [login.m, login.e]) await createUser(api, l, [])
  await createUser(api, login.hr, [{ role: 'hrm.hr', unit: q }])
  await createUser(api, login.pay, [
    { role: 'hrm.hr', unit: q },
    { role: 'hrm.payroll', unit: q },
  ])
  const dirId = await createUser(api, login.dir, [
    { role: 'hrm.hr', unit: null },
    { role: 'hrm.payroll', unit: null },
  ])
  dir = await signedInApi(baseURL!, login.dir, password)
  const m = await created(api.post('/api/hrm/employees', { data: employee(`CTM${run}`, q, { user_login: login.m }) }))
  const e = await created(api.post('/api/hrm/employees', { data: employee(`CTE${run}`, q, { user_login: login.e, manager_id: m }) }))
  const e2 = await created(api.post('/api/hrm/employees', { data: employee(`CTF${run}`, q, { manager_id: m }) }))
  const e3 = await created(api.post('/api/hrm/employees', { data: employee(`CTG${run}`, q, { manager_id: m }) }))
  const open = await created(api.post('/api/hrm/contract-types', { data: { name: kinds.open, fixed_term: false, active: true } }))
  const fixed = await created(api.post('/api/hrm/contract-types', { data: { name: kinds.fixed, fixed_term: true, active: true } }))
  const rule = await api.put('/api/approval-rules/hrm.contract', {
    data: { flow: toFlow([{ approver: { kind: 'user', user_ids: [dirId] } }]), max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' },
  })
  expect(rule.status(), await rule.text()).toBe(204)
  ids = { q, m, e, e2, e3, open, fixed }
})

const heading = (page: Page, prefix: string) => page.getByRole('heading', { name: new RegExp(`^${prefix}-\\d{4}-`) })

// approveAsDir approves the step waiting for dir on document id.
async function approveAsDir(id: number) {
  const inbox = (await (await dir.get('/api/approvals/inbox')).json()) as { items: { instance_id: number; doc_id: number; step: number }[] }
  const item = inbox.items.find((i) => i.doc_id === id)!
  const res = await dir.post(`/api/approvals/${item.instance_id}/approve`, { data: { step: item.step } })
  expect(res.status(), await res.text()).toBe(204)
}

// contractViaApi adds an original of employee: approved by dir, or only a draft with send false.
async function contractViaApi(employeeId: number, kind: number, start: string, end: string | null, send = true) {
  const id = await created(
    api.post('/api/hrm/contracts', {
      data: { employee_id: employeeId, contract_type_id: kind, start_date: start, end_date: end, terms: { salary: 9_000_000, lines: [] } },
    }),
  )
  if (!send) return id
  const res = await api.post(`/api/documents/hrm.contract/${id}/transitions`, { data: { to: 'posted', version: 1 } })
  expect(res.status(), await res.text()).toBe(200)
  await approveAsDir(id)
  return id
}

test.describe('contract kinds', () => {
  test('HR adds a fixed-term kind and an open-ended one, and retires a kind', async ({ browser }) => {
    const page = await as(browser, admin.login, '/hrm/contract-types')
    const add = async (name: string, fixed: boolean) => {
      await page.getByRole('button', { name: 'Thêm loại hợp đồng' }).click()
      const dialog = page.getByRole('dialog')
      await dialog.getByRole('textbox', { name: /^Tên loại hợp đồng/ }).fill(name)
      if (fixed) await dialog.getByRole('checkbox', { name: 'Có thời hạn' }).check()
      await dialog.getByRole('button', { name: 'Lưu' }).click()
      await expect(dialog).toBeHidden()
    }
    await add(`Thử việc ${run}`, true)
    await add(`Mùa vụ ${run}`, false)
    const table = page.getByRole('table', { name: 'Loại hợp đồng' })
    await expect(table.getByRole('row', { name: new RegExp(`Thử việc ${run}.*Có thời hạn.*Đang dùng`) })).toBeVisible()
    await expect(table.getByRole('row', { name: new RegExp(`Mùa vụ ${run}.*Không thời hạn`) })).toBeVisible()

    // A settings row opens its edit from the row itself.
    await table.getByRole('button', { name: `Mùa vụ ${run}` }).click()
    await page.getByRole('dialog').getByRole('checkbox', { name: 'Đang dùng' }).uncheck()
    await page.getByRole('dialog').getByRole('button', { name: 'Lưu' }).click()
    await expect(table.getByRole('row', { name: new RegExp(`Mùa vụ ${run}.*Ngừng dùng`) })).toBeVisible()

    // A retired kind is no longer offered on a new contract.
    await page.goto(`/hrm/contracts/new?employee=${ids.e2}`)
    await page.getByRole('combobox', { name: 'Loại hợp đồng' }).click()
    await expect(page.getByRole('option', { name: `Thử việc ${run}` })).toBeVisible()
    await expect(page.getByRole('option', { name: `Mùa vụ ${run}` })).toHaveCount(0)
  })
})

test.describe('contracts', () => {
  test('the kind decides the end date; the director approves from the inbox and sees the salary', async ({ browser }) => {
    const pay = await as(browser, login.pay, `/hrm/employees/${ids.e}?tab=contracts`)
    await pay.getByRole('link', { name: 'Hợp đồng mới' }).click()
    const kind = pay.getByRole('combobox', { name: 'Loại hợp đồng' })
    const end = pay.getByRole('textbox', { name: /^Ngày hết hạn/ })
    await kind.click()
    await pay.getByRole('option', { name: kinds.open, exact: true }).click()
    await expect(end).toHaveCount(0)
    await kind.click()
    await pay.getByRole('option', { name: kinds.fixed, exact: true }).click()
    await expect(end).toBeVisible()

    await pay.getByRole('textbox', { name: /^Ngày hiệu lực/ }).fill('01/01/2026')
    await pay.getByRole('textbox', { name: /^Lương/ }).fill('15000000')
    await pay.getByRole('button', { name: 'Thêm khoản' }).click()
    await pay.getByRole('textbox', { name: /^Tên khoản 1/ }).fill('Phụ cấp trách nhiệm')
    await pay.getByRole('textbox', { name: /^Số tiền khoản 1/ }).fill('2000000')
    // Without the end date a fixed-term contract is not saved.
    await pay.getByRole('button', { name: 'Tạo bản nháp' }).click()
    await expect(heading(pay, 'HD')).toHaveCount(0)
    await end.fill('31/12/2026')
    await pay.getByRole('button', { name: 'Tạo bản nháp' }).click()
    await expect(heading(pay, 'HD')).toBeVisible()
    const number = (await heading(pay, 'HD').textContent())!
    await pay.getByRole('button', { name: 'Gửi duyệt' }).click()
    await status(pay, 'Chờ duyệt')

    const d = await as(browser, login.dir, '/inbox')
    await d.getByRole('button', { name: new RegExp(number) }).click()
    await expect(d.getByText(/15\.000\.000/)).toBeVisible()
    await expect(d.getByText('Phụ cấp trách nhiệm')).toBeVisible()
    await d.getByRole('button', { name: 'Duyệt', exact: true }).click()
    await expect(d.getByText('Đã duyệt.')).toBeVisible()

    await pay.reload()
    await status(pay, 'Có hiệu lực')
  })

  test('an appendix from the row menu starts from the original; the original stays while it has one', async ({ browser }) => {
    const original = await contractViaApi(ids.e2, ids.open, '2025-01-01', null)
    const pay = await as(browser, login.pay, `/hrm/employees/${ids.e2}?tab=contracts`)
    const table = pay.getByRole('table', { name: 'Hợp đồng' })
    await table.getByRole('row').filter({ hasText: kinds.open }).getByRole('button', { name: 'Thao tác khác' }).click()
    await pay.getByRole('menuitem', { name: 'Thêm phụ lục' }).click()
    await expect(pay.getByRole('textbox', { name: /^Lương/ })).toHaveValue('9.000.000')
    await expect(pay.getByRole('textbox', { name: /^Ngày hết hạn/ })).toHaveCount(0)
    await pay.getByRole('textbox', { name: /^Ngày hiệu lực/ }).fill('01/07/2026')
    await pay.getByRole('textbox', { name: /^Lương/ }).fill('11000000')
    await pay.getByRole('button', { name: 'Tạo bản nháp' }).click()
    await expect(heading(pay, 'HD')).toBeVisible()
    await expect(pay.getByText(/Phụ lục của HD-2025-/).first()).toBeVisible()
    await pay.getByRole('button', { name: 'Gửi duyệt' }).click()
    await status(pay, 'Chờ duyệt')
    await approveAsDir(Number(new URL(pay.url()).pathname.split('/').pop()))

    // The appendix row has nothing to add.
    await pay.goto(`/hrm/employees/${ids.e2}?tab=contracts`)
    await expect(table.getByRole('row').filter({ hasText: 'Phụ lục của' })).toBeVisible()
    await expect(table.getByRole('row').filter({ hasText: 'Phụ lục của' }).getByRole('button', { name: 'Thao tác khác' })).toHaveCount(0)

    await pay.goto(`/hrm/contracts/${original}`)
    await pay.getByRole('button', { name: 'Hủy chứng từ' }).click()
    await pay.getByRole('dialog').getByRole('button', { name: 'Hủy chứng từ' }).click()
    await expect(pay.getByText('Hợp đồng còn phụ lục. Hủy hoặc xóa các phụ lục trước.')).toBeVisible()
    await pay.reload()
    await status(pay, 'Có hiệu lực')
  })

  test('approving a contract that overlaps one in force fails at the approver', async ({ browser }) => {
    const h = await created(api.post('/api/hrm/employees', { data: employee(`CTH${run}`, ids.q, { manager_id: ids.m }) }))
    await contractViaApi(h, ids.open, '2024-01-01', null)
    const second = await contractViaApi(h, ids.fixed, '2026-03-01', '2026-08-31', false)
    expect((await api.post(`/api/documents/hrm.contract/${second}/transitions`, { data: { to: 'posted', version: 1 } })).status()).toBe(200)
    const d = await as(browser, login.dir, `/hrm/contracts/${second}`)
    await d.getByRole('button', { name: 'Duyệt', exact: true }).click()
    await expect(d.getByText(/Nhân viên đã có hợp đồng có hiệu lực trùng thời hạn này/)).toBeVisible()
    await d.reload()
    await status(d, 'Chờ duyệt')
  })

  test('the company list filters contracts ending soon and by kind, without amounts', async ({ browser }) => {
    const ending = await contractViaApi(ids.e3, ids.fixed, inDays(-900), inDays(10))
    await contractViaApi(ids.e3, ids.open, inDays(11), null)
    const number = ((await (await api.get(`/api/hrm/contracts/${ending}`)).json()) as { number: string }).number
    const page = await as(browser, login.pay, '/hrm/contracts')
    const table = page.getByRole('table', { name: 'Hợp đồng' })
    await expect(table.getByRole('row', { name: new RegExp(number) })).toBeVisible()
    await expect(table).not.toContainText('9.000.000')

    await page.getByRole('button', { name: 'Sắp hết hạn (30 ngày)' }).click()
    await expect(page).toHaveURL(/expiring=1/)
    await expect(table.getByRole('row', { name: new RegExp(`${number}.*${vn(inDays(10))}`) })).toBeVisible()
    await expect(table.getByRole('row').filter({ hasText: kinds.open })).toHaveCount(0)
    // The filter is on the URL, so Back returns to the full list.
    await page.goBack()
    await expect(table.getByRole('row').filter({ hasText: kinds.open }).first()).toBeVisible()

    await page.getByRole('combobox', { name: 'Loại hợp đồng' }).click()
    await page.getByRole('option', { name: kinds.open, exact: true }).click()
    await expect(page.getByRole('combobox', { name: `Loại hợp đồng: ${kinds.open}` })).toBeVisible()
    await expect(table.getByRole('row').filter({ hasText: kinds.fixed })).toHaveCount(0)
    await expect(table.getByRole('row').filter({ hasText: kinds.open }).first()).toBeVisible()
  })

  test('HR without the payroll role sees contracts but no amounts, and cannot create one', async ({ browser }) => {
    const id = await contractViaApi(ids.e, ids.fixed, '2023-01-01', '2023-12-31')
    const hr = await as(browser, login.hr, `/hrm/employees/${ids.e}?tab=contracts`)
    await expect(hr.getByRole('table', { name: 'Hợp đồng' })).toBeVisible()
    await expect(hr.getByRole('link', { name: 'Hợp đồng mới' })).toHaveCount(0)
    await hr.goto(`/hrm/contracts/${id}`)
    await expect(hr.getByText('Bạn không có quyền xem lương.')).toBeVisible()
    await expect(hr.getByRole('textbox', { name: /^Lương/ })).toHaveCount(0)
    // The employee themself has no HRM role, so the HRM area sends them home.
    const e = await as(browser, login.e, `/hrm/contracts/${id}`)
    await expect(e).toHaveURL(/\/$/)
  })
})
