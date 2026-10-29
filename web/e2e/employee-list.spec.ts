import { expect, test } from '@playwright/test'
import { created, createUser, employee, signedInApi, runId } from './api'
import { signInAs } from './helpers'

// One run: a department with 25 employees, enough for a second page at 20 per page, and an
// HR user scoped to it.
const run = runId()
const hrA = `hr_a_${run}`

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty ${run}`, tax_code: '0101234567' } }))
  const a = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng A ${run}` } }))
  for (let i = 0; i <= 24; i++) {
    await created(api.post('/api/hrm/employees', { data: employee(`A${run}-${String(i).padStart(2, '0')}`, a) }))
  }
  await createUser(api, hrA, [{ role: 'hrm.hr', unit: a }])
})

test('filters, sort and page live on the URL and Back restores each step', async ({ page }) => {
  await signInAs(page, hrA, '/hrm/employees')
  const table = page.getByRole('table', { name: 'Nhân viên' })

  await page.getByLabel('Tìm kiếm').fill(run)
  await page.getByLabel('Tìm kiếm').press('Enter')
  await expect(page).toHaveURL(new RegExp(`\\?q=${run}$`))
  await expect(table.getByRole('row')).toHaveCount(26) // header + 25

  await page.getByRole('button', { name: 'Họ tên' }).click()
  await expect(page).toHaveURL(/sort=full_name/)
  // No wait for the header: a click before it re-renders still toggles.
  await page.getByRole('button', { name: 'Họ tên' }).click()
  await expect(page).toHaveURL(/sort=-full_name/)
  await expect(table.getByRole('row').nth(1)).toContainText(`A${run}-24`)

  await page.getByRole('combobox', { name: 'Số dòng mỗi trang' }).click()
  await page.getByRole('option', { name: '20 / trang' }).click()
  await expect(page).toHaveURL(/page_size=20/)
  await page.getByRole('button', { name: 'Trang 2' }).click()
  await expect(page).toHaveURL(/page=2/)
  await expect(table.getByRole('row')).toHaveCount(6)

  await page.goBack()
  await expect(page).not.toHaveURL(/page=2/)
  await expect(table.getByRole('row')).toHaveCount(21)
  await page.goBack()
  await expect(page).not.toHaveURL(/page_size/)
  await page.goBack()
  await expect(page).toHaveURL(/sort=full_name/)
  await expect(table.getByRole('row').nth(1)).toContainText(`A${run}-00`)
  await page.goBack()
  await expect(page).toHaveURL(new RegExp(`\\?q=${run}$`))
  await page.goBack()
  await expect(page).toHaveURL(/\/hrm\/employees$/)
  await expect(page.getByLabel('Tìm kiếm')).toHaveValue('')
})

test('below 1024px the employee list becomes cards and the filters wrap above them', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 800 })
  await signInAs(page, hrA, `/hrm/employees?q=${run}-0`)
  const cards = page.getByRole('list', { name: 'Nhân viên' })
  await expect(cards.getByRole('listitem').first()).toBeVisible()
  await expect(page.getByRole('table')).toHaveCount(0)
  const filters = page.getByRole('search')
  await expect(filters.getByLabel('Tìm kiếm')).toBeVisible()
  await expect(filters.getByRole('combobox', { name: 'Trạng thái' })).toBeVisible()
})
