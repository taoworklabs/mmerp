import { expect, test, type APIRequestContext } from '@playwright/test'
import { created, createUser, employee, password, signedInApi, runId } from './api'

// Department "Phòng EI" has a fixed name because the committed Excel files (go run
// ./scripts/e2e-fixtures) name it; a rerun on the same database reuses it. hr is HR of it.
const hr = `eihr_${runId()}`
let api: APIRequestContext
let unit: number

const fixture = (name: string) => new URL(`fixtures/${name}`, import.meta.url).pathname

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  const units = (await (await api.get('/api/org-units')).json()) as { id: number; name: string }[]
  unit = units?.find((u) => u.name === 'Phòng EI')?.id ?? 0
  if (!unit) {
    const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: 'Công ty EI' } }))
    unit = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: 'Phòng EI' } }))
  }
  const res = await api.post('/api/hrm/employees', { data: employee('EI01', unit) })
  expect([201, 409], await res.text()).toContain(res.status())
  await createUser(api, hr, [{ role: 'hrm.hr', unit }])
})

test('HR exports the employee list, imports a file that fixes one profile and adds one, and sees a bad file refused', async ({ page }) => {
  await page.goto('/')
  await page.getByLabel('Tên đăng nhập').fill(hr)
  await page.getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: /^Tài khoản:/ })).toBeVisible()
  await page.goto('/hrm/employees')

  // The export finishes in the background and turns into the download.
  await page.getByRole('button', { name: 'Xuất Excel' }).click()
  const download = page.waitForEvent('download')
  await page.getByRole('link', { name: 'Tải file' }).click()
  expect((await download).suggestedFilename()).toMatch(/\.xlsx$/)

  const open = async () => {
    await page.getByRole('button', { name: 'Thao tác khác' }).click()
    await page.getByRole('menuitem', { name: 'Nhập hồ sơ từ Excel' }).click()
    return page.getByRole('dialog')
  }
  const upload = async (name: string) => {
    const dialog = await open()
    await dialog.locator('input[type=file]').setInputFiles(fixture(name))
    await dialog.getByRole('button', { name: 'Nhập', exact: true }).click()
    return dialog
  }

  // A department that does not exist: the row is listed and nothing is written.
  const bad = await upload('employees-bad.xlsx')
  await expect(bad.getByRole('alert')).toContainText('File có 1 dòng lỗi')
  await bad.getByRole('button', { name: 'Đóng', exact: true }).click()

  const good = await upload('employees.xlsx')
  await expect(good.getByText('Đã nhập 2 dòng.')).toBeVisible()
  await good.getByRole('button', { name: 'Đóng', exact: true }).click()
  await page.getByRole('searchbox').fill('EI02')
  await page.getByRole('searchbox').press('Enter')
  await expect(page.getByRole('link', { name: /Nhân viên EI02/ })).toBeVisible()

  const list = (await (await api.get('/api/hrm/employees?q=EI01')).json()) as { items: { id: number; code: string }[] }
  const id = list.items.find((e) => e.code === 'EI01')!.id
  expect(((await (await api.get(`/api/hrm/employees/${id}`)).json()) as { phone: string }).phone).toBe('0900000001')
})
