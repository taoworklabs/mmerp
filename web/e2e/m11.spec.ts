import { expect, test, type APIRequestContext, type Browser, type Page } from '@playwright/test'
import { created, createUser, employee, password, signedInApi } from './api'

// One run: department Q; pay (HR with the payroll role) handles contracts, hr (no salaries)
// sees them without their attachments. staff files a leave request; boss is their manager.
const run = Date.now().toString(36)
const login = { pay: `m11pay_${run}`, hr: `m11hr_${run}`, staff: `m11staff_${run}`, boss: `m11boss_${run}` }
let api: APIRequestContext
let ids: { contract: number; leave: number }

const fixture = (name: string) => new URL(`fixtures/${name}`, import.meta.url).pathname
const scan = '%PDF-1.4\n% Hop dong lao dong da ky - ban scan mau cho e2e\n%%EOF\n'

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty M11 ${run}` } }))
  const q = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng Q ${run}` } }))
  await createUser(api, login.pay, [
    { role: 'hrm.hr', unit: q },
    { role: 'hrm.payroll', unit: q },
  ])
  await createUser(api, login.hr, [{ role: 'hrm.hr', unit: q }])
  const e = await created(api.post('/api/hrm/employees', { data: employee(`M11E${run}`, q) }))
  const kind = await created(api.post('/api/hrm/contract-types', { data: { name: `Không xác định thời hạn ${run}`, fixed_term: false, active: true } }))
  const contract = await created(
    api.post('/api/hrm/contracts', {
      data: { employee_id: e, contract_type_id: kind, start_date: '2026-01-01', end_date: null, terms: { salary: 9_000_000, lines: [] } },
    }),
  )
  for (const l of [login.staff, login.boss]) await createUser(api, l, [])
  const boss = await created(api.post('/api/hrm/employees', { data: employee(`M11B${run}`, q, { user_login: login.boss }) }))
  await created(api.post('/api/hrm/employees', { data: employee(`M11S${run}`, q, { user_login: login.staff, manager_id: boss }) }))
  const type = await created(api.post('/api/hrm/leave-types', { data: { name: `Nghỉ M11 ${run}`, deducts_balance: false, paid: true, active: true } }))
  const asStaff = await signedInApi(baseURL!, login.staff, password)
  const leave = await created(asStaff.post('/api/hrm/leaves', { data: { leave_type_id: type, start_date: '2026-06-01', end_date: '2026-06-01', days: '1', reason: null } }))
  ids = { contract, leave }
})

async function as(browser: Browser, user: string, url: string): Promise<Page> {
  const page = await (await browser.newContext()).newPage()
  await page.goto(url)
  await page.getByLabel('Tên đăng nhập').fill(user)
  await page.getByLabel(/^Mật khẩu/).fill(password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: 'Tài khoản' })).toBeVisible()
  return page
}

test.describe('attachments', () => {
  test('a signed contract is attached and downloads unchanged; without salary rights it stays hidden', async ({ browser }) => {
    const pay = await as(browser, login.pay, `/hrm/contracts/${ids.contract}`)
    const chooser = pay.waitForEvent('filechooser')
    await pay.getByRole('button', { name: 'Thêm file' }).click()
    await (await chooser).setFiles(fixture('contract-scan.pdf'))
    await expect(pay.getByText('Đính kèm (1)')).toBeVisible()

    const link = pay.getByRole('link', { name: 'contract-scan.pdf' })
    const download = pay.waitForEvent('download')
    await link.click()
    expect((await download).suggestedFilename()).toBe('contract-scan.pdf')
    const again = await pay.request.get((await link.getAttribute('href'))!)
    expect(await again.text()).toBe(scan)

    const hr = await as(browser, login.hr, `/hrm/contracts/${ids.contract}`)
    await expect(hr.getByText('Không có quyền xem đính kèm của bản ghi này.')).toBeVisible()
    await expect(hr.getByRole('link', { name: 'contract-scan.pdf' })).toHaveCount(0)
  })
})

test.describe('discussion', () => {
  test('the submitter and their manager talk on a leave request, each seeing the other', async ({ browser }) => {
    const url = `/hrm/leaves/${ids.leave}`
    const staff = await as(browser, login.staff, url)
    await staff.getByRole('textbox', { name: 'Bình luận' }).fill('Em xin nghỉ khám bệnh.\nGiấy hẹn em gửi kèm.')
    await staff.getByRole('button', { name: 'Gửi bình luận' }).click()
    await expect(staff.getByText('Trao đổi (1)')).toBeVisible()

    const boss = await as(browser, login.boss, url)
    await expect(boss.getByText('Giấy hẹn em gửi kèm.', { exact: false })).toBeVisible()
    await boss.getByRole('textbox', { name: 'Bình luận' }).fill('Ai làm thay việc của em?')
    await boss.getByRole('button', { name: 'Gửi bình luận' }).click()
    await expect(boss.getByText('Trao đổi (2)')).toBeVisible()

    await staff.reload()
    await expect(staff.getByText('Ai làm thay việc của em?')).toBeVisible()
    await expect(staff.getByText(login.boss, { exact: false }).first()).toBeVisible()
  })
})
