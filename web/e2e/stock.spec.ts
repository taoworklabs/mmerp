import { expect, test, type Page } from '@playwright/test'
import { created, createUser, ok, runId, signedInApi } from './api'
import { as, status } from './helpers'

// One run: a company with its main warehouse and a second one, a quantity item with 10 on hand
// and a serial item with two serials, entered by an adjustment. One account keeps the stock.
const run = runId()
const login = `stk_${run}`
const qty = `QA${run}`
const ser = `SR${run}`
const s1 = `S1${run}`
let main: string
let second: string
let secondId: number

const iso = new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Ho_Chi_Minh' })
const typed = (d: string) => d.split('-').reverse().join('/')

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(
    api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty KHO ${run}`, legal_name: `Công ty TNHH KHO ${run}` } }),
  )
  await createUser(api, login, [{ role: 'stock.admin', unit: null }])
  const { items } = (await (await api.get('/api/stock/warehouses?page_size=100')).json()) as {
    items: { id: number; code: string; name: string; legal_entity_id: number; is_default: boolean }[]
  }
  const w = items.find((x) => x.legal_entity_id === company && x.is_default)!
  main = `${w.code} · ${w.name}`
  secondId = await created(api.post('/api/stock/warehouses', { data: { name: `Kho phụ ${run}`, legal_entity_id: company, active: true } }))
  const w2 = (await (await api.get(`/api/stock/warehouses?page_size=100&q=${encodeURIComponent(`Kho phụ ${run}`)}`)).json()) as { items: { code: string }[] }
  second = `${w2.items[0]!.code} · Kho phụ ${run}`
  const q = await created(
    api.post('/api/items', { data: { code: qty, name: `Nước suối ${run}`, unit: 'thùng', price: 90_000, vat_rate: '8', active: true, tracking: 'quantity' } }),
  )
  const s = await created(
    api.post('/api/items', { data: { code: ser, name: `Máy in ${run}`, unit: 'cái', price: 4_000_000, vat_rate: '10', active: true, tracking: 'serial' } }),
  )
  const adj = await created(
    api.post('/api/stock/adjustments', {
      data: {
        request_id: crypto.randomUUID(),
        date: iso,
        warehouse_id: w.id,
        reason: 'Tồn đầu kỳ',
        lines: [
          { item_id: q, quantity: '10', unit_cost: 50_000 },
          { item_id: s, serial: s1, quantity: '1', unit_cost: 3_000_000 },
          { item_id: s, serial: `S2${run}`, quantity: '1', unit_cost: 3_000_000 },
        ],
      },
    }),
  )
  const { version } = (await (await api.get(`/api/stock/adjustments/${adj}`)).json()) as { version: number }
  await ok(api.post(`/api/documents/stock.adjustment/${adj}/transitions`, { data: { to: 'posted', version } }))
})

// onHand checks an item's quantity in one warehouse on the on-hand screen.
async function onHand(page: Page, warehouseId: number | null, code: string, quantity: string) {
  await page.goto(`/stock/on-hand?q=${code}${warehouseId ? `&warehouse=${warehouseId}` : ''}`)
  const table = page.getByRole('table', { name: 'Tồn kho' })
  await expect(table.getByRole('row').filter({ hasText: code }).first().getByRole('cell', { name: quantity, exact: true })).toBeVisible()
}

test('a transfer moves goods between warehouses, a stocktake posts the counted difference, and the transfer then cannot be cancelled', async ({ browser }) => {
  const page = await as(browser, login, '/stock/transfers')
  await page.getByRole('link', { name: 'Lập phiếu chuyển kho' }).click()
  await page.getByRole('textbox', { name: 'Ngày chuyển' }).fill(typed(iso))
  await page.getByRole('combobox', { name: 'Kho xuất' }).click()
  await page.getByRole('option', { name: main }).click()
  await page.getByRole('combobox', { name: 'Kho nhập' }).click()
  await page.getByRole('option', { name: second }).click()
  await page.getByRole('combobox', { name: 'Mặt hàng (1)' }).fill(qty)
  await page.getByRole('option', { name: new RegExp(qty) }).click()
  await page.getByRole('textbox', { name: 'Số lượng (1)' }).fill('4')
  await page.getByRole('button', { name: 'Thêm dòng hàng' }).click()
  await page.getByRole('combobox', { name: 'Mặt hàng (2)' }).fill(ser)
  await page.getByRole('option', { name: new RegExp(ser) }).click()
  await page.getByRole('textbox', { name: 'Số serial (2)' }).fill(s1)
  await page.getByRole('textbox', { name: 'Số lượng (2)' }).fill('1')
  await page.getByRole('button', { name: 'Tạo bản nháp' }).click()
  await expect(page.getByRole('heading', { name: /^CK-\d{4}-/ })).toBeVisible()
  const transfer = page.url()
  await page.getByRole('button', { name: 'Gửi duyệt' }).click()
  await status(page, 'Đã ghi sổ')

  await onHand(page, secondId, qty, '4')
  await onHand(page, secondId, ser, '1')

  // Count the second warehouse: the water is gone, the printer is there.
  await page.goto('/stock/stocktakes/new')
  await page.getByRole('textbox', { name: 'Ngày kiểm kê' }).fill(typed(iso))
  await page.getByRole('combobox', { name: 'Kho', exact: true }).click()
  await page.getByRole('option', { name: second }).click()
  await page.getByRole('button', { name: 'Lấy tồn kho' }).click()
  await expect(page.getByRole('textbox', { name: 'Số lượng thực tế (1)' })).toHaveValue('4')
  await page.getByRole('textbox', { name: 'Số lượng thực tế (1)' }).fill('0')
  await page.getByRole('button', { name: 'Tạo bản nháp' }).click()
  await expect(page.getByRole('heading', { name: /^KK-\d{4}-/ })).toBeVisible()
  await page.getByRole('button', { name: 'Gửi duyệt' }).click()
  await status(page, 'Đã ghi sổ')
  const lines = page.getByRole('table', { name: 'Dòng hàng' })
  await expect(lines.getByRole('row').filter({ hasText: qty }).getByRole('cell', { name: '-4', exact: true })).toBeVisible()

  await onHand(page, null, qty, '6')

  // The water the transfer brought has left the second warehouse, so the transfer stands.
  await page.goto(transfer)
  await page.getByRole('button', { name: 'Hủy chứng từ' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Hủy chứng từ' }).click()
  await expect(page.getByText(/không đủ hàng trong kho/)).toBeVisible()
  await page.reload()
  await status(page, 'Đã ghi sổ')
})
