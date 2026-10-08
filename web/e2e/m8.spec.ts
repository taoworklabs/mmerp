import { expect, test, type Page } from '@playwright/test'
import { created, createUser, password, signedInApi } from './api'
import { signedIn } from './helpers'

// One run: a company with a department, an HR user scoped to that department, and an HRM
// approval rule administrator.
const run = Date.now().toString(36)
const hr = `m8_hr_${run}`
const rules = `m8_rules_${run}`
const dept = `Phòng M8 ${run}`

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty M8 ${run}`, tax_code: '0101234567' } }))
  const unit = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: dept } }))
  await createUser(api, hr, [{ role: 'hrm.hr', unit }])
  await createUser(api, rules, [{ role: 'hrm.approval_admin', unit: null }])
  const rule = await api.put('/api/approval-rules/hrm.leave_request', {
    data: { steps: [{ approver: { kind: 'module' } }], max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' },
  })
  expect(rule.status(), await rule.text()).toBe(204)
})

async function signInAs(page: Page, login: string, url: string) {
  await page.goto(url)
  await page.getByLabel('Tên đăng nhập').fill(login)
  await page.getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: 'Tài khoản' })).toBeVisible()
}

test('an HRM user reads the whole org tree inside HRM, without edit actions', async ({ page, baseURL }) => {
  await signInAs(page, hr, '/hrm/overview')
  await page.getByRole('navigation').getByRole('link', { name: 'Cơ cấu tổ chức' }).click()
  await expect(page).toHaveURL(/\/hrm\/org$/)
  await expect(page.getByRole('table', { name: 'Cơ cấu tổ chức' }).getByText(dept)).toBeVisible()
  await expect(page.getByRole('link', { name: 'Sửa cơ cấu' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Thao tác khác' })).toHaveCount(0)

  const api = await signedInApi(baseURL!, hr, password)
  expect((await api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `X ${run}` } })).status()).toBe(403)
})

test('an administrator gets a link from the HRM tree to the tree editor', async ({ page }) => {
  await signedIn(page, '/hrm/org')
  await page.getByRole('link', { name: 'Sửa cơ cấu' }).click()
  await expect(page).toHaveURL(/\/admin\/org-units$/)
  await expect(page.getByRole('button', { name: 'Tạo đơn vị' })).toBeVisible()
})

test('an HRM approval rule administrator edits HRM rules inside HRM', async ({ page }) => {
  await signInAs(page, rules, '/hrm/overview')
  const nav = page.getByRole('navigation')
  await nav.getByRole('link', { name: 'Quy tắc duyệt' }).click()
  await expect(page).toHaveURL(/\/hrm\/approval-rules$/)
  const table = page.getByRole('table', { name: 'Quy tắc duyệt' })
  await expect(table.getByRole('link', { name: 'Đơn nghỉ phép' })).toBeVisible()
  await expect(table.getByRole('link')).toHaveCount(5)

  await table.getByRole('link', { name: 'Đơn nghỉ phép' }).click()
  await expect(page).toHaveURL(/\/hrm\/approval-rules\/hrm\.leave_request$/)
  await page.getByRole('textbox', { name: /Số cấp quản lý tối đa/ }).fill('4')
  await page.getByRole('button', { name: 'Lưu' }).click()
  await expect(page.getByText('Đã lưu quy tắc duyệt.')).toBeVisible()
  await expect(page).toHaveURL(/\/hrm\//)
})

test('the administrator finds approval rules in the HRM menu, not in administration', async ({ page }) => {
  await signedIn(page, '/hrm/overview')
  const nav = page.getByRole('navigation')
  await expect(nav.getByRole('link', { name: 'Quy tắc duyệt' })).toHaveAttribute('href', '/hrm/approval-rules')
  await page.goto('/admin/users')
  await expect(nav.getByRole('link', { name: 'Người dùng' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Quy tắc duyệt' })).toHaveCount(0)
  await page.goto('/admin/approval-rules')
  await expect(page.getByRole('heading', { name: 'Không tìm thấy trang' })).toBeVisible()
})
