import { expect, test, type APIRequestContext, type APIResponse } from '@playwright/test'
import { created, signedInApi } from './api'

// Sales on top of the Demo demo company (demo.seed.ts runs first): sales roles for the two
// sales departments, the item catalogue, customers, quotations in every status and orders,
// from quotations and entered directly. Its own accounts (password Demo@2026) do the work.
// Skipped once quotations exist; catalogue entries and customers already there are reused.

const password = 'Demo@2026'
type Kind = 'quotes' | 'orders'
type Outcome = 'draft' | 'pending' | 'approved' | 'rejected' | 'cancelled'

async function ok(res: Promise<APIResponse>, what: string) {
  const r = await res
  expect(r.ok(), `${what}: ${r.status()} ${await r.text()}`).toBe(true)
  return r
}
const get = async <T>(as: APIRequestContext, path: string): Promise<T> => (await (await ok(as.get(path), path)).json()) as T

// [code, name, unit, price, VAT rate, active]
const items: [string, string, string, number, string, boolean][] = [
  ['VPP-A4', 'Giấy A4 80gsm', 'ram', 78_000, '10', true],
  ['BUT-TL', 'Bút bi (hộp 20 cây)', 'hộp', 85_000, '10', true],
  ['LAP-14', 'Máy tính xách tay 14 inch', 'cái', 24_500_000, '10', true],
  ['MH-27', 'Màn hình 27 inch', 'cái', 5_200_000, '10', true],
  ['MIN-LZ', 'Máy in laser đơn sắc', 'cái', 3_900_000, '10', true],
  ['MUC-LZ', 'Hộp mực máy in laser', 'hộp', 1_250_000, '10', true],
  ['GHE-VP', 'Ghế văn phòng lưới', 'cái', 1_650_000, '10', true],
  ['BAN-LV', 'Bàn làm việc 1m2', 'cái', 2_300_000, '10', true],
  ['NUOC-20L', 'Nước uống đóng bình 20L', 'bình', 65_000, '5', true],
  ['GAO-5KG', 'Gạo thơm túi 5kg', 'túi', 210_000, '5', true],
  ['DV-LD', 'Dịch vụ lắp đặt tại chỗ', 'lần', 500_000, '8', true],
  ['DV-BT', 'Bảo trì thiết bị văn phòng', 'tháng', 1_500_000, '8', true],
  ['DV-VC', 'Vận chuyển nội thành', 'chuyến', 350_000, '8', true],
  ['DV-DT', 'Đào tạo sử dụng phần mềm', 'buổi', 3_000_000, 'none', true],
  ['PM-KT', 'Bản quyền phần mềm kế toán (năm)', 'năm', 6_000_000, 'none', true],
  ['FAX-01', 'Máy fax', 'cái', 2_800_000, '10', false],
]

type Team = 'hn' | 'hcm'
// [code, name, team, tax code, address, phone, contact, payment terms, active]
const customers: [string, string, Team, string | null, string, string, string | null, string, boolean][] = [
  ['KH001', 'Công ty TNHH Thương mại Hoàng Long', 'hn', '0101111111', '12 Láng Hạ, Đống Đa, Hà Nội', '024 3771 1111', 'Anh Hoàng Văn Long', 'Thanh toán trong 30 ngày', true],
  ['KH002', 'Trường THCS Nguyễn Du', 'hn', '0102222222', '45 Nguyễn Du, Hai Bà Trưng, Hà Nội', '024 3822 2222', 'Cô Phạm Thị Hạnh', 'Chuyển khoản sau nghiệm thu', true],
  ['KH003', 'Công ty CP Xây dựng An Phát', 'hn', '0103333333', '8 Phạm Hùng, Cầu Giấy, Hà Nội', '024 3793 3333', 'Chị Nguyễn Thu Hằng', 'Tạm ứng 30%, còn lại trong 15 ngày', true],
  ['KH004', 'Phòng khám Đa khoa Hà Đông', 'hn', '0104444444', '102 Quang Trung, Hà Đông, Hà Nội', '024 3355 4444', 'Bác sĩ Trần Minh', 'Thanh toán khi nhận hàng', true],
  ['KH005', 'Anh Nguyễn Văn Bình', 'hn', null, '27 Ngõ Huế, Hai Bà Trưng, Hà Nội', '0912 555 555', null, 'Thanh toán khi nhận hàng', true],
  ['KH006', 'Công ty TNHH Nhà hàng Sài Gòn Xanh', 'hcm', '0306666666', '88 Lê Thánh Tôn, Quận 1, TP.HCM', '028 3822 6666', 'Anh Lê Quốc Huy', 'Thanh toán trong 15 ngày', true],
  ['KH007', 'Công ty CP Dược phẩm Phương Nam', 'hcm', '0307777777', '210 Cộng Hòa, Tân Bình, TP.HCM', '028 3811 7777', 'Chị Võ Ngọc Trâm', 'Thanh toán trong 45 ngày', true],
  ['KH008', 'Khách sạn Bến Thành', 'hcm', '0308888888', '5 Phạm Ngũ Lão, Quận 1, TP.HCM', '028 3836 8888', 'Anh Đặng Minh Khôi', 'Tạm ứng 50%', true],
  ['KH009', 'Chị Trần Thị Lan', 'hcm', null, '14 Nguyễn Trãi, Quận 5, TP.HCM', '0903 999 999', null, 'Thanh toán khi nhận hàng', true],
  ['KH010', 'Công ty TNHH Logistics Đông Á', 'hcm', '0310101010', '3 Nguyễn Tất Thành, Quận 4, TP.HCM', '028 3940 1010', 'Anh Phan Đông', 'Thanh toán trong 30 ngày', false],
]

// A line: item code, quantity, discount %, and a unit price when it differs from the catalogue.
type Line = [string, string, string, number?]
type Quote = { by: string; customer: string; date: string; until: string; lines: Line[]; outcome: Outcome; order?: Outcome; why?: string }

const quotes: Quote[] = [
  { by: 'kd.hn', customer: 'KH001', date: '2026-09-03', until: '2026-10-31', lines: [['LAP-14', '5', '0'], ['MH-27', '5', '5'], ['DV-LD', '5', '0']], outcome: 'approved', order: 'approved' },
  { by: 'kd.hn', customer: 'KH002', date: '2026-09-10', until: '2026-10-31', lines: [['GHE-VP', '40', '12'], ['BAN-LV', '20', '12'], ['DV-VC', '3', '0']], outcome: 'approved', order: 'draft' },
  { by: 'kd.hn2', customer: 'KH003', date: '2026-09-18', until: '2026-11-15', lines: [['VPP-A4', '200', '0'], ['BUT-TL', '30', '0'], ['MUC-LZ', '10', '5']], outcome: 'approved' },
  { by: 'kd.hn2', customer: 'KH005', date: '2026-10-02', until: '2026-10-31', lines: [['LAP-14', '1', '15']], outcome: 'pending' },
  { by: 'kd.hn', customer: 'KH001', date: '2026-10-08', until: '2026-11-30', lines: [['MIN-LZ', '3', '0'], ['MUC-LZ', '6', '0']], outcome: 'draft' },
  { by: 'kd.hn', customer: 'KH003', date: '2026-08-20', until: '2026-09-15', lines: [['DV-BT', '12', '5']], outcome: 'approved' },
  { by: 'kd.hn2', customer: 'KH004', date: '2026-09-25', until: '2026-10-25', lines: [['MH-27', '10', '20']], outcome: 'rejected', why: 'Chiết khấu 20% vượt mức cho phép, đề xuất tối đa 10%' },
  { by: 'kd.hcm', customer: 'KH006', date: '2026-09-05', until: '2026-10-20', lines: [['NUOC-20L', '100', '0'], ['GAO-5KG', '50', '3'], ['DV-VC', '4', '0']], outcome: 'approved', order: 'approved' },
  { by: 'kd.hcm', customer: 'KH007', date: '2026-09-15', until: '2026-12-31', lines: [['PM-KT', '10', '10'], ['DV-DT', '4', '0']], outcome: 'cancelled' },
  { by: 'kd.hcm', customer: 'KH008', date: '2026-09-28', until: '2026-11-30', lines: [['LAP-14', '20', '8'], ['MH-27', '20', '8'], ['DV-LD', '20', '0']], outcome: 'approved', order: 'pending' },
  { by: 'kd.hcm', customer: 'KH009', date: '2026-10-06', until: '2026-10-31', lines: [['GHE-VP', '2', '0'], ['BAN-LV', '1', '0', 2_100_000]], outcome: 'draft' },
]

type Order = { by: string; customer: string; date: string; delivery?: string; lines: Line[]; outcome: Outcome }
const orders: Order[] = [
  { by: 'kd.hn', customer: 'KH002', date: '2026-09-29', delivery: '2026-10-05', lines: [['VPP-A4', '50', '0'], ['BUT-TL', '10', '0']], outcome: 'approved' },
  { by: 'kd.hcm', customer: 'KH006', date: '2026-10-01', delivery: '2026-10-03', lines: [['NUOC-20L', '60', '0']], outcome: 'approved' },
  { by: 'kd.hcm', customer: 'KH008', date: '2026-10-05', lines: [['DV-BT', '6', '0']], outcome: 'cancelled' },
  { by: 'kd.hn2', customer: 'KH005', date: '2026-10-09', delivery: '2026-10-15', lines: [['MUC-LZ', '2', '0']], outcome: 'draft' },
]

test('seed Sales on the Demo demo company', async ({ baseURL }) => {
  test.setTimeout(300_000)
  const admin = await signedInApi(baseURL!)
  if ((await get<{ total: number }>(admin, '/api/sales/quotes?page_size=20')).total > 0) {
    test.skip(true, 'Sales data already there')
  }

  // Roles: two staff in Hà Nội, one in TP.HCM, a manager per department, a sales director who
  // reads everything and approves large quotations.
  const units = await get<{ id: number; name: string }[]>(admin, '/api/org-units')
  const unit = (name: string) => units.find((u) => u.name === name)!.id
  const team: Record<Team, number> = { hn: unit('Kinh doanh (HN)'), hcm: unit('Kinh doanh (HCM)') }
  for (const [login, name] of [
    ['kd.hn', 'Trịnh Thu Trang'],
    ['kd.hn2', 'Tạ Quang An'],
    ['kd.hcm', 'Huỳnh Thị Kim'],
    ['ql.kd.hn', 'Ngô Văn Hùng'],
    ['ql.kd.hcm', 'Trương Văn Lợi'],
    ['gd.kinhdoanh', 'Nguyễn Văn Minh'],
  ]) {
    const r = await admin.post('/api/users', { data: { login, name, password } })
    expect([201, 409], await r.text()).toContain(r.status())
  }
  const users = await get<{ id: number; login: string }[]>(admin, '/api/users')
  const userId = (login: string) => users.find((u) => u.login === login)!.id
  const grant = async (login: string, role: string, org_unit_id: number | null) => {
    const r = await admin.post(`/api/users/${userId(login)}/roles`, { data: { product: 'sales', role, org_unit_id } })
    expect([201, 409], await r.text()).toContain(r.status())
  }
  await grant('kd.hn', 'staff', team.hn)
  await grant('kd.hn2', 'staff', team.hn)
  await grant('ql.kd.hn', 'manager', team.hn)
  await grant('kd.hcm', 'staff', team.hcm)
  await grant('ql.kd.hcm', 'manager', team.hcm)
  await grant('gd.kinhdoanh', 'viewer', null)

  // Approval: discounts above 10 % go to the department head, quotations above 500 million
  // also to the director; orders above 300 million to the head.
  const rule = (type: string, steps: unknown[]) =>
    ok(admin.put(`/api/approval-rules/${type}`, { data: { steps, max_levels: 3, fallback_product: 'sales', fallback_role: 'manager' } }), `rule ${type}`)
  const manager = { kind: 'role', product: 'sales', role: 'manager' }
  await rule('sales.quote', [
    { condition: { field: 'max_discount', op: 'gt', value: '10' }, approver: manager },
    { condition: { field: 'amount', op: 'gt', value: '500000000' }, approver: { kind: 'user', user_id: userId('gd.kinhdoanh') } },
  ])
  await rule('sales.order', [{ condition: { field: 'amount', op: 'gt', value: '300000000' }, approver: manager }])

  // Catalogue and customers.
  const itemId: Record<string, number> = {}
  const itemOf: Record<string, (typeof items)[number]> = {}
  for (const it of items) {
    const [code, name, unitName, price, vat_rate, active] = it
    const r = await admin.post('/api/sales/items', { data: { code, name, unit: unitName, price, vat_rate, active } })
    itemId[code] =
      r.status() === 409
        ? (await get<{ items: { id: number; code: string }[] }>(admin, `/api/sales/items?q=${code}&page_size=20`)).items.find((x) => x.code === code)!.id
        : await created(Promise.resolve(r))
    itemOf[code] = it
  }
  const customerId: Record<string, number> = {}
  const customerTerms: Record<string, string> = {}
  for (const [code, name, t, tax_code, address, phone, contact_name, payment_terms, active] of customers) {
    const r = await admin.post('/api/sales/customers', {
      data: { code, name, tax_code, address, phone, email: null, contact_name, payment_terms, org_unit_id: team[t], active },
    })
    customerId[code] =
      r.status() === 409
        ? (await get<{ items: { id: number; code: string }[] }>(admin, `/api/sales/customers?q=${code}&page_size=20`)).items.find((x) => x.code === code)!.id
        : await created(Promise.resolve(r))
    customerTerms[code] = payment_terms
  }
  const teamOf = (customer: string) => team[customers.find((c) => c[0] === customer)![2]]

  const sessions: Record<string, APIRequestContext> = {}
  for (const login of ['kd.hn', 'kd.hn2', 'kd.hcm', 'ql.kd.hn', 'ql.kd.hcm', 'gd.kinhdoanh']) {
    sessions[login] = await signedInApi(baseURL!, login, password)
  }
  const lines = (ls: Line[]) =>
    ls.map(([code, quantity, discount_percent, price]) => ({
      item_id: itemId[code],
      description: itemOf[code]![1],
      quantity,
      unit_price: price ?? itemOf[code]![3],
      discount_percent,
      vat_rate: itemOf[code]![4],
    }))

  const docType = (k: Kind) => (k === 'quotes' ? 'sales.quote' : 'sales.order')
  const version = async (as: APIRequestContext, k: Kind, id: number) => (await get<{ version: number }>(as, `/api/sales/${k}/${id}`)).version
  const move = async (as: APIRequestContext, k: Kind, id: number, to: 'posted' | 'cancelled') =>
    ok(as.post(`/api/documents/${docType(k)}/${id}/transitions`, { data: { to, version: await version(as, k, id) } }), `${to} ${k} ${id}`)
  // Whoever has it in their inbox decides, step by step.
  async function decide(id: number, reject?: string) {
    for (let k = 0; k < 5; k++) {
      let item: { instance_id: number; doc_id: number; step: number } | undefined
      let by: APIRequestContext | undefined
      for (const s of Object.values(sessions)) {
        item = (await get<{ items: { instance_id: number; doc_id: number; step: number }[] }>(s, '/api/approvals/inbox')).items.find((i) => i.doc_id === id)
        if (item) {
          by = s
          break
        }
      }
      if (!item) return
      await ok(by!.post(`/api/approvals/${item.instance_id}/${reject ? 'reject' : 'approve'}`, { data: reject ? { step: item.step, reason: reject } : { step: item.step } }), `decide ${id}`)
      if (reject) return
    }
  }
  async function play(as: APIRequestContext, k: Kind, id: number, outcome: Outcome, why?: string) {
    if (outcome === 'draft') return
    await move(as, k, id, 'posted')
    if (outcome === 'approved' || outcome === 'cancelled') await decide(id)
    if (outcome === 'rejected') await decide(id, why ?? 'Không phù hợp')
    if (outcome === 'cancelled') await move(as, k, id, 'cancelled')
  }

  for (const q of quotes) {
    const as = sessions[q.by]!
    const id = await created(
      as.post('/api/sales/quotes', {
        data: {
          request_id: crypto.randomUUID(),
          date: q.date,
          org_unit_id: teamOf(q.customer),
          customer_id: customerId[q.customer],
          valid_until: q.until,
          delivery_date: null,
          payment_terms: customerTerms[q.customer],
          delivery_terms: 'Giao tại địa chỉ khách hàng',
          note: null,
          lines: lines(q.lines),
        },
      }),
    )
    await play(as, 'quotes', id, q.outcome, q.why)
    if (q.order) {
      const order = await created(as.post(`/api/sales/quotes/${id}/order`))
      await play(as, 'orders', order, q.order)
    }
  }
  for (const o of orders) {
    const as = sessions[o.by]!
    const id = await created(
      as.post('/api/sales/orders', {
        data: {
          request_id: crypto.randomUUID(),
          date: o.date,
          org_unit_id: teamOf(o.customer),
          customer_id: customerId[o.customer],
          valid_until: null,
          delivery_date: o.delivery ?? null,
          payment_terms: customerTerms[o.customer],
          delivery_terms: null,
          note: null,
          lines: lines(o.lines),
        },
      }),
    )
    await play(as, 'orders', id, o.outcome)
  }
})
