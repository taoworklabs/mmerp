import { expect, test } from '@playwright/test'
import { createUser, signedInApi, runId, toFlow } from './api'
import { signedIn, signInAs } from './helpers'

// One run: an HRM approval rule administrator, and a leave rule for them to edit.
const run = runId()
const rules = `arm_rules_${run}`

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  await createUser(api, rules, [{ role: 'hrm.approval_admin', unit: null }])
  const rule = await api.put('/api/approval-rules/hrm.leave_request', {
    data: { flow: toFlow([{ approver: { kind: 'module' } }]), max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' },
  })
  expect(rule.status(), await rule.text()).toBe(204)
})

test('an HRM approval rule administrator edits HRM rules inside HRM', async ({ page }) => {
  await signInAs(page, rules, '/hrm/overview')
  const nav = page.getByRole('navigation')
  await nav.getByRole('link', { name: 'Quy trình duyệt' }).click()
  await expect(page).toHaveURL(/\/hrm\/approval-rules$/)
  const table = page.getByRole('table', { name: 'Quy trình duyệt' })
  await expect(table.getByRole('link', { name: 'Đơn nghỉ phép · Gửi duyệt' })).toBeVisible()
  // Six document types, each with a post and a cancel rule.
  await expect(table.getByRole('link')).toHaveCount(12)

  await table.getByRole('link', { name: 'Đơn nghỉ phép · Gửi duyệt' }).click()
  await expect(page).toHaveURL(/\/hrm\/approval-rules\/hrm\.leave_request$/)
  await page.getByRole('button', { name: 'Cài đặt chung' }).click()
  await page.getByRole('complementary').getByRole('textbox', { name: /^Khi quản lý trực tiếp lại chính là người gửi/ }).fill('4')
  await page.getByRole('button', { name: 'Lưu' }).click()
  await expect(page.getByText('Đã lưu quy trình duyệt.')).toBeVisible()
  await expect(page).toHaveURL(/\/hrm\//)
})

test('the administrator finds approval rules in the HRM menu, not in administration', async ({ page }) => {
  await signedIn(page, '/hrm/overview')
  const nav = page.getByRole('navigation')
  await expect(nav.getByRole('link', { name: 'Quy trình duyệt' })).toHaveAttribute('href', '/hrm/approval-rules')
  await page.goto('/admin/users')
  await expect(nav.getByRole('link', { name: 'Người dùng' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Quy trình duyệt' })).toHaveCount(0)
  await page.goto('/admin/approval-rules')
  await expect(page.getByRole('heading', { name: 'Không tìm thấy trang' })).toBeVisible()
})
