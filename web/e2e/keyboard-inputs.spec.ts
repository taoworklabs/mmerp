import { expect, test } from '@playwright/test'
import { created, employee, runId, signedInApi } from './api'
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
