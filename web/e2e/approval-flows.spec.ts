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

const main = (page: Page) => page.getByRole('main')
const panel = (page: Page) => page.getByRole('complementary')
// A box, chip or pill of the diagram, by its accessible name.
const box = (scope: Page | Locator, name: string | RegExp) => scope.getByRole('button', { name })
// A branch column of a split, by its chip's name ("Trường hợp 1: …", "Nhánh 2: …", "Còn lại").
const branchOf = (page: Page, name: string | RegExp) => main(page).getByRole('group', { name })

// insert opens the nth ⊕ inside scope and picks what to add there.
async function insert(scope: Page | Locator, page: Page, nth: number, element: 'Bước duyệt' | 'Tùy trường hợp' | 'Cùng lúc') {
  await scope.getByRole('button', { name: 'Thêm vào đây' }).nth(nth).click()
  await page.getByRole('menuitem', { name: new RegExp(`^${element}`) }).click()
}

async function pick(scope: Locator, page: Page, field: string, option: string) {
  await scope.getByRole('combobox', { name: field }).click()
  await page.getByRole('option', { name: option, exact: true }).click()
}

// approverIsUser opens a manager step and gives it one user instead.
async function approverIsUser(step: Locator, page: Page, user: string) {
  await step.click()
  await pick(panel(page), page, 'Ai duyệt', 'Những người cụ thể')
  await panel(page).getByRole('combobox', { name: 'Người duyệt' }).fill(user)
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
  // No rule: the diagram starts empty.
  await api.delete('/api/approval-rules/hrm.leave_request')
  const page = await as(browser, admin.login, url)
  await expect(main(page).getByText('Chưa có ai duyệt. Bấm dấu + để thêm người duyệt đầu tiên.')).toBeVisible()
  await insert(main(page), page, 0, 'Tùy trường hợp')

  await box(main(page), /^Trường hợp 1/).click()
  await panel(page).getByRole('button', { name: 'Thêm điều kiện' }).click()
  await pick(panel(page), page, 'Trường', 'Số ngày')
  await pick(panel(page), page, 'So sánh', 'Lớn hơn')
  await panel(page).getByRole('textbox', { name: /^Giá trị/ }).fill('3')
  await expect(box(main(page), 'Trường hợp 1: Nếu Số ngày > 3')).toBeVisible()

  // A parallel split before the branch's own step, which then goes.
  const first = branchOf(page, /^Trường hợp 1/)
  await insert(first, page, 0, 'Cùng lúc')
  await panel(page).getByRole('button', { name: 'Thêm nhánh' }).click()
  await box(first, 'Bước duyệt: Quản lý trực tiếp').last().click()
  await panel(page).getByRole('button', { name: 'Xóa bước' }).click()
  await approverIsUser(box(first, 'Bước duyệt: Quản lý trực tiếp').first(), page, login.a)
  await approverIsUser(box(first, 'Bước duyệt: Quản lý trực tiếp').first(), page, login.b)

  await insert(branchOf(page, 'Còn lại'), page, 0, 'Bước duyệt')
  await page.getByRole('button', { name: 'Cài đặt chung' }).click()
  await pick(panel(page), page, 'Không tìm được người duyệt thì chuyển cho', 'Nhân sự')
  await page.getByRole('button', { name: 'Lưu' }).click()
  await expect(page.getByText('Đã lưu quy tắc duyệt.')).toBeVisible()

  const saved = (await (await api.get('/api/approval-rules?product=hrm')).json()) as { doc_type: string; rule: { flow: unknown } | null }[]
  expect(saved.find((r) => r.doc_type === 'hrm.leave_request')?.rule?.flow).toEqual(built())

  await page.reload()
  await expect(box(main(page), 'Trường hợp 1: Nếu Số ngày > 3')).toBeVisible()
  await expect(box(main(page), 'Cùng lúc')).toBeVisible()
  await expect(box(main(page), `Bước duyệt: ${login.a}`)).toBeVisible()
  await expect(box(main(page), `Bước duyệt: ${login.b}`)).toBeVisible()
  await expect(box(branchOf(page, 'Còn lại'), 'Bước duyệt: Quản lý trực tiếp')).toHaveCount(1)

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

test('an unfinished condition is refused before saving and Save leads to it', async ({ browser }) => {
  await putBuilt()
  const page = await as(browser, admin.login, url)
  let puts = 0
  page.on('request', (r) => {
    if (r.method() === 'PUT' && r.url().includes('/api/approval-rules/')) puts++
  })
  const chip = box(main(page), /^Trường hợp 1/)
  await chip.click()
  await panel(page).getByRole('button', { name: 'Thêm điều kiện' }).first().click()
  // The new row is the last one; its value stays empty.
  await panel(page).getByRole('combobox', { name: 'Trường' }).last().click()
  await page.getByRole('option', { name: 'Số ngày', exact: true }).click()
  await panel(page).getByRole('combobox', { name: 'So sánh' }).last().click()
  await page.getByRole('option', { name: 'Lớn hơn', exact: true }).click()
  await expect(chip).toHaveAttribute('aria-invalid', 'true')
  await box(main(page), 'Cùng lúc').click()
  await page.getByRole('button', { name: 'Lưu' }).click()
  await expect(page.getByRole('alert').filter({ hasText: 'Còn 1 trường cần sửa' })).toBeVisible()
  await page.getByRole('button', { name: 'Đến trường tiếp theo' }).click()
  await expect(chip).toHaveAttribute('aria-pressed', 'true')
  await expect(panel(page).getByRole('alert')).toHaveText('Điều kiện không hợp lệ. Kiểm tra trường, phép so sánh và giá trị.')
  expect(puts).toBe(0)
})

test('removing a nested group down to no rows drops the group itself', async ({ browser }) => {
  await putBuilt()
  const page = await as(browser, admin.login, url)
  await box(main(page), /^Trường hợp 1/).click()
  await panel(page).getByRole('button', { name: 'Thêm nhóm điều kiện' }).first().click()
  // The nested group's own, only row; removing it must drop the group, not leave it empty (the
  // server refuses an empty "all"/"any").
  await panel(page).getByRole('button', { name: 'Bỏ điều kiện' }).last().click()
  await expect(panel(page).getByRole('button', { name: 'Bỏ nhóm điều kiện' })).toHaveCount(0)
})

test('a refused save marks the element it names, until the next edit', async ({ browser }) => {
  await putBuilt()
  const page = await as(browser, admin.login, url)
  await page.route(/\/api\/approval-rules\/hrm\.leave_request(\?|$)/, (route) =>
    route.request().method() === 'PUT'
      ? route.fulfill({ status: 422, contentType: 'application/json', body: JSON.stringify({ code: 'invalid_rule', params: { path: '1.otherwise.1', reason: 'unknown_role' } }) })
      : route.continue(),
  )
  await page.getByRole('button', { name: 'Cài đặt chung' }).click()
  await panel(page).getByRole('textbox', { name: /^Khi quản lý trực tiếp lại chính là người gửi/ }).fill('4')
  await page.getByRole('button', { name: 'Lưu' }).click()
  const step = box(branchOf(page, 'Còn lại'), 'Bước duyệt: Quản lý trực tiếp')
  await expect(step).toHaveAttribute('aria-pressed', 'true')
  await expect(step).toHaveAttribute('aria-invalid', 'true')
  await expect(panel(page).getByRole('alert')).toHaveText('Vai trò này không còn tồn tại. Chọn vai trò khác.')
  // Still unsaved: the edit is kept and Discard is offered.
  await expect(page.getByRole('button', { name: 'Bỏ thay đổi' })).toBeVisible()
  await panel(page).getByRole('textbox', { name: /^Hạn duyệt \(giờ\)/ }).fill('8')
  await expect(step).not.toHaveAttribute('aria-invalid', 'true')
})

test('moving a step keeps it selected, deleting it clears the panel', async ({ browser }) => {
  await putBuilt()
  const page = await as(browser, admin.login, url)
  // A role step after the split: the last ⊕ of the diagram.
  await insert(main(page), page, -1, 'Bước duyệt')
  await pick(panel(page), page, 'Ai duyệt', 'Người có vai trò, tại đơn vị của chứng từ')
  await pick(panel(page), page, 'Vai trò', 'Nhân sự')
  const hr = box(main(page), 'Bước duyệt: Nhân sự')
  await panel(page).getByRole('button', { name: 'Chuyển lên' }).click()
  await expect(hr).toHaveAttribute('aria-pressed', 'true')
  await expect(panel(page).getByRole('combobox', { name: 'Vai trò' })).toHaveValue('Nhân sự')
  // It now comes first, before the split.
  await expect(main(page).getByRole('button', { name: /^(Bước duyệt|Tùy trường hợp)/ }).first()).toHaveAccessibleName('Bước duyệt: Nhân sự')
  await panel(page).getByRole('button', { name: 'Xóa bước' }).click()
  await expect(hr).toHaveCount(0)
  await expect(panel(page)).toContainText('Bấm vào một ô trên sơ đồ để sửa.')
})

test('inserting a step makes the form unsaved', async ({ browser }) => {
  await putBuilt()
  const page = await as(browser, admin.login, url)
  await insert(main(page), page, 0, 'Bước duyệt')
  await expect(page.getByText('Có thay đổi chưa lưu')).toBeVisible()
  await page.getByRole('navigation').getByRole('link', { name: 'Tổng quan' }).click()
  await expect(page.getByRole('dialog', { name: 'Rời trang?' })).toBeVisible()
})

test('a disabled product shows the flow read-only', async ({ browser }) => {
  await putBuilt()
  // Cookies ignore the port, so the server without products gets its own browser context.
  const page = await (await browser.newContext()).newPage()
  await signedIn(page, `http://localhost:8091${url}`)
  await expect(box(main(page), 'Cùng lúc')).toBeVisible()
  await expect(main(page).getByRole('button', { name: 'Thêm vào đây' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Lưu' })).toHaveCount(0)
  await box(branchOf(page, 'Còn lại'), 'Bước duyệt: Quản lý trực tiếp').click()
  await expect(panel(page).getByRole('combobox', { name: 'Ai duyệt' })).toHaveAttribute('readonly', '')
  await expect(panel(page).getByRole('button', { name: 'Xóa bước' })).toHaveCount(0)
})
