import { expect, test, type Browser, type Page } from '@playwright/test'
import { created, createUser, employee, password, signedInApi } from './api'
import { signedIn } from './helpers'

// One run: staff files leave requests; boss, their manager, approves them.
const run = Date.now().toString(36)
const login = { staff: `m12staff_${run}`, boss: `m12boss_${run}` }
let leave: number

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty M12 ${run}` } }))
  const q = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng M12 ${run}` } }))
  for (const l of [login.staff, login.boss]) await createUser(api, l, [])
  const boss = await created(api.post('/api/hrm/employees', { data: employee(`M12B${run}`, q, { user_login: login.boss }) }))
  await created(api.post('/api/hrm/employees', { data: employee(`M12S${run}`, q, { user_login: login.staff, manager_id: boss }) }))
  const rule = await api.put('/api/approval-rules/hrm.leave_request', {
    data: { steps: [{ approver: { kind: 'module' } }], max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' },
  })
  expect(rule.status(), await rule.text()).toBe(204)
  const type = await created(api.post('/api/hrm/leave-types', { data: { name: `Nghỉ M12 ${run}`, deducts_balance: false, paid: true, active: true } }))
  const asStaff = await signedInApi(baseURL!, login.staff, password)
  leave = await created(asStaff.post('/api/hrm/leaves', { data: { leave_type_id: type, start_date: '2026-07-01', end_date: '2026-07-01', days: '1', reason: null } }))
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

const bell = (page: Page) => page.getByRole('link', { name: /^Thông báo/ })

test('sending and approving a leave request tells the other side, and the unread count follows', async ({ browser }) => {
  const staff = await as(browser, login.staff, `/hrm/leaves/${leave}`)
  await expect(bell(staff)).toHaveAccessibleName('Thông báo')
  await staff.getByRole('button', { name: 'Gửi duyệt' }).click()
  await expect(staff.getByText('Chờ duyệt').first()).toBeVisible()

  const boss = await as(browser, login.boss, '/')
  await expect(bell(boss)).toHaveAccessibleName('Thông báo, 1 chưa đọc')
  await bell(boss).click()
  await boss.getByRole('button', { name: /Chờ bạn duyệt/ }).click()
  await expect(boss).toHaveURL(new RegExp(`/hrm/leaves/${leave}$`))
  await expect(bell(boss)).toHaveAccessibleName('Thông báo')
  await boss.getByRole('button', { name: 'Duyệt', exact: true }).click()
  await expect(boss.getByText('Đã duyệt.')).toBeVisible()

  await staff.goto('/notifications')
  await expect(bell(staff)).toHaveAccessibleName('Thông báo, 1 chưa đọc')
  await staff.getByRole('button', { name: /Đã được duyệt/ }).click()
  await expect(staff).toHaveURL(new RegExp(`/hrm/leaves/${leave}$`))
  await expect(bell(staff)).toHaveAccessibleName('Thông báo')
})

test('a mention picked after @ tells the mentioned user, who opens the request from it', async ({ browser }) => {
  const staff = await as(browser, login.staff, `/hrm/leaves/${leave}`)
  const box = staff.getByRole('textbox', { name: 'Bình luận' })
  await box.fill('Nhờ anh xem giúp ')
  await box.pressSequentially(`@${login.boss.slice(0, 7)}`)
  await staff.getByRole('option', { name: new RegExp(login.boss) }).click()
  await expect(box).toHaveValue(`Nhờ anh xem giúp @${login.boss} `)
  await staff.getByRole('button', { name: 'Gửi bình luận' }).click()
  await expect(staff.getByText('Nhờ anh xem giúp')).toBeVisible()

  const boss = await as(browser, login.boss, '/notifications')
  await expect(bell(boss)).toHaveAccessibleName('Thông báo, 1 chưa đọc')
  await boss.getByRole('button', { name: /Nhắc tên bạn trong trao đổi/ }).click()
  await expect(boss).toHaveURL(new RegExp(`/hrm/leaves/${leave}$`))
})

test('an administrator sets the mail server; its password never shows again', async ({ browser }) => {
  const page = await (await browser.newContext()).newPage()
  await signedIn(page, '/admin/mail-server')
  await page.getByRole('button', { name: 'Cấu hình máy chủ mail' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Máy chủ').fill('smtp.example.com')
  await dialog.getByLabel(/^Mật khẩu/).fill('s3cret pass')
  await dialog.getByLabel('Địa chỉ gửi').fill('mmerp@example.com')
  await dialog.getByRole('button', { name: 'Lưu' }).click()
  await expect(page.getByText('Đã lưu cấu hình máy chủ mail.')).toBeVisible()
  await expect(page.getByText('smtp.example.com:587')).toBeVisible()
  await expect(page.getByText('s3cret pass')).toHaveCount(0)

  await page.getByRole('button', { name: 'Xóa cấu hình' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Xóa cấu hình' }).click()
  await expect(page.getByText('Chưa cấu hình máy chủ mail.')).toBeVisible()
})
