import { expect, type Page } from '@playwright/test'

// Tests run in Node; the web tsconfig has no Node types.
declare const process: { env: Record<string, string | undefined> }

// The root account created at startup; E2E_ADMIN_* point at another one, e.g. on the dev database.
export const admin = {
  login: process.env.E2E_ADMIN_LOGIN ?? 'root',
  password: process.env.E2E_ADMIN_PASSWORD ?? '1234@Abcd',
  name: process.env.E2E_ADMIN_NAME ?? 'root',
}

export async function signIn(page: Page, password = admin.password) {
  await page.getByLabel('Tên đăng nhập').fill(admin.login)
  await page.getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
}

export async function signedIn(page: Page, url = '/') {
  await page.goto(url)
  await signIn(page)
  await expect(page.getByRole('button', { name: /^Tài khoản:/ })).toBeVisible()
}
