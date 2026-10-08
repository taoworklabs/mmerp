import { expect, test, type Page } from '@playwright/test'
import { created, createUser, employee, password, signedInApi } from './api'
import { admin } from './helpers'

// One run: a company with departments A and B, employees in both, and an HR user scoped to A.
const run = Date.now().toString(36)
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

async function signInAs(page: Page, login: string, url: string) {
  await page.goto(url)
  await page.getByLabel('Tên đăng nhập').fill(login)
  await page.getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: 'Tài khoản' })).toBeVisible()
}

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
  await page.getByRole('button', { name: `Hiện ${field}` }).click()
  await expect(page.getByRole('textbox', { name: field })).toHaveValue(nationalId)
  // Hide masks it again; showing it again is another audited read.
  await page.getByRole('button', { name: `Ẩn ${field}` }).click()
  await expect(page.getByText(nationalId)).toHaveCount(0)
  await expect(page.getByRole('textbox', { name: field })).toHaveCount(0)
  await page.getByRole('button', { name: `Hiện ${field}` }).click()
  await expect(page.getByRole('textbox', { name: field })).toHaveValue(nationalId)

  // The administrator revokes the sensitive-data role while the page is open.
  const admin = await signedInApi(baseURL!)
  const grants: { id: number; role: string }[] = await (await admin.get(`/api/users/${ids.hrA}/roles`)).json()
  const grant = grants.find((g) => g.role === 'sensitive_viewer')!
  expect((await admin.delete(`/api/users/${ids.hrA}/roles/${grant.id}`)).status()).toBe(204)

  // The next request carries a new X-Authz-Version: the page reloads with nothing stale left.
  const reloaded = page.waitForEvent('load')
  await page.getByRole('navigation').getByRole('link', { name: 'Nhân viên' }).click()
  await reloaded
  await page.goto(`/hrm/employees/${ids.ea}`)
  await expect(page.getByText('Không có quyền xem').first()).toBeVisible()
  await expect(page.getByRole('button', { name: `Hiện ${field}` })).toHaveCount(0)
  await expect(page.locator(`input[value="${nationalId}"]`)).toHaveCount(0)

  // Restore for the other tests.
  await created(admin.post(`/api/users/${ids.hrA}/roles`, { data: { product: 'hrm', role: 'sensitive_viewer', org_unit_id: ids.a } }))
})

test('filters, sort and page live on the URL and Back restores each step', async ({ page }) => {
  await signInAs(page, hrA, '/hrm/employees')
  const table = page.getByRole('table', { name: 'Nhân viên' })

  await page.getByLabel('Tìm kiếm').fill(run)
  await page.getByLabel('Tìm kiếm').press('Enter')
  await expect(page).toHaveURL(new RegExp(`\\?q=${run}$`))
  await expect(table.getByRole('row')).toHaveCount(26) // header + 25

  await page.getByRole('button', { name: 'Họ tên' }).click()
  await expect(page).toHaveURL(/sort=full_name/)
  await page.getByRole('button', { name: 'Họ tên' }).click()
  await expect(page).toHaveURL(/sort=-full_name/)
  await expect(table.getByRole('row').nth(1)).toContainText(`A${run}-24`)

  await page.getByRole('combobox', { name: 'Số dòng mỗi trang' }).click()
  await page.getByRole('option', { name: '20 / trang' }).click()
  await expect(page).toHaveURL(/page_size=20/)
  await page.getByRole('button', { name: 'Trang 2' }).click()
  await expect(page).toHaveURL(/page=2/)
  await expect(table.getByRole('row')).toHaveCount(6)

  await page.goBack()
  await expect(page).not.toHaveURL(/page=2/)
  await expect(table.getByRole('row')).toHaveCount(21)
  await page.goBack()
  await expect(page).not.toHaveURL(/page_size/)
  await page.goBack()
  await expect(page).toHaveURL(/sort=full_name/)
  await expect(table.getByRole('row').nth(1)).toContainText(`A${run}-00`)
  await page.goBack()
  await page.goBack()
  await expect(page).toHaveURL(/\/hrm\/employees$/)
  await expect(page.getByLabel('Tìm kiếm')).toHaveValue('')
})

test('below 1024px the employee list becomes cards and the filters move behind a button', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 800 })
  await signInAs(page, hrA, `/hrm/employees?q=${run}-0`)
  const cards = page.getByRole('list', { name: 'Nhân viên' })
  await expect(cards.getByRole('listitem').first()).toBeVisible()
  await expect(page.getByRole('table')).toHaveCount(0)
  await page.getByRole('button', { name: 'Lọc' }).click()
  await expect(page.getByRole('dialog').getByLabel('Tìm kiếm')).toBeVisible()
})

test('an administrator builds the tree and grants a role from the admin screens', async ({ page }) => {
  await signInAs(page, admin.login, '/admin/org-units')
  await page.getByRole('button', { name: 'Tạo đơn vị' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('combobox', { name: 'Loại' }).click()
  await page.getByRole('option', { name: 'Pháp nhân' }).click()
  await dialog.getByRole('textbox', { name: 'Tên', exact: true }).fill(`UI ${run}`)
  await dialog.getByLabel('Mã số thuế').fill('0309999999')
  await dialog.getByRole('button', { name: 'Lưu' }).click()
  await expect(page.getByRole('table', { name: 'Cây tổ chức' }).getByText(`UI ${run}`)).toBeVisible()

  await page.getByRole('navigation').getByRole('link', { name: 'Người dùng' }).click()
  await page.getByRole('button', { name: 'Tạo người dùng' }).click()
  await page.getByRole('dialog').getByLabel('Tên đăng nhập').fill(`ui_${run}`)
  await page.getByRole('dialog').getByLabel('Họ tên').fill(`UI ${run}`)
  await page.getByRole('dialog').getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('dialog').getByRole('button', { name: 'Tạo người dùng' }).click()
  await expect(page.getByRole('heading', { name: `UI ${run}` })).toBeVisible()

  await page.getByRole('button', { name: 'Cấp vai trò' }).click()
  await page.getByRole('dialog').getByRole('combobox', { name: 'Vai trò' }).click()
  await page.getByRole('option', { name: 'Xem hồ sơ nhân viên' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Cấp vai trò' }).click()
  await expect(page.getByRole('table', { name: 'Vai trò' }).getByText('Xem hồ sơ nhân viên')).toBeVisible()
  await expect(page.getByRole('table', { name: 'Vai trò' }).getByText('Toàn công ty')).toBeVisible()
})

test('HR creates an employee with a sensitive field, edits it and adds a dependent', async ({ page }) => {
  await signInAs(page, hrA, '/hrm/employees')
  await page.getByRole('link', { name: 'Tạo nhân viên' }).click()
  await page.getByRole('textbox', { name: 'Mã nhân viên' }).fill(`N${run}`)
  await page.getByRole('textbox', { name: 'Họ tên' }).fill('Lê Văn Mới')
  await page.getByRole('combobox', { name: 'Phòng ban' }).click()
  await page.getByRole('option', { name: `Phòng A ${run}` }).click()
  await page.getByRole('textbox', { name: 'Ngày vào làm' }).fill('05/01/2026')
  await page.getByRole('textbox', { name: 'Số CCCD hoặc hộ chiếu' }).fill('001099000111')
  await page.getByRole('button', { name: 'Tạo nhân viên' }).click()

  await expect(page.getByRole('heading', { name: 'Lê Văn Mới' })).toBeVisible()
  // The toast pauses while hovered; move away so it closes after 3 seconds.
  await expect(page.getByText('Đã tạo nhân viên.')).toBeVisible()
  await page.mouse.move(0, 0)
  await expect(page.getByText('Đã tạo nhân viên.')).toBeHidden()
  await expect(page.getByRole('textbox', { name: 'Ngày vào làm' })).toHaveValue('05/01/2026')
  await expect(page.getByText('••••••').first()).toBeVisible()

  await page.getByRole('textbox', { name: 'Họ tên' }).fill('Lê Văn Đổi')
  await page.getByRole('button', { name: 'Lưu hồ sơ' }).click()
  await expect(page.getByRole('heading', { name: 'Lê Văn Đổi' })).toBeVisible()

  await page.getByRole('tab', { name: 'Người phụ thuộc' }).click()
  await expect(page).toHaveURL(/tab=dependents/)
  await page.getByRole('button', { name: 'Thêm người phụ thuộc' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('textbox', { name: 'Họ tên' }).fill('Lê Bé')
  await dialog.getByRole('combobox', { name: 'Quan hệ' }).click()
  await page.getByRole('option', { name: 'Con' }).click()
  await dialog.getByRole('textbox', { name: 'Giảm trừ từ tháng' }).fill('2026-01')
  await dialog.getByRole('button', { name: 'Lưu' }).click()
  await expect(page.getByRole('table', { name: 'Người phụ thuộc' }).getByText('Lê Bé')).toBeVisible()
})

test('leaving a record asks only when something was edited', async ({ page }) => {
  await signInAs(page, hrA, `/hrm/employees/${ids.ea}`)
  const nav = page.getByRole('navigation').getByRole('link', { name: 'Nhân viên' })
  await nav.click()
  await expect(page).toHaveURL(/\/hrm\/employees$/)
  await expect(page.getByRole('dialog')).toHaveCount(0)

  await page.goto(`/hrm/employees/${ids.ea}`)
  await page.getByRole('textbox', { name: 'Họ tên' }).fill('Tên đang sửa')
  await nav.click()
  await expect(page.getByRole('dialog', { name: 'Rời trang?' })).toBeVisible()
  await page.getByRole('button', { name: 'Quay lại' }).click()
  await expect(page).toHaveURL(new RegExp(`/hrm/employees/${ids.ea}$`))

  // Typing into a revealed sensitive field counts too.
  await page.goto(`/hrm/employees/${ids.ea}`)
  await page.getByRole('button', { name: 'Hiện Số CCCD hoặc hộ chiếu' }).click()
  await page.getByRole('textbox', { name: 'Số CCCD hoặc hộ chiếu' }).fill('123')
  await nav.click()
  await expect(page.getByRole('dialog', { name: 'Rời trang?' })).toBeVisible()
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
  const manager = page.getByRole('combobox', { name: 'Quản lý trực tiếp' })
  await expect(manager).toHaveValue(`M${run} · Nhân viên M${run}`)
  searches = 0
  await page.waitForTimeout(1500)
  await expect(manager).toHaveValue(`M${run} · Nhân viên M${run}`)
  expect(searches).toBe(0)
})
