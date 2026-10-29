import { expect, test } from '@playwright/test'
import { created, createUser, password, signedInApi, runId } from './api'
import { signedIn, signInAs } from './helpers'

// One run: a company with a department and an HR user scoped to that department.
const run = runId()
const hr = `oc_hr_${run}`
const dept = `Phòng OC ${run}`

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty OC ${run}`, tax_code: '0101234567' } }))
  const unit = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: dept } }))
  await createUser(api, hr, [{ role: 'hrm.hr', unit }])
})

test('an HRM user reads the whole org tree inside HRM, without edit actions', async ({ page, baseURL }) => {
  await signInAs(page, hr, '/hrm/overview')
  await page.getByRole('navigation').getByRole('link', { name: 'Sơ đồ tổ chức' }).click()
  await expect(page).toHaveURL(/\/hrm\/org$/)
  await expect(page.getByRole('list', { name: 'Sơ đồ tổ chức' }).getByText(dept)).toBeVisible()
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
