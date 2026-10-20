import { expect, test, type APIRequestContext, type Browser, type Locator, type Page } from '@playwright/test'
import { created, createUser, employee, password, signedInApi, runId } from './api'
import { admin, signedIn } from './helpers'

// One run: approvers A and B, and employee E who files leave. The leave rule is tenant-wide, so
// this file runs alone and puts back a plain rule when done.
const run = runId()
const login = { a: `afa_${run}`, b: `afb_${run}`, e: `afe_${run}` }
const url = '/hrm/approval-rules/hrm.leave_request'
let api: APIRequestContext
let ids: { a: number; b: number; type: number }
let originalRule: unknown

test.beforeAll(async ({ baseURL }) => {
  api = await signedInApi(baseURL!)
  const rules = (await (await api.get('/api/approval-rules?product=hrm')).json()) as { doc_type: string; rule: unknown }[]
  originalRule = rules.find((r) => r.doc_type === 'hrm.leave_request')?.rule ?? null
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty AF ${run}` } }))
  const unit = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng AF ${run}` } }))
  // Approvers must be able to view the request.
  const a = await createUser(api, login.a, [{ role: 'hrm.hr', unit }])
  const b = await createUser(api, login.b, [{ role: 'hrm.hr', unit }])
  await createUser(api, login.e, [])
  await created(api.post('/api/hrm/employees', { data: employee(`AFE${run}`, unit, { user_login: login.e }) }))
  const type = await created(api.post('/api/hrm/leave-types', { data: { name: `Nghỉ AF ${run}`, deducts_balance: false, paid: true, active: true } }))
  ids = { a, b, type }
})

test.afterAll(async () => {
  // beforeAll may have failed before `api` was signed in; nothing to restore then.
  if (!api) return
  const res = originalRule ? await api.put('/api/approval-rules/hrm.leave_request', { data: originalRule }) : await api.delete('/api/approval-rules/hrm.leave_request')
  expect(res.status(), await res.text()).toBe(204)
})

async function as(browser: Browser, user: string, path: string) {
  const page = await (await browser.newContext()).newPage()
  await page.goto(path)
  await page.getByLabel('Tên đăng nhập').fill(user)
  await page.getByLabel(/^Mật khẩu/).fill(user === admin.login ? admin.password : password)
  await page.getByRole('button', { name: 'Đăng nhập' }).click()
  await expect(page.getByRole('button', { name: 'Tài khoản' })).toBeVisible()
  return page
}

const status = (page: Page, text: string) => expect(page.getByRole('main').getByText(text, { exact: true }).first()).toBeVisible()

// addTo opens a block's Add menu (the block's own, the last one inside scope) and picks an element.
async function addTo(scope: Locator, page: Page, element: string) {
  await scope.getByRole('button', { name: 'Thêm', exact: true }).last().click()
  await page.getByRole('menuitem', { name: element }).click()
}

async function removeStep(step: Locator, page: Page) {
  await step.getByRole('button', { name: 'Thao tác khác' }).click()
  await page.getByRole('menuitem', { name: 'Xóa bước' }).click()
}

async function pick(scope: Locator, page: Page, field: string, option: string) {
  await scope.getByRole('combobox', { name: field }).click()
  await page.getByRole('option', { name: option, exact: true }).click()
}

async function approverIsUser(branch: Locator, page: Page, user: string) {
  const step = branch.getByRole('region', { name: 'Bước duyệt' })
  await pick(step, page, 'Người duyệt', 'Người dùng cụ thể')
  await step.getByRole('combobox', { name: 'Người dùng' }).fill(user)
  await page.getByRole('option', { name: `${user} (${user})` }).click()
  await page.keyboard.press('Escape')
}

const step = (approver: unknown) => ({ step: { approver, approvals: 1, rejections: 1 } })

// The flow the first test builds: more than three days go to A and B in parallel, the rest to the manager.
const built = () => [
  {
    split: {
      mode: 'one',
      branches: [
        {
          condition: { field: 'days', op: 'gt', value: '3' },
          body: [{ split: { mode: 'all', branches: [{ body: [step({ kind: 'user', user_ids: [ids.a] })] }, { body: [step({ kind: 'user', user_ids: [ids.b] })] }] } }],
        },
      ],
      otherwise: [step({ kind: 'module' })],
    },
  },
]

// The other tests start from that flow, so each runs on its own.
async function putBuilt() {
  const rule = await api.put('/api/approval-rules/hrm.leave_request', { data: { flow: built(), max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' } })
  expect(rule.status(), await rule.text()).toBe(204)
}

test('an administrator builds a flow with a choose-one and a parallel split, and it routes a leave request', async ({ browser, baseURL }) => {
  // The editor starts from no rule: a single manager step.
  await api.delete('/api/approval-rules/hrm.leave_request')
  const page = await as(browser, admin.login, url)
  const form = page.getByRole('main')
  await addTo(form, page, 'Rẽ nhánh chọn một')
  await removeStep(form.getByRole('region', { name: 'Bước duyệt' }).first(), page)

  const split = form.getByRole('region', { name: 'Rẽ nhánh chọn một' })
  const branch = split.getByRole('region', { name: 'Nhánh 1' })
  await branch.getByRole('button', { name: 'Thêm điều kiện' }).click()
  await pick(branch, page, 'Trường', 'Số ngày')
  await pick(branch, page, 'So sánh', 'Lớn hơn')
  await branch.getByRole('textbox', { name: /^Giá trị/ }).fill('3')

  await addTo(branch, page, 'Rẽ nhánh song song')
  await removeStep(branch.getByRole('region', { name: 'Bước duyệt' }).first(), page)
  const par = branch.getByRole('region', { name: 'Rẽ nhánh song song' })
  await par.getByRole('button', { name: 'Thêm nhánh' }).click()
  await approverIsUser(par.getByRole('region', { name: 'Nhánh 1' }), page, login.a)
  await approverIsUser(par.getByRole('region', { name: 'Nhánh 2' }), page, login.b)

  await addTo(split.getByRole('region', { name: 'Còn lại' }), page, 'Bước duyệt')
  await pick(form, page, 'Vai trò dự phòng', 'Nhân sự')
  await page.getByRole('button', { name: 'Lưu' }).click()
  await expect(page.getByText('Đã lưu quy tắc duyệt.')).toBeVisible()

  const saved = (await (await api.get('/api/approval-rules?product=hrm')).json()) as { doc_type: string; rule: { flow: unknown } | null }[]
  expect(saved.find((r) => r.doc_type === 'hrm.leave_request')?.rule?.flow).toEqual(built())

  await page.reload()
  await expect(branch.getByRole('textbox', { name: /^Giá trị/ })).toHaveValue('3')
  await expect(par.getByRole('region', { name: 'Bước duyệt' })).toHaveCount(2)
  await expect(par.getByText(`${login.a} (${login.a})`)).toBeVisible()
  await expect(par.getByText(`${login.b} (${login.b})`)).toBeVisible()
  await expect(split.getByRole('region', { name: 'Còn lại' }).getByRole('region', { name: 'Bước duyệt' })).toHaveCount(1)

  const asE = await signedInApi(baseURL!, login.e, password)
  const leave = await created(
    asE.post('/api/hrm/leaves', { data: { leave_type_id: ids.type, start_date: '2026-08-10', end_date: '2026-08-14', days: '5', reason: null } }),
  )
  const sent = await asE.post(`/api/documents/hrm.leave_request/${leave}/transitions`, { data: { to: 'posted', version: 1 } })
  expect(sent.status(), await sent.text()).toBe(200)

  for (const [user, after] of [
    [login.a, 'Chờ duyệt'],
    // Step badges also read "Đã duyệt"; the finished submission is unambiguous.
    [login.b, 'Lần duyệt này đã hoàn tất.'],
  ] as const) {
    const p = await as(browser, user, '/inbox')
    await p.getByRole('button', { name: /NP-2026-/ }).first().click()
    await expect(p.getByText('Song song', { exact: true })).toBeVisible()
    await p.getByRole('button', { name: 'Duyệt', exact: true }).click()
    await expect(p.getByText('Đã duyệt.')).toBeVisible()
    await page.goto(`/hrm/leaves/${leave}`)
    await status(page, after)
  }
})

test('a condition row without a value is refused before saving', async ({ browser }) => {
  await putBuilt()
  const page = await as(browser, admin.login, url)
  let puts = 0
  page.on('request', (r) => {
    if (r.method() === 'PUT' && r.url().includes('/api/approval-rules/')) puts++
  })
  const branch = page.getByRole('region', { name: 'Rẽ nhánh chọn một' }).getByRole('region', { name: 'Nhánh 1' })
  await branch.getByRole('button', { name: 'Thêm điều kiện' }).first().click()
  // The new row is the last one; its value stays empty.
  await branch.getByRole('combobox', { name: 'Trường' }).last().click()
  await page.getByRole('option', { name: 'Số ngày', exact: true }).click()
  await branch.getByRole('combobox', { name: 'So sánh' }).last().click()
  await page.getByRole('option', { name: 'Lớn hơn', exact: true }).click()
  await page.getByRole('button', { name: 'Lưu' }).click()
  await expect(branch.getByText('Chưa nhập.')).toHaveCount(1)
  await expect(branch.getByRole('textbox', { name: /^Giá trị/ }).last()).toHaveAttribute('aria-invalid', 'true')
  expect(puts).toBe(0)
})

test('removing a nested group down to no rows drops the group itself', async ({ browser }) => {
  await putBuilt()
  const page = await as(browser, admin.login, url)
  const branch = page.getByRole('region', { name: 'Rẽ nhánh chọn một' }).getByRole('region', { name: 'Nhánh 1' })
  await branch.getByRole('button', { name: 'Thêm nhóm điều kiện' }).first().click()
  // The nested group's own, only row; removing it must drop the group, not leave it empty (the
  // server refuses an empty "all"/"any"). This restores the form to its original values, so the
  // form goes back to clean and offers no Save; the drop itself is the thing under test.
  await branch.getByRole('button', { name: 'Bỏ điều kiện' }).last().click()
  await expect(branch.getByRole('button', { name: 'Bỏ nhóm điều kiện' })).toHaveCount(0)
})

test('a rejected flow keeps its error on the card and the form unsaved', async ({ browser }) => {
  await putBuilt()
  const page = await as(browser, admin.login, url)
  await page.route(/\/api\/approval-rules\/hrm\.leave_request(\?|$)/, (route) =>
    route.request().method() === 'PUT'
      ? route.fulfill({ status: 422, contentType: 'application/json', body: JSON.stringify({ code: 'invalid_rule', params: { path: '1', reason: 'unknown_role' } }) })
      : route.continue(),
  )
  await page.getByRole('textbox', { name: /Số cấp quản lý tối đa/ }).fill('4')
  await page.getByRole('button', { name: 'Lưu' }).click()
  const split = page.getByRole('region', { name: 'Rẽ nhánh chọn một' })
  await expect(split.getByRole('alert').first()).toHaveText('Vai trò này không còn tồn tại. Chọn vai trò khác.')
  // Still unsaved: the edit is kept and Discard is offered.
  await expect(page.getByRole('textbox', { name: /Số cấp quản lý tối đa/ })).toHaveValue('4')
  await expect(page.getByRole('button', { name: 'Bỏ thay đổi' })).toBeVisible()
})

test('a disabled product shows the flow read-only', async ({ browser }) => {
  await putBuilt()
  // Cookies ignore the port, so the server without products gets its own browser context.
  const page = await (await browser.newContext()).newPage()
  await signedIn(page, `http://localhost:8091${url}`)
  await expect(page.getByRole('region', { name: 'Rẽ nhánh song song' })).toBeVisible()
  await expect(page.getByRole('main').getByRole('button', { name: /^Thêm/ })).toHaveCount(0)
  await expect(page.getByRole('main').getByRole('button', { name: 'Thao tác khác' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Lưu' })).toHaveCount(0)
})
