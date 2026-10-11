import { expect, test, type Browser, type Page } from '@playwright/test'
import { created, createUser, employee, password, signedInApi } from './api'
import { signedIn } from './helpers'

// Probes for the parts of notifications and the approval inbox the other specs leave out:
// the notification and job links staying in the area (TB-01), approving something the
// submitter withdrew (TB-04), a notification whose permission was revoked (TB-05) and going
// back to a filtered list (TB-07). They mostly observe and log, because some of the expected
// results are still being agreed. The TB codes label the observations.

const run = Date.now().toString(36)
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

// TB-04: the submitter withdraws while the approver still has the inbox open.
test('TB-04 approving a request the submitter already withdrew', async ({ browser }) => {
  const staff = await as(browser, login.staff, `/hrm/leaves/${leave}`)
  await staff.getByRole('button', { name: 'Gửi duyệt' }).click()
  await expect(staff.getByText('Chờ duyệt').first()).toBeVisible()

  const boss = await as(browser, login.boss, '/hrm/inbox')
  // Record whether the click reaches the server at all, and with what answer.
  const calls: string[] = []
  boss.on('response', (r) => {
    if (r.url().includes('/approve')) calls.push(`${r.request().method()} ${new URL(r.url()).pathname} → ${r.status()}`)
  })
  await boss.getByRole('button', { name: /Chờ bạn duyệt|TB|NP-/ }).first().click()
  await expect(boss.getByRole('button', { name: 'Duyệt', exact: true })).toBeVisible()

  // The submitter withdraws behind the approver's back.
  await staff.getByRole('button', { name: 'Rút lại' }).click()
  await expect(staff.getByText('Nháp').first()).toBeVisible()

  await boss.getByRole('button', { name: 'Duyệt', exact: true }).click()
  // A toast renders outside main, so read the whole page and look for the withdrawn text.
  const withdrawn = boss.getByText(/rút lại lần duyệt này|đã được xử lý|approval_closed/)
  const explained = await withdrawn.first().isVisible({ timeout: 3000 }).catch(() => false)
  console.log('TB-04 an explanation of the state is shown:', explained)
  if (explained) {
    // The message can vanish between being seen and being read: the panel holding it unmounts.
    const text = await withdrawn.first().innerText({ timeout: 2000 }).catch((e) => `(gone before it could be read: ${e.message.split('\n')[0]})`)
    console.log('TB-04 the exact wording shown:', text.replace(/\s+/g, ' '))
    console.log('TB-04 still visible a second later:', await withdrawn.first().isVisible().catch(() => false))
  }
  console.log('TB-04 approve calls the click produced:', calls.length ? calls.join(', ') : '(none — the click never reached the server)')
  console.log('TB-04 the whole page after approving:', (await boss.locator('body').innerText()).replace(/\s+/g, ' ').slice(0, 500))
})

// TB-04 through the API, because the inbox drops the item asynchronously and the UI
// probe above is therefore timing-dependent: this pins down what the server answers.
test('TB-04 the API refusal when approving a withdrawn request', async ({ baseURL }) => {
  const admin = await signedInApi(baseURL!)
  const type = await created(admin.post('/api/hrm/leave-types', { data: { name: `Nghỉ TB4 ${run}`, deducts_balance: false, paid: true, active: true } }))
  const asStaff = await signedInApi(baseURL!, login.staff, password)
  const id = await created(asStaff.post('/api/hrm/leaves', { data: { leave_type_id: type, start_date: '2026-08-10', end_date: '2026-08-10', days: '1', reason: null } }))

  const doc = await (await asStaff.get(`/api/hrm/leaves/${id}`)).json()
  const sent = await asStaff.post(`/api/documents/hrm.leave_request/${id}/transitions`, { data: { to: 'posted', version: doc.version } })
  console.log('TB-04 submit:', sent.status(), (await sent.text()).slice(0, 200))

  const asBoss = await signedInApi(baseURL!, login.boss, password)
  const inbox = await (await asBoss.get('/api/approvals/inbox')).json()
  const item = inbox.items.find((i: { doc_id: number }) => i.doc_id === id)
  console.log('TB-04 the approver sees it in the inbox:', JSON.stringify(item))

  const after = await (await asStaff.get(`/api/hrm/leaves/${id}`)).json()
  const pulled = await asStaff.post(`/api/documents/hrm.leave_request/${id}/transitions`, { data: { to: 'draft', version: after.version } })
  console.log('TB-04 the submitter withdraws:', pulled.status(), (await pulled.text()).slice(0, 200))

  // The approver acts on the instance and step they saw before the withdrawal.
  const res = await asBoss.post(`/api/approvals/${item.instance_id}/approve`, { data: { step: item.step } })
  console.log('TB-04 approving afterwards →', res.status(), (await res.text()).slice(0, 300))
  console.log('TB-04 the inbox afterwards holds this request:',
    (await (await asBoss.get('/api/approvals/inbox')).json()).items.some((i: { doc_id: number }) => i.doc_id === id))
  console.log('TB-04 the document ended up as:', (await (await asStaff.get(`/api/hrm/leaves/${id}`)).json()).status)
})

// TB-04 from the document page rather than the inbox: the panel is not inside a subtree that
// disappears with the item, so this isolates where the lost message comes from.
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
  const message = boss.getByText(/rút lại lần duyệt này|đã được xử lý|approval_closed|không còn/)
  const shown = await message.first().isVisible({ timeout: 3000 }).catch(() => false)
  console.log('TB-04 from the document page, a message is shown:', shown)
  if (shown) console.log('TB-04 the wording on the document page:', (await message.first().innerText()).replace(/\s+/g, ' '))
  console.log('TB-04 the document page after approving:', (await boss.getByRole('main').innerText()).replace(/\s+/g, ' ').slice(0, 400))
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
  // The status filter is a Mantine Select, so it is an input carrying the chosen label.
  const filter = page.locator('input[aria-label="Trạng thái"]')
  const shown = async () => `count=${await filter.count()} value=${await filter.first().inputValue().catch((e) => `(${e.message.split('\n')[0]})`)}`
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
