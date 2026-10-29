import { expect, test } from '@playwright/test'
import { created, employee, signedInApi, runId } from './api'
import { signedIn } from './helpers'

// One run: a department with 3 employees and a draft timesheet, read on :8091 where no product
// is enabled.
const run = runId()
let timesheet: number

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty PD ${run}`, tax_code: '0101234567' } }))
  const a = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng A PD ${run}` } }))
  for (let i = 0; i < 3; i++) await created(api.post('/api/hrm/employees', { data: employee(`PDA${run}-${i}`, a) }))
  timesheet = await created(api.post('/api/hrm/timesheets', { data: { org_unit_id: a, month: '2026-09' } }))
})

test('with HRM disabled, its data stays readable behind a read-only banner', async ({ browser, page: withHrm }) => {
  await signedIn(withHrm, '/hrm/employees')
  await expect(withHrm.getByText('Phân hệ đang tắt: chỉ xem và xuất được dữ liệu.')).toHaveCount(0)

  // Cookies ignore the port, so the server without hrm gets its own browser context.
  const page = await (await browser.newContext()).newPage()
  await signedIn(page, `http://localhost:8091/hrm/employees?q=PDA${run}`)
  await expect(page.getByText('Phân hệ đang tắt: chỉ xem và xuất được dữ liệu.')).toBeVisible()
  await expect(page.getByRole('banner').getByText('Nhân sự', { exact: true })).toBeVisible()
  await page.getByRole('link', { name: `PDA${run}-0` }).click()
  await expect(page.getByRole('heading', { name: `Nhân viên PDA${run}-0` })).toBeVisible()
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
    // A "more" menu may stay for exports, but offers nothing that writes.
    const more = page.getByRole('main').getByRole('button', { name: 'Thao tác khác' })
    if (await more.count()) {
      await more.first().click()
      await expect(page.getByRole('menuitem').first(), path).toBeVisible()
      await expect(page.getByRole('menuitem', { name: writes }), path).toHaveCount(0)
      await page.keyboard.press('Escape')
    }
  }
  await page.goto(`http://localhost:8091/hrm/employees?q=PDA${run}`)
  await page.getByRole('link', { name: `PDA${run}-0` }).click()
  await expect(page.getByRole('heading', { name: `Nhân viên PDA${run}-0` })).toBeVisible()
  await expect(page.getByRole('main').getByRole('button', { name: /^(Lưu|Thêm)/ })).toHaveCount(0)

  // Export stays; import goes.
  await page.goto(`http://localhost:8091/hrm/timesheets/${timesheet}`)
  await expect(page.getByRole('button', { name: 'Xuất Excel' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Nhập từ Excel' })).toHaveCount(0)

  // The home page still offers HRM, read-only.
  await page.goto('http://localhost:8091/')
  await page.getByRole('main').getByRole('link', { name: 'Nhân sự' }).click()
  await expect(page).toHaveURL(/:8091\/hrm\/overview$/)
  await expect(page.getByText('Phân hệ đang tắt: chỉ xem và xuất được dữ liệu.')).toBeVisible()
})
