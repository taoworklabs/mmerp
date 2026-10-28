import { expect, test, type APIRequestContext } from '@playwright/test'
import { created, createUser, employee, password, signedInApi, runId } from './api'

const run = runId()
const hr = `tmhr_${run}`
let api: APIRequestContext
let emp: number

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty TM ${run}` } }))
  const unit = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng TM ${run}` } }))
  emp = await created(api.post('/api/hrm/employees', { data: employee(`TM${run}`, unit, { hire_date: '2020-01-01' }) }))
  await createUser(api, hr, [{ role: 'hrm.hr', unit }])
})

test('HR terminates an employee through a termination, and the profile then shows the date and links to it', async ({ page }) => {
  await page.goto('/')
  await page.getByLabel('Tên đăng nhập').fill(hr)
  await page.getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: /^Tài khoản:/ })).toBeVisible()
  await page.goto(`/hrm/employees/${emp}`)

  await page.getByRole('button', { name: 'Thao tác khác' }).click()
  await page.getByRole('menuitem', { name: 'Cho nghỉ việc' }).click()
  await expect(page).toHaveURL(new RegExp(`/hrm/terminations/new\\?employee=${emp}`))
  await page.getByRole('textbox', { name: 'Ngày nghỉ việc' }).fill('31/12/2026')
  await page.getByRole('combobox', { name: 'Lý do' }).click()
  await page.getByRole('option', { name: /Hai bên thỏa thuận/ }).click()
  await page.getByRole('textbox', { name: 'Số tháng đóng BHTN' }).fill('60')
  await page.getByRole('textbox', { name: 'Nội dung bàn giao' }).fill('Bàn giao máy tính cho trưởng phòng')
  await page.getByRole('button', { name: 'Tạo bản nháp' }).click()

  await expect(page.getByRole('heading', { name: /^NV-\d{4}-/ })).toBeVisible()
  const number = (await page.getByRole('heading', { name: /^NV-\d{4}-/ }).textContent())!
  // 84 months worked, 60 insured: 24 months, two years of severance; HR does not see the amount.
  await expect(page.getByText('Trợ cấp thôi việc', { exact: true }).first()).toBeVisible()
  await expect(page.getByText('2 năm (24 tháng)')).toBeVisible()
  await expect(page.getByText(/Số tiền chỉ hiện với người được xem lương/)).toBeVisible()
  const employeeDate = async () => (await (await api.get(`/api/hrm/employees/${emp}`)).json()).termination_date
  expect(await employeeDate()).toBeNull()

  // No approval rule: sending puts it in force at once.
  await page.getByRole('button', { name: 'Gửi duyệt' }).click()
  await expect(page.getByRole('main').getByText('Có hiệu lực', { exact: true }).first()).toBeVisible()
  expect(await employeeDate()).toBe('2026-12-31')

  await page.goto(`/hrm/employees/${emp}`)
  await expect(page.getByText('31/12/2026')).toBeVisible()
  await expect(page.getByRole('link', { name: number })).toBeVisible()
  await page.getByRole('button', { name: 'Thao tác khác' }).click()
  await expect(page.getByRole('menuitem', { name: 'Cho nghỉ việc' })).toHaveCount(0)
})
