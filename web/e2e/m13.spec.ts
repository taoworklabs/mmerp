import { expect, test, type APIRequestContext } from '@playwright/test'
import { created, createUser, employee, password, signedInApi } from './api'
import { signedIn } from './helpers'

// One run: department Q, where pay (HR with the payroll role) prints a draft contract.
const run = Date.now().toString(36)
const login = { pay: `m13pay_${run}` }
let api: APIRequestContext
let contract: { id: number; number: string }

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty M13 ${run}` } }))
  const q = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng Q ${run}` } }))
  await createUser(api, login.pay, [
    { role: 'hrm.hr', unit: q },
    { role: 'hrm.payroll', unit: q },
  ])
  const e = await created(api.post('/api/hrm/employees', { data: employee(`M13E${run}`, q) }))
  const kind = await created(api.post('/api/hrm/contract-types', { data: { name: `Không xác định thời hạn ${run}`, fixed_term: false, active: true } }))
  const id = await created(
    api.post('/api/hrm/contracts', {
      data: { employee_id: e, contract_type_id: kind, start_date: '2026-01-01', end_date: null, terms: { salary: 9_000_000, lines: [] } },
    }),
  )
  const asPay = await signedInApi(baseURL!, login.pay, password)
  contract = { id, number: ((await (await asPay.get(`/api/hrm/contracts/${id}`)).json()) as { number: string }).number }
})

test('a contract prints to a PDF the requester downloads', async ({ browser }) => {
  const page = await (await browser.newContext()).newPage()
  await page.goto(`/hrm/contracts/${contract.id}`)
  await page.getByLabel('Tên đăng nhập').fill(login.pay)
  await page.getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()

  await page.getByRole('button', { name: 'In hợp đồng' }).click()
  const link = page.getByRole('link', { name: 'Tải file' })
  const href = (await link.getAttribute('href'))!
  const download = page.waitForEvent('download')
  await link.click()
  expect((await download).suggestedFilename()).toBe(`${contract.number}.pdf`)
  const again = await page.request.get(href)
  expect(again.headers()['content-type']).toBe('application/pdf')
  expect((await again.text()).startsWith('%PDF-')).toBe(true)
})

test('an administrator rewords a contract clause, saved as a new version', async ({ page }) => {
  await signedIn(page, '/admin/print-templates')
  const row = page.getByRole('row', { name: /Hợp đồng lao động/ })
  await row.getByRole('button', { name: 'Thao tác khác' }).click()
  await page.getByRole('menuitem', { name: 'Sửa lời văn' }).click()
  const clause = page.getByRole('textbox', { name: 'Điều khoản hợp đồng (tiếng Việt)' })
  await clause.fill('Người lao động {employee_name} tuân thủ nội quy của {employer_name}. Lần chạy ' + run)
  await page.getByRole('button', { name: 'Lưu phiên bản mới' }).click()
  await expect(page.getByText('Đã lưu phiên bản mới của mẫu in.')).toBeVisible()
  await expect(row.getByText(/Phiên bản \d+/)).toBeVisible()

  await row.getByRole('button', { name: 'Thao tác khác' }).click()
  await page.getByRole('menuitem', { name: 'Sửa lời văn' }).click()
  await clause.fill('Lương {salary}')
  await page.getByRole('button', { name: 'Lưu phiên bản mới' }).click()
  await expect(page.getByText('Lời văn dùng chỗ điền không có trong mẫu: salary.')).toBeVisible()
})
