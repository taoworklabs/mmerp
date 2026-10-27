import { expect, test, type Page } from '@playwright/test'
import { created, createUser, runId, signedInApi } from './api'
import { signedIn } from './helpers'

// Dialogs and menus used from the keyboard, and popups opened inside a dialog. Two fresh users, one
// delegating to the other, so the list has a row of its own and nothing else is touched.
const run = runId()
const login = { from: `kov_${run}a`, to: `kov_${run}b`, to2: `kov_${run}c` }

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const from = await createUser(api, login.from, [])
  const to = await createUser(api, login.to, [])
  await createUser(api, login.to2, [])
  await created(api.post('/api/approval-delegations', { data: { user_id: from, delegate_id: to, starts_on: '2030-01-01', ends_on: '2030-01-31' } }))
})

async function openCreate(page: Page) {
  await signedIn(page, '/delegations')
  const opener = page.getByRole('button', { name: 'Tạo ủy quyền' }).first()
  await opener.focus()
  await page.keyboard.press('Enter')
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  return { opener, dialog }
}

test('dialog traps focus and returns it', async ({ page }) => {
  const { opener, dialog } = await openCreate(page)
  await dialog.getByRole('button', { name: 'Tạo ủy quyền' }).focus()
  await page.keyboard.press('Tab')
  await expect(dialog.getByRole('button', { name: 'Đóng hộp thoại' })).toBeFocused()
  await page.keyboard.press('Escape')
  await expect(dialog).toBeHidden()
  await expect(opener).toBeFocused()
})

test('row menu works by keyboard', async ({ page }) => {
  await signedIn(page, '/delegations')
  const more = page.getByRole('row', { name: new RegExp(`${login.from}.*${login.to}`) }).getByRole('button', { name: 'Thao tác khác' })
  await more.focus()
  await page.keyboard.press('Enter')
  const menu = page.getByRole('menu')
  await expect(menu).toBeVisible()
  await page.keyboard.press('ArrowDown')
  await expect(menu.getByRole('menuitem').first()).toBeFocused()
  await page.keyboard.press('Escape')
  await expect(menu).toBeHidden()
  await expect(more).toBeFocused()
})

test('controls are 40px in a modal and 30px in a filter', async ({ page }) => {
  const { dialog } = await openCreate(page)
  // The control's box is the input's frame; the input itself sits inside its border.
  const height = (el: HTMLElement) => el.parentElement!.offsetHeight
  expect(await dialog.getByRole('textbox', { name: /^Đến ngày/ }).evaluate(height)).toBe(40)
  await page.keyboard.press('Escape')
  await page.goto('/hrm/employees')
  expect(await page.getByRole('searchbox').first().evaluate(height)).toBe(30)
})

test('select and date field work inside a dialog', async ({ page }) => {
  const { dialog } = await openCreate(page)
  await dialog.getByRole('combobox', { name: 'Người ủy quyền' }).click()
  await page.getByRole('option', { name: `${login.from} (${login.from})`, exact: true }).click()
  await dialog.getByRole('combobox', { name: /^Người được ủy quyền/ }).click()
  await page.getByRole('option', { name: `${login.to2} (${login.to2})`, exact: true }).click()
  // Typed first (leaving the field keeps it) so the picker opens on a known month, then a day is clicked in it.
  const from = dialog.getByRole('textbox', { name: /^Từ ngày/ })
  await from.fill('01/06/2030')
  await from.press('Tab')
  const to = dialog.getByRole('textbox', { name: /^Đến ngày/ })
  await to.fill('10/06/2030')
  await to.press('Tab')
  await to.click()
  await page.getByRole('gridcell', { name: '15', exact: true }).click()
  await expect(to).toHaveValue('15/06/2030')
  await dialog.getByRole('button', { name: 'Tạo ủy quyền' }).click()
  await expect(page.getByText('Đã tạo ủy quyền.')).toBeVisible()
  await expect(page.getByRole('row', { name: new RegExp(`${login.from}.*${login.to2}.*Sắp tới`) })).toBeVisible()
})
