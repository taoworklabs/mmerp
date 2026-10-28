import { expect, test, type Page } from '@playwright/test'
import { created, createUser, employee, password, runId, signedInApi } from './api'
import { signedIn } from './helpers'

// Typeable selects are used from the keyboard alone: type part of a name, arrow down, Enter.
const run = runId()
const unit = `Xưởng ${run}u`
const code = `KB${run}`

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty KB ${run}`, tax_code: '0101234567' } }))
  const dept = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: unit } }))
  await created(api.post('/api/hrm/employees', { data: employee(code, dept) }))
})

test('remote select is typeable and keyboard-selectable', async ({ page }) => {
  await signedIn(page, '/hrm/employees/new')
  const field = page.getByRole('combobox', { name: 'Phòng ban' })
  await field.focus()
  // Only this unit's name holds the tag with its trailing u (the company's has the bare tag).
  await field.pressSequentially(`${run}u`)
  await expect(page.getByRole('option', { name: unit })).toBeVisible()
  await field.press('ArrowDown')
  await field.press('Enter')
  await expect(field).toHaveValue(unit)
})

test('searchable select accepts typed text', async ({ page }) => {
  await signedIn(page, '/hrm/employees/new')
  const field = page.getByRole('combobox', { name: 'Quản lý trực tiếp' })
  await field.focus()
  await field.pressSequentially(code)
  const label = `${code} · Nhân viên ${code}`
  await expect(page.getByRole('option', { name: label })).toBeVisible()
  await field.press('ArrowDown')
  await field.press('Enter')
  await expect(field).toHaveValue(label)
})

// Dates: a leave request of an employee with a login, a leave in March and one in July to filter by.
const dates = { user: `kbd_${run}`, march: '', july: '' }

test.describe('dates', () => {
  test.beforeAll(async ({ baseURL }) => {
    const api = await signedInApi(baseURL!)
    const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty ngày ${run}` } }))
    const dept = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng ngày ${run}` } }))
    await createUser(api, dates.user, [])
    const manager = await created(api.post('/api/hrm/employees', { data: employee(`KM${run}`, dept) }))
    await created(api.post('/api/hrm/employees', { data: employee(`KE${run}`, dept, { user_login: dates.user, manager_id: manager }) }))
    const type = await created(api.post('/api/hrm/leave-types', { data: { name: `Phép ngày ${run}`, deducts_balance: false, paid: true, active: true } }))
    const as = await signedInApi(baseURL!, dates.user, password)
    for (const [key, day] of [['march', '2026-03-05'], ['july', '2026-07-08']] as const) {
      await created(as.post('/api/hrm/leaves', { data: { leave_type_id: type, start_date: day, end_date: day, days: '1', reason: null } }))
      dates[key] = day
    }
  })

  async function newLeave(page: Page) {
    await page.goto('/me/requests/leave/new')
    await page.getByLabel('Tên đăng nhập').fill(dates.user)
    await page.getByLabel(/^Mật khẩu/).fill(password)
    await page.getByRole('button', { name: 'Đăng nhập' }).click()
    await page.getByRole('combobox', { name: 'Loại nghỉ' }).click()
    await page.getByRole('option', { name: `Phép ngày ${run}` }).click()
  }

  test('date field accepts dd/mm/yyyy typed by hand', async ({ page }) => {
    await newLeave(page)
    await page.getByRole('textbox', { name: /^Từ ngày/ }).fill('05/03/2026')
    await page.getByRole('textbox', { name: /^Đến ngày/ }).fill('05/03/2026')
    await page.getByRole('textbox', { name: /^Lý do/ }).click()
    await page.getByRole('button', { name: 'Tạo bản nháp' }).click()
    await expect(page.getByRole('heading', { name: /^NP-2026-/ })).toBeVisible()
    await expect(page.getByRole('main').getByText('05/03/2026').first()).toBeVisible()
  })

  test('date picker moves by keyboard', async ({ page }) => {
    await newLeave(page)
    const field = page.getByRole('textbox', { name: /^Từ ngày/ })
    await field.fill('05/03/2026')
    await field.press('Enter')
    await field.press('ArrowDown')
    await page.keyboard.press('ArrowRight')
    await page.keyboard.press('Enter')
    await expect(field).toHaveValue('06/03/2026')
  })

  test('month filter changes the list month', async ({ page }) => {
    await signedIn(page, '/hrm/leaves')
    const rows = page.getByRole('main')
    const pick = async (name: string, month: RegExp) => {
      await page.getByRole('button', { name }).click()
      await page.getByRole('gridcell', { name: month }).click()
    }
    await pick('Tháng', /^(thg|Tháng) 7$/i)
    await expect(page.getByRole('button', { name: 'Tháng: 07/2026' })).toBeVisible()
    await expect(rows.getByText('08/07/2026 – 08/07/2026').first()).toBeVisible()
    await expect(rows.getByText('05/03/2026 – 05/03/2026')).toHaveCount(0)
    await pick('Tháng: 07/2026', /^(thg|Tháng) 3$/i)
    await expect(rows.getByText('05/03/2026 – 05/03/2026').first()).toBeVisible()
    await expect(rows.getByText('08/07/2026 – 08/07/2026')).toHaveCount(0)
  })
})
