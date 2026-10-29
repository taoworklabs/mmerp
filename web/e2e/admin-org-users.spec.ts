import { expect, test } from '@playwright/test'
import { password, runId } from './api'
import { admin, signInAs } from './helpers'

const run = runId()

test('an administrator builds the tree and grants a role from the admin screens', async ({ page }) => {
  await signInAs(page, admin.login, '/admin/org-units')
  await page.getByRole('button', { name: 'Tạo đơn vị' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('combobox', { name: 'Loại' }).click()
  await page.getByRole('option', { name: 'Pháp nhân' }).click()
  await dialog.getByRole('textbox', { name: 'Tên', exact: true }).fill(`UI ${run}`)
  await dialog.getByLabel('Mã số thuế').fill('0309999999')
  await dialog.getByRole('button', { name: 'Lưu' }).click()
  await expect(page.getByRole('list', { name: 'Cây tổ chức' }).getByText(`UI ${run}`)).toBeVisible()

  await page.getByRole('navigation').getByRole('link', { name: 'Người dùng' }).click()
  await page.getByRole('button', { name: 'Tạo người dùng' }).click()
  await page.getByRole('dialog').getByLabel('Tên đăng nhập').fill(`ui_${run}`)
  await page.getByRole('dialog').getByLabel('Họ tên').fill(`UI ${run}`)
  await page.getByRole('dialog').getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('dialog').getByRole('button', { name: 'Tạo người dùng' }).click()
  await expect(page.getByRole('heading', { name: `UI ${run}` })).toBeVisible()

  await page.getByRole('button', { name: 'Cấp vai trò' }).click()
  await page.getByRole('dialog').getByRole('combobox', { name: 'Vai trò' }).click()
  await page.getByRole('option', { name: 'Xem hồ sơ nhân viên' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Cấp vai trò' }).click()
  // Granted from the header, the role shows in the overview's digest and in its tab.
  await expect(page.getByRole('region', { name: 'Vai trò' }).getByText('Xem hồ sơ nhân viên')).toBeVisible()
  await expect(page.getByRole('region', { name: 'Vai trò' }).getByText('Toàn công ty')).toBeVisible()
  await page.getByRole('tab', { name: 'Vai trò (1)' }).click()
  await expect(page.getByRole('table', { name: 'Vai trò' }).getByText('Xem hồ sơ nhân viên')).toBeVisible()
})
