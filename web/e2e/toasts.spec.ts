import { expect, test, type Page } from '@playwright/test'
import { created, createUser, password, runId, signedInApi } from './api'

// Toasts (Undo, survival across a route change) and the record tab row, driven as one catalogue user.
const run = runId()
const user = `tst_${run}`
const unit = `Đơn vị toast ${run}`

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty toast ${run}`, legal_name: `Công ty TNHH toast ${run}` } }))
  const dept = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: unit } }))
  await createUser(api, user, [
    { role: 'sales.staff', unit: dept },
    { role: 'partner.editor', unit: dept },
    { role: 'item.admin', unit: null },
  ])
  await created(api.post('/api/items', { data: { code: `TS${run}`, name: `Bút toast ${run}`, unit: 'hộp', price: 1000, vat_rate: '10', active: true } }))
})

async function open(page: Page, url: string) {
  await page.goto(url)
  await page.getByLabel('Tên đăng nhập').fill(user)
  await page.getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: /^Tài khoản:/ })).toBeVisible()
}

test('undo restores a retired item', async ({ page }) => {
  await open(page, '/sales/items')
  const row = page.getByRole('row', { name: new RegExp(`Bút toast ${run}`) })
  await expect(row.getByText('Đang bán', { exact: true })).toBeVisible()
  await row.getByRole('button', { name: 'Thao tác khác' }).click()
  await page.getByRole('menuitem', { name: 'Ngừng bán' }).click()
  await expect(page.getByText(`Đã ngừng bán Bút toast ${run}.`)).toBeVisible()
  await expect(row.getByText('Ngừng bán', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Hoàn tác' }).click()
  await expect(row.getByText('Đang bán', { exact: true })).toBeVisible()
  await expect(page.getByText(`Đã ngừng bán Bút toast ${run}.`)).toBeHidden()
})

test('toast survives navigation, and record tabs move by arrow keys', async ({ page }) => {
  await open(page, '/sales/customers/new')
  await page.getByRole('textbox', { name: 'Mã khách hàng' }).fill(`KT${run}`)
  await page.getByRole('textbox', { name: 'Tên khách hàng' }).fill(`Khách toast ${run}`)
  await page.getByRole('combobox', { name: 'Đơn vị phụ trách' }).click()
  await page.getByRole('option', { name: unit }).click()
  await page.getByRole('button', { name: 'Thêm khách hàng' }).click()
  // Saving moves to the new record; the toast stays on screen across the route change.
  await expect(page.getByRole('heading', { name: `Khách toast ${run}` })).toBeVisible()
  await expect(page.getByText('Đã thêm khách hàng.')).toBeVisible()

  const tabs = page.getByRole('tab')
  await tabs.first().focus()
  await page.keyboard.press('ArrowRight')
  await expect(tabs.nth(1)).toBeFocused()
  await expect(tabs.nth(1)).toHaveAttribute('aria-selected', 'true')
  await expect(page).toHaveURL(/tab=/)
})
