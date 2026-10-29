import { expect, test } from '@playwright/test'
import { created, createUser, employee, signedInApi, runId } from './api'
import { signInAs } from './helpers'

// One run: department A with one employee holding a national ID, and an HR user of A who may
// read sensitive fields.
const run = runId()
const hrA = `hr_a_${run}`
let ids: { ea: number }

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty ${run}`, tax_code: '0101234567' } }))
  const a = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng A ${run}` } }))
  const ea = await created(api.post('/api/hrm/employees', { data: employee(`A${run}-00`, a, { sensitive: { national_id: '079123456789' } }) }))
  await createUser(api, hrA, [
    { role: 'hrm.hr', unit: a },
    { role: 'hrm.sensitive_viewer', unit: a },
  ])
  ids = { ea }
})

test('HR creates an employee with a sensitive field, edits it and adds a dependent', async ({ page }) => {
  await signInAs(page, hrA, '/hrm/employees')
  await page.getByRole('link', { name: 'Tạo nhân viên' }).click()
  await page.getByRole('textbox', { name: 'Mã nhân viên' }).fill(`N${run}`)
  await page.getByRole('textbox', { name: 'Họ tên' }).fill('Lê Văn Mới')
  await page.getByRole('combobox', { name: 'Phòng ban' }).click()
  await page.getByRole('option', { name: `Phòng A ${run}` }).click()
  await page.getByRole('textbox', { name: 'Ngày vào làm' }).fill('05/01/2026')
  await page.getByRole('button', { name: 'Thêm thông tin nhạy cảm' }).click()
  await expect(page.getByRole('textbox', { name: 'Số CCCD hoặc hộ chiếu' })).toBeFocused()
  await page.getByRole('textbox', { name: 'Số CCCD hoặc hộ chiếu' }).fill('001099000111')
  await page.getByRole('button', { name: 'Tạo nhân viên' }).click()

  await expect(page.getByRole('heading', { name: 'Lê Văn Mới' })).toBeVisible()
  // The toast pauses while hovered; move away so it closes after 3 seconds.
  await expect(page.getByText('Đã tạo nhân viên.')).toBeVisible()
  await page.mouse.move(0, 0)
  await expect(page.getByText('Đã tạo nhân viên.')).toBeHidden()
  const main = page.getByRole('region', { name: 'Thông tin chính' })
  await expect(main.getByText('05/01/2026')).toBeVisible()
  await expect(page.getByText('••••••').first()).toBeVisible()

  // Each part of the profile is edited on its own, in place in its card.
  await main.getByRole('button', { name: 'Sửa Thông tin chính' }).click()
  await main.getByRole('textbox', { name: 'Họ tên' }).fill('Lê Văn Đổi')
  await main.getByRole('button', { name: 'Lưu' }).click()
  await expect(page.getByRole('heading', { name: 'Lê Văn Đổi' })).toBeVisible()
  await expect(main.getByRole('textbox', { name: 'Họ tên' })).toHaveCount(0)
  await expect(main.getByText(`Phòng A ${run}`)).toBeVisible()

  await page.getByRole('tab', { name: 'Người phụ thuộc' }).click()
  await expect(page).toHaveURL(/tab=dependents/)
  await page.getByRole('button', { name: 'Thêm người phụ thuộc' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('textbox', { name: 'Họ tên' }).fill('Lê Bé')
  await dialog.getByRole('combobox', { name: 'Quan hệ' }).click()
  await page.getByRole('option', { name: 'Con' }).click()
  // Months are picked, not typed: January of the year the picker opens on.
  await dialog.getByRole('button', { name: 'Giảm trừ từ tháng' }).click()
  await page.getByRole('gridcell', { name: /^(thg|Tháng) 1$/i }).click()
  await dialog.getByRole('button', { name: 'Thêm', exact: true }).click()
  await expect(page.getByRole('table', { name: 'Người phụ thuộc' }).getByText('Lê Bé')).toBeVisible()
})

test('the employee page tabs live on the URL and survive a reload', async ({ page }) => {
  await signInAs(page, hrA, `/hrm/employees/${ids.ea}`)
  await expect(page.getByRole('tab', { name: 'Tổng quan' })).toHaveAttribute('aria-selected', 'true')
  await page.getByRole('tab', { name: 'Người phụ thuộc' }).click()
  await expect(page).toHaveURL(new RegExp(`/hrm/employees/${ids.ea}\\?tab=dependents$`))
  await page.reload()
  await expect(page.getByRole('tab', { name: 'Người phụ thuộc' })).toHaveAttribute('aria-selected', 'true')
  await expect(page.getByRole('button', { name: 'Thêm người phụ thuộc' })).toBeVisible()
  // The first tab is the bare URL.
  await page.getByRole('tab', { name: 'Tổng quan' }).click()
  await expect(page).toHaveURL(new RegExp(`/hrm/employees/${ids.ea}$`))
  // Links to the activity card it replaced open the activity tab.
  await page.goto(`/hrm/employees/${ids.ea}?tab=discussion`)
  await expect(page.getByRole('tab', { name: 'Hoạt động' })).toHaveAttribute('aria-selected', 'true')
  await expect(page.getByRole('group', { name: 'Lọc hoạt động' })).toBeVisible()
})

test('leaving a record asks only when something was edited', async ({ page }) => {
  await signInAs(page, hrA, `/hrm/employees/${ids.ea}`)
  const nav = page.getByRole('navigation', { name: 'Điều hướng chính' }).getByRole('link', { name: 'Nhân viên' })
  await nav.click()
  await expect(page).toHaveURL(/\/hrm\/employees$/)
  await expect(page.getByRole('dialog')).toHaveCount(0)

  // The way out is the browser's Back, within the app: the record is opened from the list, not
  // loaded afresh.
  const openFromList = async () => {
    await page.goto(`/hrm/employees?q=A${run}-00`)
    await page.getByRole('table', { name: 'Nhân viên' }).getByRole('link', { name: `A${run}-00` }).click()
    await expect(page).toHaveURL(new RegExp(`/hrm/employees/${ids.ea}$`))
  }
  await openFromList()
  await page.getByRole('button', { name: 'Sửa Thông tin chính' }).click()
  await page.getByRole('textbox', { name: 'Họ tên' }).fill('Tên đang sửa')
  await page.evaluate(() => history.back())
  await expect(page.getByRole('dialog', { name: 'Rời trang?' })).toBeVisible()
  await page.getByRole('button', { name: 'Quay lại' }).click()
  await expect(page).toHaveURL(new RegExp(`/hrm/employees/${ids.ea}$`))

  // Typing into a revealed sensitive field counts too.
  await openFromList()
  await page.getByRole('button', { name: 'Sửa Thông tin nhạy cảm' }).click()
  const sensitive = page.getByRole('region', { name: 'Thông tin nhạy cảm' })
  await sensitive.getByRole('button', { name: 'Hiện Số CCCD hoặc hộ chiếu' }).click()
  await sensitive.getByRole('textbox', { name: 'Số CCCD hoặc hộ chiếu' }).fill('123')
  await page.evaluate(() => history.back())
  await expect(page.getByRole('dialog', { name: 'Rời trang?' })).toBeVisible()
})
