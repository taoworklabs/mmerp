import { expect, request, test } from '@playwright/test'
import { admin, signIn, signedIn } from './helpers'

test('wrong password shows the error on the form; the right one opens the app', async ({ page }) => {
  await page.goto('/')
  await signIn(page, 'wrong password')
  await expect(page.getByRole('alert')).toContainText('Tên đăng nhập hoặc mật khẩu không đúng')
  await expect(page.getByLabel(/^Mật khẩu/)).toBeVisible()

  await page.getByLabel(/^Mật khẩu/).fill(admin.password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('heading', { name: `Xin chào, ${admin.name}` })).toBeVisible()
  await expect(page.getByRole('main').getByRole('link', { name: 'Quản lý nhân sự' })).toBeVisible()
})

test('a session revoked on the server sends every open tab to the login screen once', async ({ context, page, baseURL }) => {
  await signedIn(page, '/hrm/employees')
  const other = await context.newPage()
  await other.goto('/')
  await expect(other.getByRole('button', { name: 'Tài khoản' })).toBeVisible()

  // Revoke from outside the browser, as an administrator or expiry would.
  const cookie = (await context.cookies()).find((c) => c.name === 'session')!
  const outside = await request.newContext({ baseURL, extraHTTPHeaders: { cookie: `session=${cookie.value}` } })
  expect((await outside.post('/api/auth/logout')).status()).toBe(204)

  const loads = { page: 0, other: 0 }
  page.on('load', () => loads.page++)
  other.on('load', () => loads.other++)

  // Any request of the open tab now gets 401.
  await page.getByRole('button', { name: 'Tài khoản' }).click()
  await page.getByRole('menuitem', { name: 'English' }).click()

  await expect(page).toHaveURL(/\/login\?next=%2Fhrm%2Femployees$/)
  await expect(page.getByRole('button', { name: 'Đăng nhập' })).toBeVisible()
  await expect(other.getByRole('button', { name: 'Đăng nhập' })).toBeVisible()
  // No reload loop: each tab loaded exactly once more, and stays put.
  await page.waitForTimeout(1500)
  expect(loads).toEqual({ page: 1, other: 1 })
  await expect(page.getByRole('button', { name: 'Đăng nhập' })).toBeVisible()

  // Signing in returns to where the session ended.
  await signIn(page)
  await expect(page).toHaveURL(/\/hrm\/employees$/)
})

test('changing the language changes the whole interface, menu included', async ({ page }) => {
  await signedIn(page, '/hrm/overview')
  const nav = page.getByRole('navigation')
  await expect(nav.getByRole('link', { name: 'Nhân viên' })).toBeVisible()

  await page.getByRole('button', { name: 'Tài khoản' }).click()
  await page.getByRole('menuitem', { name: 'English' }).click()
  await expect(page.getByRole('heading', { name: 'Overview' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Employees' })).toBeVisible()
  await expect(page.getByRole('banner').getByText('Human resource management')).toBeVisible()
  await expect(page.locator('html')).toHaveAttribute('lang', 'en')

  // Kept on the server: a fresh load is still English.
  await page.reload()
  await expect(nav.getByRole('link', { name: 'Employees' })).toBeVisible()

  await page.getByRole('button', { name: 'Account' }).click()
  await page.getByRole('menuitem', { name: 'Tiếng Việt' }).click()
  await expect(nav.getByRole('link', { name: 'Nhân viên' })).toBeVisible()
})
