import { expect, test, type Browser, type Page } from '@playwright/test'
import { created, createUser, employee, password, signedInApi, runId } from './api'
import { signedIn } from './helpers'

// Covers the parts of notifications and the approval inbox the other specs leave out: the
// notification and job links staying in the area (TB-01), a notification opening its own
// document with the permission checked as it opens (TB-02), approving something the submitter
// withdrew (TB-04), a notification whose permission was revoked (TB-05) and going back to a
// filtered list (TB-07). TB-05 still observes and logs, since what should happen to a
// notification nobody may read any more is not settled; the rest assert.

const run = runId()
const login = { staff: `tbstaff_${run}`, boss: `tbboss_${run}`, peer: `tbpeer_${run}` }
let dept: number
let leave: number
let peerId: number

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty TB ${run}` } }))
  dept = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng TB ${run}` } }))
  await createUser(api, login.staff, [])
  await createUser(api, login.boss, [])
  // The peer may view the request through the HR role, until it is revoked.
  peerId = await createUser(api, login.peer, [{ role: 'hrm.hr', unit: dept }])
  const boss = await created(api.post('/api/hrm/employees', { data: employee(`TBB${run}`, dept, { user_login: login.boss }) }))
  await created(api.post('/api/hrm/employees', { data: employee(`TBS${run}`, dept, { user_login: login.staff, manager_id: boss }) }))
  const rule = await api.put('/api/approval-rules/hrm.leave_request', {
    data: { steps: [{ approver: { kind: 'module' } }], max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' },
  })
  expect(rule.status(), await rule.text()).toBe(204)
  const type = await created(api.post('/api/hrm/leave-types', { data: { name: `Nghỉ TB ${run}`, deducts_balance: false, paid: true, active: true } }))
  const asStaff = await signedInApi(baseURL!, login.staff, password)
  leave = await created(asStaff.post('/api/hrm/leaves', { data: { leave_type_id: type, start_date: '2026-08-03', end_date: '2026-08-03', days: '1', reason: null } }))
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

// TB-01 for the two header links no other spec asserts: notifications and background jobs.
test('TB-01 the notification and job links open inside the current area', async ({ page }) => {
  await signedIn(page, '/hrm/leaves')
  const header = page.getByRole('banner')
  const nav = page.getByRole('navigation')

  await header.getByRole('link', { name: /^Thông báo/ }).click()
  await expect(page).toHaveURL(/\/hrm\/notifications$/)
  await expect(header.getByText('Quản lý nhân sự')).toBeVisible()
  await expect(nav).toHaveCount(1)

  await header.getByRole('link', { name: /^Việc chạy nền/ }).click()
  await expect(page).toHaveURL(/\/hrm\/jobs$/)
  await expect(header.getByText('Quản lý nhân sự')).toBeVisible()

  // From home the same links sit at the root, with no area menu.
  await page.goto('/')
  await header.getByRole('link', { name: /^Thông báo/ }).click()
  await expect(page).toHaveURL(/\/notifications$/)
  await expect(nav).toHaveCount(0)
  await page.goto('/')
  await header.getByRole('link', { name: /^Việc chạy nền/ }).click()
  await expect(page).toHaveURL(/\/jobs$/)
  await expect(nav).toHaveCount(0)
})

// TB-04: the submitter withdraws while the approver still has the inbox open. The refusal has
// to be explained where the approver is looking, and the explanation has to stay put: the
// request leaves the inbox at that moment, and the panel explaining it must not leave with it.
test('TB-04 approving a request the submitter already withdrew', async ({ browser }) => {
  const staff = await as(browser, login.staff, `/hrm/leaves/${leave}`)
  await staff.getByRole('button', { name: 'Gửi duyệt' }).click()
  await expect(staff.getByText('Chờ duyệt').first()).toBeVisible()

  const boss = await as(browser, login.boss, '/hrm/inbox')
  const calls: string[] = []
  boss.on('response', (r) => {
    if (r.url().includes('/approve')) calls.push(String(r.status()))
  })
  await boss.getByRole('button', { name: /Chờ bạn duyệt|TB|NP-/ }).first().click()
  await expect(boss.getByRole('button', { name: 'Duyệt', exact: true })).toBeVisible()

  // The submitter withdraws behind the approver's back.
  await staff.getByRole('button', { name: 'Rút lại' }).click()
  await expect(staff.getByText('Nháp').first()).toBeVisible()

  await boss.getByRole('button', { name: 'Duyệt', exact: true }).click()
  const withdrawn = boss.getByText(/rút lại lần duyệt này|đã được xử lý/).first()
  await expect(withdrawn).toBeVisible()
  // Still there a moment later: a message that only survives one frame is no message.
  await boss.waitForTimeout(1500)
  await expect(withdrawn).toBeVisible()
  // And the button that cannot work any more is gone.
  await expect(boss.getByRole('button', { name: 'Duyệt', exact: true })).toHaveCount(0)
  expect(calls, 'the click reached the server and was refused').toEqual(['409'])
})

// TB-04 through the API as well, to pin the answer itself rather than how it is shown.
test('TB-04 the API refusal when approving a withdrawn request', async ({ baseURL }) => {
  const admin = await signedInApi(baseURL!)
  const type = await created(admin.post('/api/hrm/leave-types', { data: { name: `Nghỉ TB4 ${run}`, deducts_balance: false, paid: true, active: true } }))
  const asStaff = await signedInApi(baseURL!, login.staff, password)
  const id = await created(asStaff.post('/api/hrm/leaves', { data: { leave_type_id: type, start_date: '2026-08-10', end_date: '2026-08-10', days: '1', reason: null } }))

  const doc = await (await asStaff.get(`/api/hrm/leaves/${id}`)).json()
  const sent = await asStaff.post(`/api/documents/hrm.leave_request/${id}/transitions`, { data: { to: 'posted', version: doc.version } })
  expect(sent.status(), await sent.text()).toBe(204)

  const asBoss = await signedInApi(baseURL!, login.boss, password)
  const inbox = await (await asBoss.get('/api/approvals/inbox')).json()
  const item = inbox.items.find((i: { doc_id: number }) => i.doc_id === id)
  expect(item, 'the approver has it waiting').toBeTruthy()

  const after = await (await asStaff.get(`/api/hrm/leaves/${id}`)).json()
  const pulled = await asStaff.post(`/api/documents/hrm.leave_request/${id}/transitions`, { data: { to: 'draft', version: after.version } })
  expect(pulled.status(), await pulled.text()).toBe(204)

  // The approver acts on the instance and step they saw before the withdrawal.
  const res = await asBoss.post(`/api/approvals/${item.instance_id}/approve`, { data: { step: item.step } })
  expect(res.status()).toBe(409)
  expect((await res.json()).code).toBe('approval_closed')
  const left = (await (await asBoss.get('/api/approvals/inbox')).json()).items
  expect(left.some((i: { doc_id: number }) => i.doc_id === id), 'it no longer waits for anyone').toBe(false)
  expect((await (await asStaff.get(`/api/hrm/leaves/${id}`)).json()).status, 'the withdrawal stands').toBe('draft')
})

// The same refusal from the document page, which reaches it by a different route than the
// inbox: here nothing ever removes the panel, so it guards the wording itself.
test('TB-04 the same refusal seen from the document page', async ({ browser, baseURL }) => {
  const admin = await signedInApi(baseURL!)
  const type = await created(admin.post('/api/hrm/leave-types', { data: { name: `Nghỉ TB4b ${run}`, deducts_balance: false, paid: true, active: true } }))
  const asStaff = await signedInApi(baseURL!, login.staff, password)
  const id = await created(asStaff.post('/api/hrm/leaves', { data: { leave_type_id: type, start_date: '2026-08-17', end_date: '2026-08-17', days: '1', reason: null } }))
  const doc = await (await asStaff.get(`/api/hrm/leaves/${id}`)).json()
  expect((await asStaff.post(`/api/documents/hrm.leave_request/${id}/transitions`, { data: { to: 'posted', version: doc.version } })).status()).toBe(204)

  const boss = await as(browser, login.boss, `/hrm/leaves/${id}`)
  const approve = boss.getByRole('button', { name: 'Duyệt', exact: true })
  await expect(approve).toBeVisible()

  // Withdraw through the API so the approver's page is untouched until they click.
  const now = await (await asStaff.get(`/api/hrm/leaves/${id}`)).json()
  expect((await asStaff.post(`/api/documents/hrm.leave_request/${id}/transitions`, { data: { to: 'draft', version: now.version } })).status()).toBe(204)

  await approve.click()
  await expect(boss.getByText('Người gửi đã rút lại lần duyệt này.')).toBeVisible()
  await expect(boss.getByRole('main').getByText('Nháp').first()).toBeVisible()
})

// TB-02: a notification opens the document it names, and whether the user may still view it
// is decided when it opens, not when the list was loaded.
test('TB-02 a notification opens its own document and re-checks the permission', async ({ browser, baseURL }) => {
  const admin = await signedInApi(baseURL!)
  const reader = `tbread_${run}`
  const readerId = await createUser(admin, reader, [{ role: 'hrm.hr', unit: dept }])
  const type = await created(admin.post('/api/hrm/leave-types', { data: { name: `Nghỉ TB2 ${run}`, deducts_balance: false, paid: true, active: true } }))
  const asStaff = await signedInApi(baseURL!, login.staff, password)
  const mentioned = async (day: string) => {
    const id = await created(asStaff.post('/api/hrm/leaves', { data: { leave_type_id: type, start_date: day, end_date: day, days: '1', reason: null } }))
    const said = await asStaff.post(`/api/records/hrm.leave_request/${id}/comments`, { data: { body: `Nhờ xem giúp @${reader}` } })
    expect(said.status(), await said.text()).toBe(204)
    return { id, number: (await (await asStaff.get(`/api/hrm/leaves/${id}`)).json()).number as string }
  }
  const first = await mentioned('2026-08-24')
  const second = await mentioned('2026-08-25')

  // Two notifications of the same kind: the one clicked is the one opened.
  const page = await as(browser, reader, '/notifications')
  const row = (number: string) => page.getByRole('button').filter({ hasText: number })
  await row(first.number).click()
  await expect(page).toHaveURL(new RegExp(`/hrm/leaves/${first.id}$`))
  await expect(page.getByRole('main').getByText(first.number).first()).toBeVisible()

  // Revoked while the list still shows the second one in full.
  await page.goBack()
  await expect(row(second.number)).toBeVisible()
  const grants: { id: number }[] = await (await admin.get(`/api/users/${readerId}/roles`)).json()
  for (const g of grants) expect((await admin.delete(`/api/users/${readerId}/roles/${g.id}`)).status()).toBe(204)
  // The changed permissions reload the page on the next answer, which may land back on the list
  // or on the document; either way the request's content is gone.
  await row(second.number).click()
  await expect(page.getByRole('main').getByText(/bạn không còn quyền xem|Không tìm thấy dữ liệu, hoặc không có quyền xem/).first()).toBeVisible()
  await expect(page.getByRole('main').getByText(second.number)).toHaveCount(0)
})

// TB-05: a notification whose record the user may no longer view.
test('TB-05 a notification after the permission was revoked', async ({ browser, baseURL }) => {
  const staff = await as(browser, login.staff, `/hrm/leaves/${leave}`)
  const box = staff.getByRole('textbox', { name: 'Bình luận' })
  await box.fill('Nhờ anh xem giúp ')
  await box.pressSequentially(`@${login.peer.slice(0, 7)}`)
  await staff.getByRole('option', { name: new RegExp(login.peer) }).click()
  await staff.getByRole('button', { name: 'Gửi bình luận' }).click()
  await expect(staff.getByText('Nhờ anh xem giúp')).toBeVisible()

  const peer = await as(browser, login.peer, '/notifications')
  // Wait for the mention to arrive, or the before-picture is of an empty list.
  await expect(peer.getByText(/Nhắc tên bạn/).first()).toBeVisible()
  console.log('TB-05 with permission, the notification reads:', (await peer.getByRole('main').innerText()).replace(/\s+/g, ' ').slice(0, 300))

  const api = await signedInApi(baseURL!)
  const grants: { id: number }[] = await (await api.get(`/api/users/${peerId}/roles`)).json()
  for (const g of grants) expect((await api.delete(`/api/users/${peerId}/roles/${g.id}`)).status()).toBe(204)

  const after = await as(browser, login.peer, '/notifications')
  console.log('TB-05 after the role was revoked:', (await after.getByRole('main').innerText()).replace(/\s+/g, ' ').slice(0, 300))
  const item = after.getByRole('button').filter({ hasText: /Nhắc tên|không còn quyền/ }).first()
  if (await item.count()) {
    await item.click()
    await after.waitForTimeout(500)
    console.log('TB-05 clicking it goes to:', after.url(), '| main:', (await after.getByRole('main').innerText()).replace(/\s+/g, ' ').slice(0, 200))
  } else {
    console.log('TB-05 no notification row to click')
  }
})

// TB-07: going back from a document to the list it was opened from.
test('TB-07 back from a document restores the filtered list', async ({ page }) => {
  // draft: the request TB-04 withdrew is one, so the filtered list has a row.
  await signedIn(page, '/hrm/leaves?status=draft')
  const before = page.url()
  const row = page.getByRole('table').getByRole('link').first()
  await expect(row).toBeVisible()
  // The status filter is a chip that names its value ("Trạng thái: Nháp").
  const filter = page.getByRole('button', { name: /^Trạng thái/ })
  const shown = async () => `count=${await filter.count()} text=${await filter.first().textContent().catch((e) => `(${e.message.split('\n')[0]})`)}`
  console.log('TB-07 the filter control before opening:', await shown())
  const rowCount = await page.getByRole('table').getByRole('link').count()
  await row.click()
  await expect(page).toHaveURL(/\/hrm\/leaves\/\d+$/)
  await page.goBack()
  console.log('TB-07 back went to:', page.url(), '| opened from:', before)
  await expect(page).toHaveURL(/status=draft/)
  await expect(page.getByRole('table').getByRole('link').first()).toBeVisible()
  console.log('TB-07 the filter control after going back:', await shown())
  console.log('TB-07 rows before/after:', rowCount, await page.getByRole('table').getByRole('link').count())
})
