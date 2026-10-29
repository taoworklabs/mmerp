import { expect, test, type Page } from '@playwright/test'
import { created, createUser, employee, signedInApi, runId } from './api'
import { signedIn, signInAs } from './helpers'

// One run: department A (3 employees) with an HR user and a menu viewer scoped to it, and a user
// with an employee record in B but no role.
const run = runId()
const hr = `nav_hr_${run}`
const menuViewer = `nav_menu_${run}`
const emp = `nav_emp_${run}` // no role: files leave only

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty NAV ${run}`, tax_code: '0101234567' } }))
  const a = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng A NAV ${run}` } }))
  const b = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng B NAV ${run}` } }))
  for (let i = 0; i < 3; i++) await created(api.post('/api/hrm/employees', { data: employee(`NAVA${run}-${i}`, a) }))
  await createUser(api, hr, [{ role: 'hrm.hr', unit: a }])
  await createUser(api, menuViewer, [{ role: 'hrm.viewer', unit: a }])
  await createUser(api, emp, [])
  await created(api.post('/api/hrm/employees', { data: employee(`NAVE${run}`, b, { user_login: emp }) }))
})

const tile = (page: Page, label: string) => page.getByRole('link', { name: new RegExp(`^${label}`) })

test('home has no sidebar; each area shows only its own menu; the header names the area', async ({ page }) => {
  await signedIn(page, '/')
  const header = page.getByRole('banner')
  const nav = page.getByRole('navigation')
  await expect(header.getByText('Hệ thống quản trị doanh nghiệp')).toBeVisible()
  await expect(nav).toHaveCount(0)

  // Administration is a tile too; its menu holds no HRM item, and HRM's none of its.
  await page.getByRole('main').getByRole('link', { name: 'Quản trị' }).click()
  await expect(page).toHaveURL(/\/admin\/org-units$/)
  await expect(nav.getByRole('link', { name: 'Sơ đồ tổ chức' })).toHaveCount(0)
  await header.getByRole('link', { name: 'Trang chủ' }).click()
  await page.getByRole('main').getByRole('link', { name: 'Nhân sự' }).click()
  await expect(nav.getByRole('link', { name: 'Cây tổ chức' })).toHaveCount(0)

  // The inbox is in the header on every page and opens inside the current area.
  await header.getByRole('link', { name: /^Hộp duyệt/ }).click()
  await expect(page).toHaveURL(/\/hrm\/inbox$/)
  await expect(header.getByText('Nhân sự')).toBeVisible()
  await expect(nav).toHaveCount(1)

  await page.goto('/hrm/overview')
  await expect(header.getByText('Nhân sự')).toBeVisible()
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
  await expect(nav.getByRole('group', { name: 'Nhân sự' }).getByRole('link', { name: 'Hồ sơ nhân viên' })).toBeVisible()
  await expect(nav.getByRole('group', { name: 'Chấm công & nghỉ phép' }).getByRole('link', { name: 'Đơn nghỉ phép' })).toBeVisible()
  await expect(nav.getByRole('group', { name: 'Tiền lương' })).toHaveCount(0)
  await expect(nav.getByRole('group', { name: 'Thiết lập' })).toHaveCount(0)
})

test('an administrator folds Configuration and it stays folded after a reload', async ({ page }) => {
  await signedIn(page, '/hrm/overview')
  const config = page.getByRole('navigation').getByRole('group', { name: 'Thiết lập' })
  await expect(config.getByRole('link', { name: 'Loại nghỉ phép' })).toBeVisible()
  await config.getByRole('button', { name: 'Thiết lập' }).click()
  await expect(config.getByRole('link', { name: 'Loại nghỉ phép' })).toHaveCount(0)
  await page.reload()
  await expect(page.getByRole('navigation').getByRole('group', { name: 'Thiết lập' }).getByRole('link', { name: 'Loại nghỉ phép' })).toHaveCount(0)
  await page.getByRole('navigation').getByRole('button', { name: 'Thiết lập' }).click()
  await expect(page.getByRole('navigation').getByRole('link', { name: 'Loại nghỉ phép' })).toBeVisible()
})

test('on a phone the drawer shows the groups and closes on a choice', async ({ browser }) => {
  const page = await (await browser.newContext({ viewport: { width: 375, height: 800 } })).newPage()
  await signedIn(page, '/hrm/overview')
  await page.getByRole('button', { name: 'Mở thanh điều hướng' }).click()
  const nav = page.getByRole('navigation')
  await expect(nav.getByRole('group', { name: 'Chấm công & nghỉ phép' })).toBeVisible()
  await nav.getByRole('link', { name: 'Đơn nghỉ phép' }).click()
  await expect(page).toHaveURL(/\/hrm\/leaves$/)
  await expect(nav.getByRole('link', { name: 'Đơn nghỉ phép' })).not.toBeInViewport()
})

test('home shows the products the user may enter; Back walks list, overview, home', async ({ page }) => {
  await signInAs(page, hr, '/')
  await expect(page).toHaveURL(/\/$/)
  await page.getByRole('main').getByRole('link', { name: 'Nhân sự' }).click()
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
  await expect(page.getByRole('main').getByRole('link', { name: 'Nhân sự' })).toHaveCount(0)
  await expect(page.getByRole('main').getByRole('link', { name: 'Cá nhân' })).toBeVisible()
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
  await page.getByRole('main').getByRole('link', { name: 'Nhân sự' }).click()
  await page.getByRole('button', { name: 'Mở thanh điều hướng' }).click()
  await expect(page.getByRole('navigation').getByRole('link', { name: 'Tổng quan' })).toBeVisible()
})
