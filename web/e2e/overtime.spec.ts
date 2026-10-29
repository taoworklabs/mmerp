import { expect, test, type Page } from '@playwright/test'
import { created, createUser, employee, signedInApi, runId, toFlow } from './api'
import { admin, as, status } from './helpers'

// One run: department Q with manager M; employees E and E2 report to M, only E has an account.
// hr is HR of Q; dir approves holiday overtime after the manager.
const run = runId()
const login = { m: `otm_${run}`, e: `ote_${run}`, hr: `othr_${run}`, dir: `otdir_${run}` }

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty OT ${run}` } }))
  const q = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng Q ${run}` } }))
  // The holiday overtime below must fall on a holiday of the company's calendar.
  const holiday = await api.put(`/api/hrm/calendars/${company}/holidays/2026-09-02`, { data: { name: 'Quốc khánh' } })
  expect(holiday.ok(), await holiday.text()).toBe(true)
  for (const l of [login.m, login.e]) await createUser(api, l, [])
  await createUser(api, login.hr, [{ role: 'hrm.hr', unit: q }])
  const dirId = await createUser(api, login.dir, [
    { role: 'hrm.hr', unit: null },
    { role: 'hrm.payroll', unit: null },
  ])
  const m = await created(api.post('/api/hrm/employees', { data: employee(`OTM${run}`, q, { user_login: login.m }) }))
  await created(api.post('/api/hrm/employees', { data: employee(`OTE${run}`, q, { user_login: login.e, manager_id: m }) }))
  await created(api.post('/api/hrm/employees', { data: employee(`OTF${run}`, q, { manager_id: m }) }))
  const steps = [{ approver: { kind: 'module' } }, { condition: { field: 'day_kind', op: 'eq', value: 'holiday' }, approver: { kind: 'user', user_ids: [dirId] } }]
  const rule = await api.put('/api/approval-rules/hrm.overtime_request', { data: { flow: toFlow(steps), max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' } })
  expect(rule.status(), await rule.text()).toBe(204)
})

const heading = (page: Page, prefix: string) => page.getByRole('heading', { name: new RegExp(`^${prefix}-\\d{4}-`) })

test.describe('overtime', () => {
  test('an employee files, withdraws and resends; the manager rejects, then approves from the inbox', async ({ browser }) => {
    const e = await as(browser, login.e, '/me/requests/overtime/new')
    await e.getByRole('textbox', { name: /^Ngày tăng ca/ }).fill('15/09/2026')
    await e.getByRole('textbox', { name: /^Giờ ca ngày/ }).fill('0')
    await e.getByRole('button', { name: 'Tạo bản nháp' }).click()
    await expect(e.getByText('Số giờ theo nửa giờ, tổng giờ ca ngày và ca đêm lớn hơn 0 và không quá 24.')).toBeVisible()
    await e.getByRole('textbox', { name: /^Giờ ca ngày/ }).fill('2,5')
    await e.getByRole('textbox', { name: /^Giờ ca đêm/ }).fill('1')
    await e.getByRole('button', { name: 'Tạo bản nháp' }).click()
    await expect(heading(e, 'TC')).toBeVisible()
    const number = (await heading(e, 'TC').textContent())!
    await e.getByRole('button', { name: 'Gửi duyệt' }).click()
    await status(e, 'Chờ duyệt')
    await e.getByRole('button', { name: 'Rút lại' }).click()
    // Withdrawing undoes the approvals so far, so it asks first.
    await e.getByRole('dialog').getByRole('button', { name: 'Rút lại' }).click()
    await status(e, 'Nháp')
    await e.getByRole('button', { name: 'Gửi duyệt' }).click()
    await status(e, 'Chờ duyệt')

    const m = await as(browser, login.m, '/me/team/overtimes')
    await m.getByRole('link', { name: number }).click()
    await m.getByRole('button', { name: 'Từ chối' }).click()
    await m.getByRole('dialog').getByRole('textbox', { name: /^Lý do/ }).fill('Chưa có kế hoạch tăng ca')
    await m.getByRole('dialog').getByRole('button', { name: 'Từ chối' }).click()
    await status(m, 'Nháp')

    await e.reload()
    await expect(e.getByText('Lý do: Chưa có kế hoạch tăng ca')).toBeVisible()
    await e.getByRole('button', { name: 'Gửi duyệt' }).click()
    await status(e, 'Chờ duyệt')

    await m.goto('/inbox')
    await m.getByRole('button', { name: new RegExp(number) }).click()
    await expect(m.getByText('2,5')).toBeVisible()
    await m.getByRole('button', { name: 'Duyệt', exact: true }).click()
    await expect(m.getByText('Đã duyệt.')).toBeVisible()
    await m.goto('/me/team/overtimes')
    await expect(m.getByRole('row', { name: new RegExp(`${number}.*Đã duyệt`) })).toBeVisible()
  })

  test('holiday overtime needs the director after the manager', async ({ browser }) => {
    const e = await as(browser, login.e, '/me/requests/overtime/new')
    await e.getByRole('textbox', { name: /^Ngày tăng ca/ }).fill('02/09/2026')
    await e.getByRole('combobox', { name: 'Loại ngày' }).click()
    await e.getByRole('option', { name: 'Ngày lễ' }).click()
    await e.getByRole('textbox', { name: /^Giờ ca ngày/ }).fill('8')
    await e.getByRole('button', { name: 'Tạo bản nháp' }).click()
    await expect(heading(e, 'TC')).toBeVisible()
    const number = (await heading(e, 'TC').textContent())!
    await e.getByRole('button', { name: 'Gửi duyệt' }).click()
    await status(e, 'Chờ duyệt')

    const m = await as(browser, login.m, '/inbox')
    await m.getByRole('button', { name: new RegExp(number) }).click()
    await m.getByRole('button', { name: 'Duyệt', exact: true }).click()
    await expect(m.getByText('Đã duyệt.')).toBeVisible()
    await e.reload()
    await status(e, 'Chờ duyệt')

    const d = await as(browser, login.dir, '/inbox')
    await d.getByRole('button', { name: new RegExp(number) }).click()
    await d.getByRole('button', { name: 'Duyệt', exact: true }).click()
    await expect(d.getByText('Đã duyệt.')).toBeVisible()
    await e.reload()
    await status(e, 'Đã duyệt')
  })

  test('HR files overtime for an employee without an account', async ({ browser }) => {
    const hr = await as(browser, login.hr, '/hrm/overtimes/new')
    await hr.getByRole('combobox', { name: 'Nhân viên' }).fill(`OTF${run}`)
    await hr.getByRole('option', { name: new RegExp(`OTF${run}`) }).click()
    await hr.getByRole('textbox', { name: /^Ngày tăng ca/ }).fill('20/09/2026')
    await hr.getByRole('combobox', { name: 'Loại ngày' }).click()
    await hr.getByRole('option', { name: 'Ngày nghỉ hằng tuần' }).click()
    await hr.getByRole('textbox', { name: /^Giờ ca ngày/ }).fill('6')
    await hr.getByRole('button', { name: 'Tạo bản nháp' }).click()
    await expect(heading(hr, 'TC')).toBeVisible()
    await expect(hr.getByText(`Nhân viên OTF${run}`).first()).toBeVisible()
  })

  test('the rule editor names day kinds in the user language', async ({ browser }) => {
    const page = await as(browser, admin.login, '/hrm/approval-rules/hrm.overtime_request')
    await expect(page.getByRole('main').getByRole('button', { name: /Ngày lễ/ }).first()).toBeVisible()
  })

  test('at 375px an employee files overtime and the manager approves it', async ({ browser }) => {
    const small = { width: 375, height: 800 }
    const e = await as(browser, login.e, '/me/requests/overtime/new', small)
    await e.getByRole('textbox', { name: /^Ngày tăng ca/ }).fill('22/09/2026')
    await e.getByRole('textbox', { name: /^Giờ ca đêm/ }).fill('2')
    await e.getByRole('button', { name: 'Tạo bản nháp' }).click()
    await expect(heading(e, 'TC')).toBeVisible()
    const number = (await heading(e, 'TC').textContent())!
    await e.getByRole('button', { name: 'Gửi duyệt' }).click()
    await status(e, 'Chờ duyệt')

    const m = await as(browser, login.m, '/inbox', small)
    await m.getByRole('button', { name: new RegExp(number) }).click()
    await m.getByRole('button', { name: 'Duyệt', exact: true }).click()
    await expect(m.getByText('Đã duyệt.')).toBeVisible()
    await m.goto('/me/team/overtimes')
    await expect(m.getByRole('list', { name: 'Đơn tăng ca của nhóm' }).getByText(number)).toBeVisible()
  })
})
