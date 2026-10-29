import { expect, test, type Page } from '@playwright/test'
import { created, createUser, employee, password, signedInApi, runId } from './api'
import { signedIn, signInAs } from './helpers'

// One run: a company with departments A (3 employees) and B (2), an HR user and a viewer scoped to A.
// The tiles count tenant-wide data, so this file runs alone.
const run = runId()
const hr = `ov_hr_${run}`
const viewer = `ov_viewer_${run}`
const emp = `ov_emp_${run}` // no role: files leave only
let viewerId: number

const tiles = ['Nhân sự đang làm việc', 'Hợp đồng sắp hết hạn', 'Đơn nghỉ phép chờ duyệt', 'Đơn tăng ca chờ duyệt', 'Hợp đồng chờ duyệt']

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty OV ${run}`, tax_code: '0101234567' } }))
  const a = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng A OV ${run}` } }))
  const b = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng B OV ${run}` } }))
  for (let i = 0; i < 3; i++) await created(api.post('/api/hrm/employees', { data: employee(`OVA${run}-${i}`, a) }))
  for (let i = 0; i < 2; i++) await created(api.post('/api/hrm/employees', { data: employee(`OVB${run}-${i}`, b) }))
  await createUser(api, hr, [{ role: 'hrm.hr', unit: a }])
  viewerId = await createUser(api, viewer, [{ role: 'hrm.viewer', unit: a }])
  // A leave draft, so HRM has documents even when this file runs alone.
  await createUser(api, emp, [])
  await created(api.post('/api/hrm/employees', { data: employee(`OVE${run}`, b, { user_login: emp }) }))
  const type = await created(api.post('/api/hrm/leave-types', { data: { name: `Phép OV ${run}`, deducts_balance: false, paid: true, active: true } }))
  const asEmp = await signedInApi(baseURL!, emp, password)
  await created(asEmp.post('/api/hrm/leaves', { data: { leave_type_id: type, start_date: '2026-05-11', end_date: '2026-05-11', days: '1', reason: null } }))
})

const tile = (page: Page, label: string) => page.getByRole('link', { name: new RegExp(`^${label}`) })

// Every tile's number equals the row count of the list it opens.
async function tilesMatchLists(page: Page) {
  for (const label of tiles) {
    await page.goto('/hrm/overview')
    const value = tile(page, label).locator('p').last()
    await expect(value).toHaveText(/^\d+$/)
    const n = await value.textContent()
    await tile(page, label).click()
    if (n === '0') await expect(page.getByRole('table')).toHaveCount(0)
    else await expect(page.getByText(`${n} dòng`)).toBeVisible()
  }
}

test('HRM opens on the overview; each number matches its list within the user scope', async ({ page }) => {
  await signInAs(page, hr, '/hrm')
  await expect(page).toHaveURL(/\/hrm\/overview$/)
  await expect(tile(page, 'Nhân sự đang làm việc')).toContainText('3')
  await tilesMatchLists(page)
})

test('the administrator sees every number match its list', async ({ page }) => {
  await signedIn(page, '/hrm/overview')
  await tilesMatchLists(page)
})

test('a viewer sees only the employee block; revoking the role mid-session sends them home', async ({ page, baseURL }) => {
  await signInAs(page, viewer, '/hrm/overview')
  await expect(tile(page, 'Nhân sự đang làm việc')).toContainText('3')
  for (const label of tiles.slice(1)) await expect(tile(page, label)).toHaveCount(0)

  const admin = await signedInApi(baseURL!)
  const grants: { id: number }[] = await (await admin.get(`/api/users/${viewerId}/roles`)).json()
  expect((await admin.delete(`/api/users/${viewerId}/roles/${grants[0]!.id}`)).status()).toBe(204)

  // The next request carries a new X-Authz-Version and the app reloads.
  const reloaded = page.waitForEvent('load')
  await page.getByRole('navigation').getByRole('link', { name: 'Đơn nghỉ phép' }).click()
  await reloaded
  // With no HRM role left, the HRM area sends them home.
  await page.goto('/hrm/overview')
  await expect(page).toHaveURL(/\/$/)
  await expect(tile(page, 'Nhân sự đang làm việc')).toHaveCount(0)
})
