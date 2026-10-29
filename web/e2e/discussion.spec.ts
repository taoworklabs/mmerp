import { expect, test } from '@playwright/test'
import { created, createUser, employee, password, signedInApi, runId } from './api'
import { as } from './helpers'

// One run: staff files a leave request; boss is their manager.
const run = runId()
const login = { staff: `dsstaff_${run}`, boss: `dsboss_${run}` }
let ids: { leave: number }

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty DS ${run}` } }))
  const q = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng Q ${run}` } }))
  for (const l of [login.staff, login.boss]) await createUser(api, l, [])
  const boss = await created(api.post('/api/hrm/employees', { data: employee(`DSB${run}`, q, { user_login: login.boss }) }))
  await created(api.post('/api/hrm/employees', { data: employee(`DSS${run}`, q, { user_login: login.staff, manager_id: boss }) }))
  const type = await created(api.post('/api/hrm/leave-types', { data: { name: `Nghỉ DS ${run}`, deducts_balance: false, paid: true, active: true } }))
  const asStaff = await signedInApi(baseURL!, login.staff, password)
  const leave = await created(asStaff.post('/api/hrm/leaves', { data: { leave_type_id: type, start_date: '2026-06-01', end_date: '2026-06-01', days: '1', reason: null } }))
  ids = { leave }
})

test.describe('discussion', () => {
  test('the submitter and their manager talk on a leave request, each seeing the other', async ({ browser }) => {
    const url = `/me/requests/leave/${ids.leave}`
    const staff = await as(browser, login.staff, url)
    await staff.getByRole('textbox', { name: 'Bình luận' }).fill('Em xin nghỉ khám bệnh.\nGiấy hẹn em gửi kèm.')
    await staff.getByRole('button', { name: 'Gửi bình luận' }).click()
    // The comment joins the document's activity feed.
    await expect(staff.getByText('Giấy hẹn em gửi kèm.', { exact: false })).toBeVisible()

    const boss = await as(browser, login.boss, url)
    await expect(boss.getByText('Giấy hẹn em gửi kèm.', { exact: false })).toBeVisible()
    await boss.getByRole('textbox', { name: 'Bình luận' }).fill('Ai làm thay việc của em?')
    await boss.getByRole('button', { name: 'Gửi bình luận' }).click()
    await expect(boss.getByText('Ai làm thay việc của em?')).toBeVisible()

    await staff.reload()
    await expect(staff.getByText('Ai làm thay việc của em?')).toBeVisible()
    await expect(staff.getByText(login.boss, { exact: false }).first()).toBeVisible()
  })
})
