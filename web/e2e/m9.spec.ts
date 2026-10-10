import { expect, test, type Page } from '@playwright/test'
import { created, createUser, employee, password, signedInApi } from './api'
import { signedIn } from './helpers'

// One run: a company with departments A (3 employees) and B (2), an HR user and a viewer scoped to A.
const run = Date.now().toString(36)
const hr = `m9_hr_${run}`
const viewer = `m9_viewer_${run}`
const menuViewer = `m9_menu_${run}`
const emp = `m9_emp_${run}` // no role: files leave only
let viewerId: number
let timesheet: number

const tiles = ['Nhân sự đang làm việc', 'Hợp đồng sắp hết hạn', 'Đơn nghỉ chờ duyệt', 'Đơn tăng ca chờ duyệt', 'Hợp đồng chờ duyệt']

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty M9 ${run}`, tax_code: '0101234567' } }))
  const a = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng A M9 ${run}` } }))
  const b = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng B M9 ${run}` } }))
  for (let i = 0; i < 3; i++) await created(api.post('/api/hrm/employees', { data: employee(`M9A${run}-${i}`, a) }))
  for (let i = 0; i < 2; i++) await created(api.post('/api/hrm/employees', { data: employee(`M9B${run}-${i}`, b) }))
  await createUser(api, hr, [{ role: 'hrm.hr', unit: a }])
  viewerId = await createUser(api, viewer, [{ role: 'hrm.viewer', unit: a }])
  await createUser(api, menuViewer, [{ role: 'hrm.viewer', unit: a }])
  // A leave draft, so HRM has documents even when this file runs alone.
  await createUser(api, emp, [])
  await created(api.post('/api/hrm/employees', { data: employee(`M9E${run}`, b, { user_login: emp }) }))
  const type = await created(api.post('/api/hrm/leave-types', { data: { name: `Phép M9 ${run}`, deducts_balance: false, paid: true, active: true } }))
  const asEmp = await signedInApi(baseURL!, emp, password)
  await created(asEmp.post('/api/hrm/leaves', { data: { leave_type_id: type, start_date: '2026-05-11', end_date: '2026-05-11', days: '1', reason: null } }))
  timesheet = await created(api.post('/api/hrm/timesheets', { data: { org_unit_id: a, month: '2026-09' } }))
})

async function signInAs(page: Page, login: string, url: string) {
  await page.goto(url)
  await page.getByLabel('Tên đăng nhập').fill(login)
  await page.getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: 'Tài khoản' })).toBeVisible()
}

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

test('a viewer sees only the employee block; revoking the role mid-session removes it', async ({ page, baseURL }) => {
  await signInAs(page, viewer, '/hrm/overview')
  await expect(tile(page, 'Nhân sự đang làm việc')).toContainText('3')
  for (const label of tiles.slice(1)) await expect(tile(page, label)).toHaveCount(0)

  const admin = await signedInApi(baseURL!)
  const grants: { id: number }[] = await (await admin.get(`/api/users/${viewerId}/roles`)).json()
  expect((await admin.delete(`/api/users/${viewerId}/roles/${grants[0]!.id}`)).status()).toBe(204)

  // The next request carries a new X-Authz-Version and the app reloads.
  const reloaded = page.waitForEvent('load')
  await page.getByRole('navigation').getByRole('link', { name: 'Đơn nghỉ' }).click()
  await reloaded
  await page.goto('/hrm/overview')
  await expect(page.getByText('Chưa có số liệu nào cho bạn.')).toBeVisible()
  await expect(tile(page, 'Nhân sự đang làm việc')).toHaveCount(0)
})

test('home has no sidebar; each area shows only its own menu; the header names the area', async ({ page }) => {
  await signedIn(page, '/')
  const header = page.getByRole('banner')
  const nav = page.getByRole('navigation')
  await expect(header.getByText('Hệ thống quản trị doanh nghiệp')).toBeVisible()
  await expect(nav).toHaveCount(0)

  // Administration is a tile too; its menu holds no HRM item, and HRM's none of its.
  await page.getByRole('main').getByRole('link', { name: 'Quản trị' }).click()
  await expect(page).toHaveURL(/\/admin\/org-units$/)
  await expect(nav.getByRole('link', { name: 'Cơ cấu tổ chức' })).toHaveCount(0)
  await header.getByRole('link', { name: 'Trang chủ' }).click()
  await page.getByRole('main').getByRole('link', { name: 'Quản lý nhân sự' }).click()
  await expect(nav.getByRole('link', { name: 'Cây tổ chức' })).toHaveCount(0)

  // The inbox is in the header on every page and opens inside the current area.
  await header.getByRole('link', { name: /^Hộp duyệt/ }).click()
  await expect(page).toHaveURL(/\/hrm\/inbox$/)
  await expect(header.getByText('Quản lý nhân sự')).toBeVisible()
  await expect(nav).toHaveCount(1)

  await page.goto('/hrm/overview')
  await expect(header.getByText('Quản lý nhân sự')).toBeVisible()
  await page.goto('/admin/users')
  await expect(header.getByText('Quản trị', { exact: true })).toBeVisible()
  // From home it has no area.
  await page.goto('/inbox')
  await expect(header.getByText('Hệ thống quản trị doanh nghiệp')).toBeVisible()
  await expect(nav).toHaveCount(0)
})

test('the HRM menu is grouped and empty groups are hidden', async ({ page }) => {
  await signInAs(page, menuViewer, '/hrm/overview')
  const nav = page.getByRole('navigation')
  await expect(nav.getByRole('group', { name: 'Nhân sự' }).getByRole('link', { name: 'Nhân viên' })).toBeVisible()
  await expect(nav.getByRole('group', { name: 'Chấm công & nghỉ phép' }).getByRole('link', { name: 'Đơn nghỉ' })).toBeVisible()
  await expect(nav.getByRole('group', { name: 'Tiền lương' })).toHaveCount(0)
  await expect(nav.getByRole('group', { name: 'Cấu hình' })).toHaveCount(0)
})

test('an administrator folds Configuration and it stays folded after a reload', async ({ page }) => {
  await signedIn(page, '/hrm/overview')
  const config = page.getByRole('navigation').getByRole('group', { name: 'Cấu hình' })
  await expect(config.getByRole('link', { name: 'Loại nghỉ' })).toBeVisible()
  await config.getByRole('button', { name: 'Cấu hình' }).click()
  await expect(config.getByRole('link', { name: 'Loại nghỉ' })).toHaveCount(0)
  await page.reload()
  await expect(page.getByRole('navigation').getByRole('group', { name: 'Cấu hình' }).getByRole('link', { name: 'Loại nghỉ' })).toHaveCount(0)
  await page.getByRole('navigation').getByRole('button', { name: 'Cấu hình' }).click()
  await expect(page.getByRole('navigation').getByRole('link', { name: 'Loại nghỉ' })).toBeVisible()
})

test('on a phone the drawer shows the groups and closes on a choice', async ({ browser }) => {
  const page = await (await browser.newContext({ viewport: { width: 375, height: 800 } })).newPage()
  await signedIn(page, '/hrm/overview')
  await page.getByRole('button', { name: 'Mở thanh điều hướng' }).click()
  const nav = page.getByRole('navigation')
  await expect(nav.getByRole('group', { name: 'Chấm công & nghỉ phép' })).toBeVisible()
  await nav.getByRole('link', { name: 'Đơn nghỉ' }).click()
  await expect(page).toHaveURL(/\/hrm\/leaves$/)
  await expect(nav.getByRole('link', { name: 'Đơn nghỉ' })).not.toBeInViewport()
})

test('with HRM disabled, its data stays readable behind a read-only banner', async ({ browser, page: withHrm }) => {
  await signedIn(withHrm, '/hrm/employees')
  await expect(withHrm.getByText('Phân hệ đang tắt: chỉ xem và xuất được dữ liệu.')).toHaveCount(0)

  // Cookies ignore the port, so the server without hrm gets its own browser context.
  const page = await (await browser.newContext()).newPage()
  await signedIn(page, `http://localhost:8091/hrm/employees?q=M9A${run}`)
  await expect(page.getByText('Phân hệ đang tắt: chỉ xem và xuất được dữ liệu.')).toBeVisible()
  await expect(page.getByRole('banner').getByText('Quản lý nhân sự', { exact: true })).toBeVisible()
  await page.getByRole('link', { name: `M9A${run}-0` }).click()
  await expect(page.getByRole('heading', { name: `Nhân viên M9A${run}-0` })).toBeVisible()
  await expect(page.getByText('Phân hệ đang tắt: chỉ xem và xuất được dữ liệu.')).toBeVisible()
})

test('with HRM disabled, no screen offers to create, import or edit', async ({ browser, page: withHrm }) => {
  const writes = /^(Tạo|Thêm|Nhập|Lập|Hợp đồng mới)/
  await signedIn(withHrm, '/hrm/employees')
  await expect(withHrm.getByRole('main').getByRole('link', { name: writes })).toHaveCount(1)

  const page = await (await browser.newContext()).newPage()
  await signedIn(page, 'http://localhost:8091/hrm/employees')
  for (const path of ['employees', 'leaves', 'overtimes', 'contracts', 'timesheets', 'payrolls', 'leave-types', 'contract-types', 'work-calendar', 'legal-params']) {
    await page.goto(`http://localhost:8091/hrm/${path}`)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    await expect(page.getByRole('main').getByRole('button', { name: writes }), path).toHaveCount(0)
    await expect(page.getByRole('main').getByRole('link', { name: writes }), path).toHaveCount(0)
    await expect(page.getByRole('main').getByRole('button', { name: 'Thao tác khác' }).first(), path).toBeHidden()
  }
  await page.goto(`http://localhost:8091/hrm/employees?q=M9A${run}`)
  await page.getByRole('link', { name: `M9A${run}-0` }).click()
  await expect(page.getByRole('heading', { name: `Nhân viên M9A${run}-0` })).toBeVisible()
  await expect(page.getByRole('main').getByRole('button', { name: /^(Lưu|Thêm)/ })).toHaveCount(0)

  // Export stays; import goes.
  await page.goto(`http://localhost:8091/hrm/timesheets/${timesheet}`)
  await expect(page.getByRole('button', { name: 'Xuất Excel' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Nhập từ Excel' })).toHaveCount(0)

  // The home page still offers HRM, read-only.
  await page.goto('http://localhost:8091/')
  await page.getByRole('main').getByRole('link', { name: 'Quản lý nhân sự' }).click()
  await expect(page).toHaveURL(/:8091\/hrm\/overview$/)
  await expect(page.getByText('Phân hệ đang tắt: chỉ xem và xuất được dữ liệu.')).toBeVisible()
})

test('home shows the products the user may enter; Back walks list, overview, home', async ({ page }) => {
  await signInAs(page, hr, '/')
  await expect(page).toHaveURL(/\/$/)
  await page.getByRole('main').getByRole('link', { name: 'Quản lý nhân sự' }).click()
  await expect(page).toHaveURL(/\/hrm\/overview$/)
  await tile(page, 'Nhân sự đang làm việc').click()
  await expect(page).toHaveURL(/\/hrm\/employees\?status=active$/)
  await page.goBack()
  await expect(page).toHaveURL(/\/hrm\/overview$/)
  await page.goBack()
  await expect(page).toHaveURL(/\/$/)
})

test('a user without HRM permissions has no HRM tile', async ({ page }) => {
  await signInAs(page, emp, '/')
  await expect(page.getByRole('main').getByRole('link', { name: 'Quản lý nhân sự' })).toHaveCount(0)
  await expect(page.getByText('Bạn chưa được cấp quyền vào phân hệ nào.')).toBeVisible()
})

test('a deep link to a filtered list opens after signing in', async ({ page }) => {
  await signInAs(page, hr, '/hrm/employees?status=active')
  await expect(page).toHaveURL(/\/hrm\/employees\?status=active$/)
  await expect(page.getByText('3 dòng')).toBeVisible()
})

test('on a phone the home grid shows, then the HRM drawer opens', async ({ browser }) => {
  const page = await (await browser.newContext({ viewport: { width: 375, height: 800 } })).newPage()
  await signedIn(page, '/')
  await expect(page.getByRole('button', { name: 'Mở thanh điều hướng' })).toHaveCount(0)
  await page.getByRole('main').getByRole('link', { name: 'Quản lý nhân sự' }).click()
  await page.getByRole('button', { name: 'Mở thanh điều hướng' }).click()
  await expect(page.getByRole('navigation').getByRole('link', { name: 'Tổng quan' })).toBeVisible()
})
