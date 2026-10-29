import { expect, test, type Page } from '@playwright/test'
import { created, createUser, employee, password, signedInApi, runId, toFlow } from './api'
import { as } from './helpers'

// One run: staff files leave requests; boss, their manager, approves them.
const run = runId()
const login = { staff: `ntstaff_${run}`, boss: `ntboss_${run}` }
let leave: number

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty NT ${run}` } }))
  const q = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng NT ${run}` } }))
  for (const l of [login.staff, login.boss]) await createUser(api, l, [])
  const boss = await created(api.post('/api/hrm/employees', { data: employee(`NTB${run}`, q, { user_login: login.boss }) }))
  await created(api.post('/api/hrm/employees', { data: employee(`NTS${run}`, q, { user_login: login.staff, manager_id: boss }) }))
  const rule = await api.put('/api/approval-rules/hrm.leave_request', {
    data: { flow: toFlow([{ approver: { kind: 'module' } }]), max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' },
  })
  expect(rule.status(), await rule.text()).toBe(204)
  const type = await created(api.post('/api/hrm/leave-types', { data: { name: `Nghỉ NT ${run}`, deducts_balance: false, paid: true, active: true } }))
  const asStaff = await signedInApi(baseURL!, login.staff, password)
  leave = await created(asStaff.post('/api/hrm/leaves', { data: { leave_type_id: type, start_date: '2026-07-01', end_date: '2026-07-01', days: '1', reason: null } }))
})

const bell = (page: Page) => page.getByRole('link', { name: /^Thông báo/ })

test('sending and approving a leave request tells the other side, and the unread count follows', async ({ browser }) => {
  const staff = await as(browser, login.staff, `/me/requests/leave/${leave}`)
  await expect(bell(staff)).toHaveAccessibleName('Thông báo')
  await staff.getByRole('button', { name: 'Gửi duyệt' }).click()
  await expect(staff.getByText('Chờ duyệt').first()).toBeVisible()

  const boss = await as(browser, login.boss, '/')
  await expect(bell(boss)).toHaveAccessibleName('Thông báo, 1 chưa đọc')
  await bell(boss).click()
  await boss.getByRole('button', { name: /Chờ bạn duyệt/ }).click()
  await expect(boss).toHaveURL(new RegExp(`/me/requests/leave/${leave}$`))
  await expect(bell(boss)).toHaveAccessibleName('Thông báo')
  await boss.getByRole('button', { name: 'Duyệt', exact: true }).click()
  await expect(boss.getByText('Đã duyệt.')).toBeVisible()

  await staff.goto('/notifications')
  await expect(bell(staff)).toHaveAccessibleName('Thông báo, 1 chưa đọc')
  await staff.getByRole('button', { name: /Đã được duyệt/ }).click()
  await expect(staff).toHaveURL(new RegExp(`/me/requests/leave/${leave}$`))
  await expect(bell(staff)).toHaveAccessibleName('Thông báo')
})

test('a mention picked after @ tells the mentioned user, who opens the request from it', async ({ browser }) => {
  const staff = await as(browser, login.staff, `/me/requests/leave/${leave}`)
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
  await expect(boss).toHaveURL(new RegExp(`/me/requests/leave/${leave}$`))
})
