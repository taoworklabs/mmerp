import { expect, test, type APIRequestContext, type Browser, type Page } from '@playwright/test'
import { created, createUser, employee, password, signedInApi } from './api'

// One run: department P with M5E01, M5E02 and M5E03 (who joins on 16/03), the codes the
// committed Excel files use (go run ./scripts/e2e-fixtures). hr is HR of P; dir approves timesheets.
const run = Date.now().toString(36)
const login = { hr: `m5hr_${run}`, dir: `m5dir_${run}` }
const fixture = (name: string) => new URL(`fixtures/${name}`, import.meta.url).pathname
let api: APIRequestContext

// ensureEmployee creates an employee, or moves the one a previous run on the same database made.
async function ensureEmployee(code: string, unit: number, hire: string) {
  const data = employee(code, unit, { hire_date: hire })
  const res = await api.post('/api/hrm/employees', { data })
  if (res.status() === 201) return
  expect(res.status(), await res.text()).toBe(409)
  const list = (await (await api.get(`/api/hrm/employees?q=${code}`)).json()) as { items: { id: number; code: string }[] }
  const id = list.items.find((e) => e.code === code)!.id
  const put = await api.put(`/api/hrm/employees/${id}`, { data })
  expect(put.status(), await put.text()).toBe(204)
}

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty M5 ${run}` } }))
  const p = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng P ${run}` } }))
  await ensureEmployee('M5E01', p, '2026-01-05')
  await ensureEmployee('M5E02', p, '2026-01-05')
  await ensureEmployee('M5E03', p, '2026-03-16')
  await createUser(api, login.hr, [{ role: 'hrm.hr', unit: p }])
  const dir = await createUser(api, login.dir, [{ role: 'hrm.hr', unit: null }])
  const rule = await api.put('/api/approval-rules/hrm.timesheet', {
    data: { steps: [{ approver: { kind: 'user', user_id: dir } }], max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' },
  })
  expect(rule.status(), await rule.text()).toBe(204)
})

async function as(browser: Browser, user: string, url: string) {
  const page = await (await browser.newContext()).newPage()
  await page.goto(url)
  await page.getByLabel('Tên đăng nhập').fill(user)
  await page.getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: 'Tài khoản' })).toBeVisible()
  return page
}

// total reads the Total cell of an employee's row in the grid.
const total = (page: Page, code: string) =>
  page.getByRole('table', { name: 'Ngày công theo nhân viên và ngày' }).getByRole('row', { name: new RegExp(`^${code}`) }).getByRole('cell').last()

async function importFile(page: Page, name: string) {
  const dialog = page.getByRole('dialog')
  await dialog.locator('input[type=file]').setInputFiles(fixture(name))
  await dialog.getByRole('button', { name: 'Nhập', exact: true }).click()
  return dialog
}

test('HR imports the March timesheet from Excel, fixes a bad file, follows the job and gets it approved', async ({ browser }) => {
  const hr = await as(browser, login.hr, '/hrm/timesheets')
  await hr.getByRole('button', { name: 'Tạo bảng công' }).click()
  const create = hr.getByRole('dialog')
  await create.getByRole('combobox', { name: 'Đơn vị' }).click()
  await hr.getByRole('option', { name: new RegExp(`Phòng P ${run}`) }).click()
  await create.getByRole('button', { name: /^Kỳ/ }).click()
  await hr.getByRole('button', { name: 'Th03' }).click()
  await create.getByRole('button', { name: 'Tạo bản nháp' }).click()
  await expect(hr.getByRole('heading', { name: /^BC-2026-/ })).toBeVisible()
  await expect(total(hr, 'M5E01')).toHaveText('0')

  // A file with bad rows lists each row to fix, and nothing is written.
  await hr.getByRole('button', { name: 'Nhập từ Excel' }).click()
  let dialog = await importFile(hr, 'timesheet-2026-03-bad.xlsx')
  await expect(dialog.getByRole('alert')).toContainText('File có 3 dòng lỗi')
  const errors = dialog.getByRole('table', { name: 'Các dòng lỗi' })
  await expect(errors.getByRole('row', { name: /^2 .*Ngày 3/ })).toBeVisible()
  await expect(errors.getByRole('row', { name: /^3 .*M5X99/ })).toBeVisible()
  await expect(errors.getByRole('row', { name: /^4 .*Ngày 2 nằm ngoài thời gian làm việc/ })).toBeVisible()

  // The right file: the job runs in the background and the dialog follows it to the end.
  await dialog.getByRole('button', { name: 'Chọn file khác' }).click()
  dialog = await importFile(hr, 'timesheet-2026-03.xlsx')
  await expect(dialog.getByText('Đã nhập 3 dòng.')).toBeVisible()
  await dialog.getByRole('button', { name: 'Đóng', exact: true }).click()
  await expect(total(hr, 'M5E01')).toHaveText('22')
  await expect(total(hr, 'M5E02')).toHaveText('24')
  await expect(total(hr, 'M5E03')).toHaveText('12')
  // Before joining, M5E03's cells take no value.
  await expect(hr.getByRole('textbox', { name: 'Nhân viên M5E03, 13/03/2026' })).toHaveCount(0)
  await expect(hr.getByRole('textbox', { name: 'Nhân viên M5E03, 16/03/2026' })).toHaveValue('1')
  const number = (await hr.getByRole('heading', { name: /^BC-2026-/ }).textContent())!

  // Both imports are in the user's own background jobs.
  await hr.getByRole('link', { name: 'Việc chạy nền' }).click()
  const jobs = hr.getByRole('table', { name: 'Việc chạy nền của tôi' })
  await expect(jobs.getByRole('row', { name: /Nhập Excel.*Bảng công.*Xong.*Đã nhập 3 dòng/ })).toBeVisible()
  await expect(jobs.getByRole('row', { name: /Nhập Excel.*Bảng công.*Thất bại.*Xem 3 dòng lỗi/ })).toBeVisible()
  await hr.goBack()

  await hr.getByRole('button', { name: 'Gửi duyệt' }).click()
  await expect(hr.getByRole('main').getByText('Chờ duyệt', { exact: true }).first()).toBeVisible()
  const dir = await as(browser, login.dir, '/inbox')
  await dir.getByRole('button', { name: new RegExp(number) }).click()
  await expect(dir.getByText('Tổng ngày công')).toBeVisible()
  await dir.getByRole('button', { name: 'Duyệt', exact: true }).click()
  await expect(dir.getByText('Đã duyệt.')).toBeVisible()
  await hr.reload()
  await expect(hr.getByRole('main').getByText('Đã chốt', { exact: true }).first()).toBeVisible()
  await expect(hr.getByRole('button', { name: 'Nhập từ Excel' })).toHaveCount(0)

  // The posted timesheet still exports, in the import's layout.
  await hr.getByRole('button', { name: 'Xuất Excel' }).click()
  const download = hr.waitForEvent('download')
  await hr.getByRole('link', { name: 'Tải file' }).click()
  expect((await download).suggestedFilename()).toBe(`${number}.xlsx`)
})
