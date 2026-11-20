import { expect, test, type APIRequestContext } from '@playwright/test'
import { created, ok, runId, signedInApi } from './api'
import { signedIn } from './helpers'

// One run: a company of its own sells 1 000 000 đ cash in May and spends 300 000 đ. Root, the
// accountant, closes May from the screen, then reads the income statement and the balance sheet.
// The file sets the closing accounts and statement forms for the whole tenant, so it runs alone.
const run = runId()
let api: APIRequestContext
let c: number
const acct: Record<string, number> = {}

test.beforeAll(async ({ baseURL }) => {
  test.setTimeout(120_000)
  api = await signedInApi(baseURL!)
  const me = ((await (await api.get('/api/users')).json()) as { id: number; login: string }[]).find((u) => u.login === 'root')
  if (me) await api.post(`/api/users/${me.id}/roles`, { data: { product: 'accounting', role: 'admin', org_unit_id: null } })
  c = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty KC ${run}` } }))

  // Other files may have built part of the chart already; reuse what is there.
  const existing = ((await (await api.get('/api/accounting/accounts')).json()) as { items: { id: number; code: string }[] }).items
  for (const [code, name, type] of [
    ['111', 'Tiền mặt', 'asset'],
    ['511', 'Doanh thu bán hàng và cung cấp dịch vụ', 'revenue'],
    ['642', 'Chi phí quản lý doanh nghiệp', 'expense'],
    ['911', 'Xác định kết quả kinh doanh', 'equity'],
    ['4212', 'Lợi nhuận sau thuế chưa phân phối năm nay', 'equity'],
  ] as const) {
    acct[code] =
      existing.find((a) => a.code === code)?.id ?? (await created(api.post('/api/accounting/accounts', { data: { code, name, type, parent_id: null, active: true } })))
  }
  await ok(api.put('/api/accounting/closing-accounts', { data: { result_account_id: acct['911'], retained_account_id: acct['4212'] } }))
  await ok(
    api.put('/api/accounting/statement-forms/income_statement', {
      data: {
        rows: [
          { code: '01', label: 'Doanh thu bán hàng và cung cấp dịch vụ', formula: 'CT(511)', bold: false },
          { code: '24', label: 'Chi phí quản lý kinh doanh', formula: 'CK(642) - DK(642)', bold: false },
          { code: '60', label: 'Lợi nhuận sau thuế thu nhập doanh nghiệp', formula: 'DK(511) - CK(642)', bold: true },
        ],
      },
    }),
  )
  await ok(
    api.put('/api/accounting/statement-forms/balance_sheet', {
      data: {
        rows: [
          { code: '110', label: 'Tiền và các khoản tương đương tiền', formula: 'DB(111)', bold: false },
          { code: '417', label: 'Lợi nhuận sau thuế chưa phân phối', formula: '-B(421)', bold: false },
        ],
      },
    }),
  )
  await ok(api.put(`/api/accounting/starts/${c}`, { data: { start_date: '2026-01-01' } }))
  for (const [debit, credit, amount] of [
    ['111', '511', 1_000_000],
    ['642', '111', 300_000],
  ] as const) {
    const entry = await created(
      api.post('/api/accounting/journal-entries', {
        data: {
          request_id: crypto.randomUUID(),
          date: '2026-05-10',
          legal_entity_id: c,
          description: 'Phát sinh tháng 5',
          lines: [
            { account_id: acct[debit], debit: amount, credit: 0 },
            { account_id: acct[credit], debit: 0, credit: amount },
          ],
        },
      }),
    )
    await ok(api.post(`/api/documents/accounting.journal_entry/${entry}/transitions`, { data: { to: 'posted', version: 1 } }))
  }
})

test('a closing entry moves May to retained earnings; the statements read it', async ({ page }) => {
  // Before closing, the income statement says the period is not closed.
  const query = `legal_entity_id=${c}&from=2026-05-01&to=2026-05-31`
  await signedIn(page, `/accounting/income-statement?${query}`)
  await expect(page.getByText('Còn số dư doanh thu, chi phí chưa kết chuyển')).toBeVisible()

  await page.goto('/accounting/closings/new')
  await page.getByRole('textbox', { name: /^Ngày kết chuyển/ }).fill('31/05/2026')
  await page.getByRole('combobox', { name: /^Pháp nhân/ }).click()
  await page.getByRole('option', { name: `Công ty KC ${run}` }).click()
  await page.getByRole('textbox', { name: /^Diễn giải/ }).fill('Kết chuyển tháng 5')
  await page.getByRole('button', { name: 'Tạo nháp' }).click()
  await expect(page.getByRole('heading', { name: /^KC-/ })).toBeVisible()
  // The draft previews the entry: both accounts cleared through 911, the result to 4212.
  await expect(page.getByText('911 · Xác định kết quả kinh doanh').first()).toBeVisible()
  await page.getByRole('button', { name: 'Gửi duyệt' }).click()
  await expect(page.getByText('Đã ghi sổ').first()).toBeVisible()

  await page.goto(`/accounting/income-statement?${query}`)
  await expect(page.getByText('Còn số dư doanh thu, chi phí chưa kết chuyển')).toBeHidden()
  const profit = page.getByRole('row').filter({ hasText: 'Lợi nhuận sau thuế thu nhập doanh nghiệp' })
  await expect(profit).toContainText('700.000')

  await page.goto(`/accounting/balance-sheet?legal_entity_id=${c}&to=2026-05-31`)
  await expect(page.getByRole('row').filter({ hasText: 'Tiền và các khoản tương đương tiền' })).toContainText('700.000')
  await expect(page.getByRole('row').filter({ hasText: 'Lợi nhuận sau thuế chưa phân phối' })).toContainText('700.000')
})
