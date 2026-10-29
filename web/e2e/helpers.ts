import { expect, type Browser, type Page } from '@playwright/test'

// Tests run in Node; the web tsconfig has no Node types.
declare const process: { env: Record<string, string | undefined> }

// The root account created at startup; E2E_ADMIN_* point at another one, e.g. on the dev database.
export const admin = {
  login: process.env.E2E_ADMIN_LOGIN ?? 'root',
  password: process.env.E2E_ADMIN_PASSWORD ?? '1234@Abcd',
  name: process.env.E2E_ADMIN_NAME ?? 'root',
}

// The password of every account the tests create.
export const password = 'e2e password'

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

// signInAs opens url and signs in as login; the admin keeps its own password.
export async function signInAs(page: Page, login: string, url: string) {
  await page.goto(url)
  await page.getByLabel('Tên đăng nhập').fill(login)
  await page.getByLabel(/^Mật khẩu/).fill(login === admin.login ? admin.password : password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: /^Tài khoản:/ })).toBeVisible()
}

// as signs login in on a page of its own browser context, so several users act side by side.
export async function as(browser: Browser, login: string, url: string, viewport?: { width: number; height: number }): Promise<Page> {
  const page = await (await browser.newContext(viewport ? { viewport } : {})).newPage()
  await signInAs(page, login, url)
  return page
}

// status reads the badge in the page header: the activity below still lists earlier statuses.
export const status = (page: Page, text: string) => expect(page.getByRole('main').locator('[data-page-header]').getByText(text, { exact: true })).toBeVisible()

export const fixture = (name: string) => new URL(`fixtures/${name}`, import.meta.url).pathname
