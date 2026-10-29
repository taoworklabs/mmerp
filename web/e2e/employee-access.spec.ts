import { expect, test } from '@playwright/test'
import { created, createUser, employee, password, signedInApi, runId } from './api'
import { signInAs } from './helpers'

// One run: a company with departments A and B, employees in both, and an HR user scoped to A.
const run = runId()
const hrA = `hr_a_${run}`
const nationalId = '079123456789'
let ids: { a: number; b: number; ea: number; eb: number; hrA: number }

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty ${run}`, tax_code: '0101234567' } }))
  const a = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng A ${run}` } }))
  const b = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng B ${run}` } }))
  const ea = await created(api.post('/api/hrm/employees', { data: employee(`A${run}-00`, a, { sensitive: { national_id: nationalId } }) }))
  const eb = await created(api.post('/api/hrm/employees', { data: employee(`B${run}-00`, b) }))
  // Enough employees in A for a second page at 20 per page.
  for (let i = 1; i <= 24; i++) {
    await created(api.post('/api/hrm/employees', { data: employee(`A${run}-${String(i).padStart(2, '0')}`, a) }))
  }
  const hr = await createUser(api, hrA, [
    { role: 'hrm.hr', unit: a },
    { role: 'hrm.sensitive_viewer', unit: a },
  ])
  ids = { a, b, ea, eb, hrA: hr }
})

test('a user sees only employees within their org units, in the API and the UI', async ({ page, baseURL }) => {
  await signInAs(page, hrA, `/hrm/employees?q=${run}`)
  const table = page.getByRole('table', { name: 'Nhân viên' })
  await expect(table.getByRole('link', { name: `A${run}-00` })).toBeVisible()
  await expect(table.getByRole('link', { name: `B${run}-00` })).toHaveCount(0)
  await expect(page.getByText('25 dòng')).toBeVisible()

  await page.goto(`/hrm/employees/${ids.eb}`)
  await expect(page.getByRole('heading', { name: 'Không tìm thấy trang' })).toBeVisible()

  const api = await signedInApi(baseURL!, hrA, password)
  expect((await api.get(`/api/hrm/employees/${ids.eb}`)).status()).toBe(404)
  const list = await (await api.get(`/api/hrm/employees?q=${run}&page_size=100`)).json()
  expect(list.items.map((e: { code: string }) => e.code)).not.toContain(`B${run}-00`)
})

test('a sensitive field is masked, shown on request, and cleared when the permission is revoked mid-session', async ({ page, baseURL }) => {
  await signInAs(page, hrA, `/hrm/employees/${ids.ea}`)
  const field = 'Số CCCD hoặc hộ chiếu'
  await expect(page.getByText('••••••').first()).toBeVisible()
  await expect(page.getByText(nationalId)).toHaveCount(0)
  // The last four digits come masked from the API, the rest never reaches the page unasked.
  await expect(page.getByText('Đã ẩn, kết thúc bằng 6789')).toBeAttached()
  // Shown as text in the profile, not as an input: viewing is not editing.
  await page.getByRole('button', { name: `Hiện ${field}` }).click()
  await expect(page.getByText(nationalId)).toBeVisible()
  await expect(page.getByRole('textbox', { name: field })).toHaveCount(0)
  // Hide masks it again; showing it again is another audited read.
  await page.getByRole('button', { name: `Ẩn ${field}` }).click()
  await expect(page.getByText(nationalId)).toHaveCount(0)
  await page.getByRole('button', { name: `Hiện ${field}` }).click()
  await expect(page.getByText(nationalId)).toBeVisible()

  // The administrator revokes the sensitive-data role while the page is open.
  const admin = await signedInApi(baseURL!)
  const grants: { id: number; role: string }[] = await (await admin.get(`/api/users/${ids.hrA}/roles`)).json()
  const grant = grants.find((g) => g.role === 'sensitive_viewer')!
  expect((await admin.delete(`/api/users/${ids.hrA}/roles/${grant.id}`)).status()).toBe(204)

  // The next request carries a new X-Authz-Version: the page reloads with nothing stale left.
  const reloaded = page.waitForEvent('load')
  await page.getByRole('navigation', { name: 'Điều hướng chính' }).getByRole('link', { name: 'Nhân viên' }).click()
  await reloaded
  await page.goto(`/hrm/employees/${ids.ea}`)
  await expect(page.getByText('Không có quyền xem').first()).toBeVisible()
  await expect(page.getByRole('button', { name: `Hiện ${field}` })).toHaveCount(0)
  await expect(page.locator(`input[value="${nationalId}"]`)).toHaveCount(0)

  // Restore for the other tests.
  await created(admin.post(`/api/users/${ids.hrA}/roles`, { data: { product: 'hrm', role: 'sensitive_viewer', org_unit_id: ids.a } }))
})

test('show all reveals every sensitive value, each an audited read, and hide all masks them again', async ({ page }) => {
  await signInAs(page, hrA, `/hrm/employees/${ids.ea}`)
  const card = page.getByRole('region', { name: 'Thông tin nhạy cảm' })
  const reads: string[] = []
  page.on('request', (r) => r.url().includes('/sensitive/') && reads.push(r.url()))
  await card.getByRole('button', { name: 'Hiện tất cả' }).click()
  await expect(card.getByText(nationalId)).toBeVisible()
  // One audited call per field holding a value (only the ID here), none for the empty ones.
  expect(reads).toEqual([expect.stringContaining(`/api/hrm/employees/${ids.ea}/sensitive/national_id`)])
  await card.getByRole('button', { name: 'Ẩn tất cả' }).click()
  await expect(card.getByText(nationalId)).toHaveCount(0)
  await expect(card.getByRole('button', { name: 'Hiện tất cả' })).toBeVisible()
})

test('recent activity on the overview leaves reads out; the activity tab keeps them', async ({ page }) => {
  await signInAs(page, hrA, `/hrm/employees/${ids.ea}`)
  await page.getByRole('button', { name: 'Hiện Số CCCD hoặc hộ chiếu' }).click()
  await expect(page.getByText(nationalId)).toBeVisible()
  await page.reload()
  const recent = page.getByRole('region', { name: 'Hoạt động gần đây' })
  await expect(recent.getByLabel('Bình luận', { exact: true })).toBeVisible()
  await expect(recent.getByRole('group', { name: 'Lọc hoạt động' })).toHaveCount(0)
  await expect(recent.getByText('xem thông tin nhạy cảm')).toHaveCount(0)
  await recent.getByRole('link', { name: 'Xem tất cả' }).click()
  await expect(page).toHaveURL(/tab=activity/)
  await expect(page.getByText('xem thông tin nhạy cảm').first()).toBeVisible()
})

test('status comes from the API, and the manager picker keeps one stable label', async ({ page, baseURL }) => {
  const api = await signedInApi(baseURL!)
  const boss = await created(api.post('/api/hrm/employees', { data: employee(`M${run}`, ids.a) }))
  const left = await created(
    api.post('/api/hrm/employees', { data: employee(`L${run}`, ids.a, { manager_id: boss, termination_date: '2026-02-01' }) }),
  )

  await signInAs(page, hrA, `/hrm/employees?q=L${run}`)
  await expect(page.getByRole('table', { name: 'Nhân viên' }).getByText('Đã nghỉ việc')).toBeVisible()

  let searches = 0
  page.on('request', (r) => r.url().includes('/api/hrm/employees?') && searches++)
  await page.goto(`/hrm/employees/${left}`)
  await expect(page.getByText('Đã nghỉ việc')).toBeVisible()
  await page.getByRole('button', { name: 'Sửa Thông tin chính' }).click()
  const manager = page.getByRole('combobox', { name: 'Quản lý trực tiếp' })
  await expect(manager).toHaveValue(`M${run} · Nhân viên M${run}`)
  searches = 0
  await page.waitForTimeout(1500)
  await expect(manager).toHaveValue(`M${run} · Nhân viên M${run}`)
  expect(searches).toBe(0)
})
