import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { createUser, password, runId, signedInApi } from './api'
import { admin, signedIn } from './helpers'

// The account lifecycle: an administrator resets and locks a user; the user changes a
// temporary password and ends sessions. Each test has its own user, never root.
const run = runId()
let api: APIRequestContext

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
})

async function signInAs(page: Page, login: string, pw: string) {
  await page.getByLabel('Tên đăng nhập').fill(login)
  await page.getByLabel(/^Mật khẩu/).fill(pw)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
}

const account = (page: Page) => page.getByRole('button', { name: /^Tài khoản:/ })

test('a reset password must be changed at the next sign-in, before anything else', async ({ page, browser }) => {
  const login = `reset_${run}`
  const id = await createUser(api, login, [{ role: 'hrm.hr', unit: null }])
  const temporary = 'Tạm thời 2026'
  const chosen = 'Mật khẩu mới 2026'

  await signedIn(page, `/admin/users/${id}`)
  await page.getByRole('button', { name: 'Thao tác khác' }).click()
  await page.getByRole('menuitem', { name: 'Đặt lại mật khẩu' }).click()
  const modal = page.getByRole('dialog')
  await modal.getByLabel(/^Mật khẩu tạm/).fill(temporary)
  await modal.getByRole('button', { name: 'Đặt lại mật khẩu' }).click()
  await expect(page.getByText('Đã đặt lại mật khẩu.')).toBeVisible()

  const user = await (await browser.newContext()).newPage()
  await user.goto('/hrm/employees')
  await signInAs(user, login, temporary)
  await expect(user.getByText('Quản trị viên đã đặt mật khẩu tạm thời cho bạn.', { exact: false })).toBeVisible()
  // Only the change and logout: no menu, no area.
  await expect(user.getByRole('navigation')).toHaveCount(0)
  await expect(account(user)).toHaveCount(0)
  await expect(user.getByRole('button', { name: 'Đăng xuất' })).toBeVisible()
  await user.goto('/admin/users')
  await expect(user.getByLabel(/^Mật khẩu hiện tại/)).toBeVisible()
  await expect(user.getByRole('heading', { name: 'Người dùng' })).toHaveCount(0)

  await user.getByLabel(/^Mật khẩu hiện tại/).fill(temporary)
  await user.getByLabel(/^Mật khẩu mới/).fill(chosen)
  await user.getByLabel(/^Nhập lại mật khẩu mới/).fill(chosen)
  await user.getByRole('button', { name: 'Đổi mật khẩu' }).click()
  await expect(user.getByRole('heading', { name: `Xin chào, ${login}` })).toBeVisible()
  await expect(account(user)).toBeVisible()
  await user.context().close()
})

test('a locked user is signed out and cannot sign in until unlocked', async ({ page, browser }) => {
  const login = `lock_${run}`
  const id = await createUser(api, login, [{ role: 'hrm.hr', unit: null }])

  const user = await (await browser.newContext()).newPage()
  await user.goto('/hrm/employees')
  await signInAs(user, login, password)
  await expect(account(user)).toBeVisible()

  await signedIn(page, `/admin/users/${id}`)
  await page.getByRole('button', { name: 'Thao tác khác' }).click()
  await page.getByRole('menuitem', { name: 'Khoá tài khoản' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Khoá tài khoản' }).click()
  await expect(page.getByText('Đã khoá tài khoản.')).toBeVisible()
  await expect(page.getByText('Đã khoá', { exact: true })).toBeVisible()

  // The open page's next request finds its session gone.
  await account(user).click()
  await user.getByRole('menuitem', { name: 'Phiên đăng nhập' }).click()
  await expect(user.getByRole('button', { name: 'Đăng nhập' })).toBeVisible()
  // A wrong password says nothing about the lock; the right one does.
  await signInAs(user, login, 'wrong password')
  await expect(user.getByRole('alert')).toContainText('Tên đăng nhập hoặc mật khẩu không đúng')
  await signInAs(user, login, password)
  await expect(user.getByRole('alert')).toContainText('Tài khoản đã bị khóa.')

  await page.getByRole('button', { name: 'Thao tác khác' }).click()
  await page.getByRole('menuitem', { name: 'Mở khoá' }).click()
  await expect(page.getByText('Đã mở khoá tài khoản.')).toBeVisible()
  await expect(page.getByText('Đã khoá', { exact: true })).toHaveCount(0)

  await signInAs(user, login, password)
  await expect(account(user)).toBeVisible()
  await user.context().close()
})

test('the root account cannot be locked, reset or lose its administrator role', async ({ page }) => {
  const { id } = (await (await api.get('/api/me')).json()) as { id: number }
  await signedIn(page, `/admin/users/${id}`)
  await expect(page.getByRole('heading', { name: admin.name })).toBeVisible()
  await expect(page.getByText('Tài khoản gốc')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Thao tác khác' })).toHaveCount(0)

  await page.getByRole('tab', { name: /^Vai trò/ }).click()
  const adminRow = page.getByRole('row', { name: /Quản trị hệ thống/ })
  await expect(adminRow).toBeVisible()
  await expect(adminRow.getByRole('button', { name: 'Thao tác khác' })).toHaveCount(0)
})

test('a user ends a session on another device, which is then signed out', async ({ browser }) => {
  const login = `sessions_${run}`
  await createUser(api, login, [{ role: 'hrm.hr', unit: null }])

  const here = await (await browser.newContext()).newPage()
  const there = await (await browser.newContext()).newPage()
  for (const p of [here, there]) {
    await p.goto('/hrm/employees')
    await signInAs(p, login, password)
    await expect(account(p)).toBeVisible()
  }

  await account(here).click()
  await here.getByRole('menuitem', { name: 'Phiên đăng nhập' }).click()
  const table = here.getByRole('table', { name: 'Phiên đăng nhập' })
  await expect(table.getByRole('row')).toHaveCount(3) // header + 2
  const other = table.getByRole('row').filter({ has: here.getByRole('button', { name: 'Kết thúc' }) })
  await other.getByRole('button', { name: 'Kết thúc' }).click()
  await expect(here.getByText('Đã kết thúc phiên.')).toBeVisible()
  await expect(table.getByRole('row')).toHaveCount(2)
  await expect(table.getByText('Thiết bị này')).toBeVisible()

  await there.reload()
  await expect(there.getByRole('button', { name: 'Đăng nhập' })).toBeVisible()
  await here.reload()
  await expect(account(here)).toBeVisible()
  await here.context().close()
  await there.context().close()
})
