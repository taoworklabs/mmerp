import { expect, test } from '@playwright/test'
import { signedIn } from './helpers'

test('an administrator sets the mail server; its password never shows again', async ({ browser }) => {
  const page = await (await browser.newContext()).newPage()
  await signedIn(page, '/admin/mail-server')
  // The settings are a form on the page itself.
  await page.getByRole('button', { name: 'Cấu hình máy chủ mail' }).click()
  await page.getByRole('textbox', { name: /^Máy chủ/ }).fill('smtp.example.com')
  await page.getByLabel(/^Mật khẩu/).fill('s3cret pass')
  await page.getByRole('textbox', { name: /^Địa chỉ gửi/ }).fill('mmerp@example.com')
  await page.getByRole('button', { name: 'Lưu' }).click()
  await expect(page.getByText('Đã lưu cấu hình máy chủ mail.')).toBeVisible()
  await expect(page.getByRole('textbox', { name: /^Máy chủ/ })).toHaveValue('smtp.example.com')
  await expect(page.getByRole('textbox', { name: /^Cổng/ })).toHaveValue('587')
  // The password is never read back: the field is empty and only says one is kept.
  await expect(page.getByLabel(/^Mật khẩu/)).toHaveValue('')
  await expect(page.getByText('s3cret pass')).toHaveCount(0)

  await page.getByRole('button', { name: 'Xóa cấu hình' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Xóa cấu hình' }).click()
  await expect(page.getByText('Chưa cấu hình máy chủ mail.')).toBeVisible()
})
