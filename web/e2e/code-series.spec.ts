import { expect, test } from '@playwright/test'
import { created, runId, signedInApi } from './api'
import { signedIn } from './helpers'

// One run: a team to own the customer, and a customer prefix no other file uses.
const run = runId()
const team = `Kinh doanh mã ${run}`
const prefix = `P${run.slice(-6)}-`

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty mã ${run}`, legal_name: `Công ty TNHH mã ${run}` } }))
  await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: team } }))
})

test('an administrator changes the customer prefix; a customer saved without a code gets it, and keeps it', async ({ page }) => {
  await signedIn(page, '/admin/code-series')
  await page.getByRole('row', { name: /Khách hàng/ }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('textbox', { name: 'Tiền tố' }).fill(prefix)
  await dialog.getByRole('button', { name: 'Lưu' }).click()
  await expect(page.getByText('Đã lưu tiền tố.')).toBeVisible()
  await expect(page.getByRole('row', { name: /Khách hàng/ })).toContainText(prefix)

  await page.goto('/sales/customers')
  await page.getByRole('link', { name: 'Thêm khách hàng' }).click()
  await page.getByRole('textbox', { name: 'Tên khách hàng' }).fill(`Công ty Bình Minh ${run}`)
  await page.getByRole('combobox', { name: 'Đơn vị phụ trách' }).click()
  await page.getByRole('option', { name: team }).click()
  await page.getByRole('button', { name: 'Thêm khách hàng' }).click()
  await expect(page.getByRole('heading', { name: `Công ty Bình Minh ${run}` })).toBeVisible()
  await expect(page.getByText(new RegExp(`^${prefix}\\d{5}$`))).toBeVisible()

  // The code reads but no longer edits.
  await page.getByRole('button', { name: 'Sửa Thông tin chung' }).click()
  await expect(page.getByRole('dialog').getByRole('textbox', { name: 'Mã khách hàng' })).toHaveAttribute('readonly')
})
