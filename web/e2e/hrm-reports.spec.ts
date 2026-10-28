import { expect, test, type Page } from '@playwright/test'
import { created, createUser, password, runId, signedInApi } from './api'

const id = runId()
const reporter = `rep_${id}`
const salaryOnly = `repsal_${id}`
let company: number

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty BC ${id}` } }))
  await createUser(api, reporter, [
    { role: 'hrm.payroll', unit: null },
    { role: 'hrm.sensitive_viewer', unit: null },
  ])
  await createUser(api, salaryOnly, [{ role: 'hrm.payroll', unit: null }])
})

async function signIn(page: Page, login: string) {
  await page.goto('/')
  await page.getByLabel('Tên đăng nhập').fill(login)
  await page.getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: /^Tài khoản:/ })).toBeVisible()
}

test('a report over months without a posted payroll is refused, naming the months', async ({ page }) => {
  await signIn(page, reporter)
  await page.goto(`/hrm/reports?legal_entity_id=${company}&month=2026-06`)
  await expect(page.getByRole('heading', { name: 'Báo cáo BHXH & thuế TNCN' })).toBeVisible()
  const d02 = page.getByRole('row').filter({ hasText: 'D02-LT' })
  await d02.getByRole('button', { name: 'Xuất Excel' }).click()
  await expect(page.getByText(/05\/2026, 06\/2026/)).toBeVisible()
})

test('the reports need the sensitive permission too', async ({ page }) => {
  await signIn(page, salaryOnly)
  await page.goto('/hrm/reports')
  await expect(page.getByText('Không có quyền truy cập')).toBeVisible()
})
