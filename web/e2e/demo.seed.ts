import { expect, test, type APIRequestContext, type APIResponse } from '@playwright/test'
import { created, signedInApi } from './api'

// The Demo demo company, built from an empty database (make reset-db) through the API:
// org chart, accounts and roles, employees with sensitive data and dependents, leave types and
// balances, contracts, leave and overtime requests in every status, timesheets, the work
// calendar, July 2026 payrolls posted, August computed with adjustments, and a period lock.

const password = 'Demo@2026'
type Unit = 'bgd' | 'hcns' | 'kt' | 'kdhn' | 'kythuat' | 'kdhcm' | 'kho' | 'dp' | 'dx'
type Outcome = 'approved' | 'partial' | 'pending' | 'draft' | 'rejected' | 'cancelled'
type DocType = 'hrm.contract' | 'hrm.leave_request' | 'hrm.overtime_request' | 'hrm.timesheet' | 'hrm.payroll'
const docPath: Record<DocType, string> = {
  'hrm.contract': 'contracts',
  'hrm.leave_request': 'leaves',
  'hrm.overtime_request': 'overtimes',
  'hrm.timesheet': 'timesheets',
  'hrm.payroll': 'payrolls',
}

// [code, full name, gender, birth date, unit, hire date, termination date, account login]
const people: [string, string, 'male' | 'female', string, Unit, string, string?, string?][] = [
  ['NV0001', 'Nguyễn Văn Minh', 'male', '1972-03-15', 'bgd', '2015-01-05', undefined, 'giamdoc'],
  ['NV0002', 'Trần Thị Thu Hà', 'female', '1980-07-22', 'bgd', '2016-03-01', undefined, 'pgd.nhansu'],
  ['NV0003', 'Lê Thị Hồng', 'female', '1988-11-02', 'hcns', '2018-06-01', undefined, 'nhansu.hn'],
  ['NV0004', 'Phạm Văn Đức', 'male', '1993-05-14', 'hcns', '2021-09-06'],
  ['NV0005', 'Đỗ Thị Mai', 'female', '1997-01-30', 'hcns', '2024-02-19'],
  ['NV0006', 'Hoàng Văn Nam', 'male', '1985-09-09', 'kt', '2017-04-03', undefined, 'tp.ketoan.hn'],
  ['NV0007', 'Vũ Thị Lan', 'female', '1990-12-12', 'kt', '2019-08-12', undefined, 'ketoan.luong'],
  ['NV0008', 'Bùi Thị Ngọc', 'female', '1995-04-18', 'kt', '2022-03-01'],
  ['NV0009', 'Ngô Văn Hùng', 'male', '1984-02-25', 'kdhn', '2016-10-10', undefined, 'tp.kinhdoanh.hn'],
  ['NV0010', 'Đặng Quốc Tuấn', 'male', '1991-08-08', 'kdhn', '2020-01-06'],
  ['NV0011', 'Trịnh Thu Trang', 'female', '1996-06-21', 'kdhn', '2023-05-15'],
  ['NV0012', 'Dương Văn Long', 'male', '1994-10-03', 'kdhn', '2021-02-22', '2026-03-31'],
  ['NV0013', 'Tạ Quang An', 'male', '1998-03-11', 'kdhn', '2024-07-01', undefined, 'an.tq'],
  ['NV0014', 'Lý Thị Hương', 'female', '1999-09-27', 'kdhn', '2026-08-17'],
  ['NV0015', 'Mai Văn Thắng', 'male', '1983-01-19', 'kythuat', '2015-08-03', undefined, 'tp.kythuat.hn'],
  ['NV0016', 'Phan Thanh Bình', 'male', '1989-07-07', 'kythuat', '2018-11-05'],
  ['NV0017', 'Cao Minh Khoa', 'male', '1992-11-23', 'kythuat', '2019-03-18'],
  ['NV0018', 'Hồ Thị Thanh', 'female', '1994-02-14', 'kythuat', '2020-09-14'],
  ['NV0019', 'Lâm Đức Thịnh', 'male', '1997-05-05', 'kythuat', '2022-06-06'],
  ['NV0020', 'Kiều Văn Phúc', 'male', '1995-12-30', 'kythuat', '2023-01-09', '2026-08-14'],
  ['NV0021', 'Nguyễn Hoàng Yến', 'female', '2001-04-04', 'kythuat', '2026-10-01'],
  ['NV0022', 'Trương Văn Lợi', 'male', '1986-06-16', 'kdhcm', '2017-02-13', undefined, 'tp.kinhdoanh.hcm'],
  ['NV0023', 'Huỳnh Thị Kim', 'female', '1993-03-03', 'kdhcm', '2019-07-01'],
  ['NV0024', 'Lưu Minh Tâm', 'male', '1996-09-12', 'kdhcm', '2021-04-12'],
  ['NV0025', 'Châu Ngọc Ánh', 'female', '1998-08-28', 'kdhcm', '2023-09-04'],
  ['NV0026', 'Tôn Thất Hiếu', 'male', '1990-01-08', 'kdhcm', '2020-05-18'],
  ['NV0027', 'Võ Văn Sang', 'male', '1982-04-20', 'kho', '2016-05-09', undefined, 'tp.khovan.hcm'],
  ['NV0028', 'Đinh Văn Tài', 'male', '1991-11-11', 'kho', '2018-08-20'],
  ['NV0029', 'La Văn Phước', 'male', '1994-07-17', 'kho', '2020-12-01'],
  ['NV0030', 'Thái Thị Diễm', 'female', '1997-02-02', 'kho', '2022-10-17'],
  ['NV0031', 'Quách Văn Khải', 'male', '1999-10-10', 'kho', '2024-03-04'],
  ['NV0032', 'Ông Văn Tú', 'male', '1990-05-25', 'kho', '2019-04-01', '2025-12-31'],
  ['NV0033', 'Phùng Văn Hải', 'male', '1981-08-30', 'dp', '2018-01-02', undefined, 'tp.dieuphoi'],
  ['NV0034', 'Hà Thị Nhung', 'female', '1992-12-05', 'dp', '2019-10-07', undefined, 'nhansu.hcm'],
  ['NV0035', 'Đoàn Văn Kiên', 'male', '1995-03-27', 'dp', '2021-07-19'],
  ['NV0036', 'Từ Minh Nhật', 'male', '1988-06-06', 'dx', '2017-09-11', undefined, 'tp.doixe'],
  ['NV0037', 'Lại Văn Thành', 'male', '1987-02-17', 'dx', '2018-04-23'],
  ['NV0038', 'Kim Văn Hậu', 'male', '1990-09-19', 'dx', '2020-03-02'],
  ['NV0039', 'Mạc Văn Toàn', 'male', '1993-11-29', 'dx', '2021-11-15'],
  ['NV0040', 'Giáp Văn Lực', 'male', '1996-01-21', 'dx', '2023-06-12'],
  ['NV0041', 'Ninh Văn Bảo', 'male', '1998-07-03', 'dx', '2025-08-04'],
]

// [employee, name, relationship, birth date, deduction from, deduction to]
const dependents: [string, string, 'child' | 'spouse' | 'parent', string, string, string?][] = [
  ['NV0001', 'Nguyễn Minh Khang', 'child', '2008-04-10', '2020-01', '2026-04'],
  ['NV0001', 'Nguyễn Minh Châu', 'child', '2012-09-01', '2020-01'],
  ['NV0006', 'Hoàng Gia Bảo', 'child', '2015-06-20', '2020-01'],
  ['NV0009', 'Ngô Bảo Ngọc', 'child', '2014-02-11', '2020-01'],
  ['NV0009', 'Phạm Thị Lựu', 'parent', '1955-05-05', '2023-01'],
  ['NV0015', 'Mai Anh Thư', 'child', '2012-12-24', '2020-01'],
  ['NV0015', 'Mai Đức Anh', 'child', '2016-03-08', '2020-01'],
  ['NV0016', 'Phan Gia Huy', 'child', '2020-10-02', '2020-11'],
  ['NV0022', 'Trương Mỹ Linh', 'child', '2018-07-15', '2020-01'],
  ['NV0027', 'Nguyễn Thị Bích', 'spouse', '1985-01-01', '2024-01'],
  ['NV0027', 'Võ Minh Quân', 'child', '2017-11-30', '2020-01'],
  ['NV0033', 'Phùng Thu Hằng', 'child', '2010-08-08', '2020-01'],
  ['NV0037', 'Lại Văn Bình', 'parent', '1950-03-03', '2022-01'],
]

const holidays: [string, string][] = [
  ['2026-01-01', 'Tết Dương lịch'],
  ['2026-02-16', 'Tết Nguyên đán'],
  ['2026-02-17', 'Tết Nguyên đán'],
  ['2026-02-18', 'Tết Nguyên đán'],
  ['2026-02-19', 'Tết Nguyên đán'],
  ['2026-02-20', 'Tết Nguyên đán'],
  ['2026-04-27', 'Nghỉ bù Giỗ Tổ Hùng Vương'],
  ['2026-04-30', 'Ngày Giải phóng miền Nam'],
  ['2026-05-01', 'Quốc tế Lao động'],
  ['2026-09-01', 'Quốc khánh'],
  ['2026-09-02', 'Quốc khánh'],
]

// [employee, leave type, start, end, reason, outcome]; a "self:" request is filed from the employee's own account.
const leaves: [string, string, string, string, string, Outcome][] = [
  ['NV0004', 'Phép năm', '2026-07-06', '2026-07-07', 'Việc gia đình', 'approved'],
  ['NV0010', 'Phép năm', '2026-07-13', '2026-07-17', 'Du lịch hè cùng gia đình', 'approved'],
  ['NV0017', 'Nghỉ ốm', '2026-07-20', '2026-07-21', 'Sốt xuất huyết', 'approved'],
  ['NV0028', 'Phép năm', '2026-07-24', '2026-07-24', 'Đi khám sức khỏe', 'approved'],
  ['NV0035', 'Việc riêng có lương', '2026-07-27', '2026-07-29', 'Kết hôn', 'approved'],
  ['NV0038', 'Nghỉ không lương', '2026-07-30', '2026-07-31', 'Về quê có việc', 'approved'],
  ['NV0011', 'Phép năm', '2026-08-03', '2026-08-04', 'Việc cá nhân', 'approved'],
  ['NV0030', 'Thai sản', '2026-08-03', '2026-12-02', 'Nghỉ sinh con', 'approved'],
  ['NV0023', 'Phép năm', '2026-08-10', '2026-08-14', 'Nghỉ hè', 'approved'],
  ['NV0018', 'Nghỉ ốm', '2026-08-18', '2026-08-18', 'Cảm cúm', 'approved'],
  ['NV0040', 'Phép năm', '2026-08-20', '2026-08-21', 'Việc gia đình', 'rejected'],
  ['NV0026', 'Phép năm', '2026-08-24', '2026-08-25', 'Đi du lịch', 'cancelled'],
  ['NV0016', 'Phép năm', '2026-09-03', '2026-09-04', 'Nghỉ nối lễ Quốc khánh', 'approved'],
  ['NV0024', 'Phép năm', '2026-09-07', '2026-09-07', 'Việc cá nhân', 'approved'],
  ['self:NV0013', 'Phép năm', '2026-09-14', '2026-09-15', 'Về quê thăm bố mẹ', 'approved'],
  ['NV0039', 'Nghỉ ốm', '2026-09-21', '2026-09-23', 'Đau lưng', 'approved'],
  ['NV0005', 'Phép năm', '2026-09-28', '2026-09-30', 'Du lịch', 'approved'],
  ['NV0041', 'Phép năm', '2026-10-08', '2026-10-09', 'Việc gia đình', 'rejected'],
  ['NV0019', 'Phép năm', '2026-10-12', '2026-10-16', 'Du lịch nước ngoài', 'partial'],
  ['self:NV0013', 'Phép năm', '2026-10-19', '2026-10-20', 'Đám cưới bạn thân', 'pending'],
  ['NV0029', 'Việc riêng có lương', '2026-10-26', '2026-10-28', 'Tang lễ ông nội', 'pending'],
  ['NV0008', 'Phép năm', '2026-11-02', '2026-11-06', 'Du lịch', 'draft'],
  ['NV0025', 'Phép năm', '2026-12-28', '2026-12-31', 'Nghỉ cuối năm', 'draft'],
]

// [employee, date, day kind, day hours, night hours, reason, outcome]. Logistics works Saturdays.
type Overtime = [string, string, 'weekday' | 'weekly_off' | 'holiday', string, string, string, Outcome]
const overtimes: Overtime[] = [
  ['NV0016', '2026-07-04', 'weekly_off', '8', '0', 'Nâng cấp máy chủ', 'approved'],
  ['NV0017', '2026-07-08', 'weekday', '3', '0', 'Xử lý sự cố hệ thống', 'approved'],
  ['NV0028', '2026-07-11', 'weekly_off', '6', '0', 'Kiểm kê kho', 'approved'],
  ['NV0029', '2026-07-11', 'weekly_off', '6', '0', 'Kiểm kê kho', 'approved'],
  ['NV0010', '2026-07-15', 'weekday', '2', '0', 'Gặp khách hàng ngoài giờ', 'approved'],
  ['NV0038', '2026-07-19', 'weekly_off', '8', '2', 'Giao hàng gấp cho khách', 'approved'],
  ['NV0008', '2026-07-30', 'weekday', '3', '0', 'Chốt số liệu cuối tháng', 'approved'],
  ['NV0007', '2026-07-31', 'weekday', '2.5', '0', 'Chốt số liệu cuối tháng', 'approved'],
  // More than the monthly limit: the August payroll warns about it.
  ...['02', '09', '16', '23', '30'].map((d): Overtime => ['NV0037', `2026-08-${d}`, 'weekly_off', '8', '0', 'Chạy tuyến Bắc – Nam', 'approved']),
  ['NV0037', '2026-08-12', 'weekday', '3', '0', 'Chạy tuyến Bắc – Nam', 'approved'],
  ['NV0024', '2026-08-08', 'weekly_off', '4', '0', 'Hỗ trợ sự kiện khách hàng', 'approved'],
  ['NV0018', '2026-08-19', 'weekday', '2', '2', 'Triển khai phần mềm cho khách', 'approved'],
  ['NV0035', '2026-08-22', 'weekday', '3', '0', 'Điều phối chuyến hàng gấp', 'approved'],
  ['NV0011', '2026-08-27', 'weekday', '2', '0', 'Chuẩn bị hồ sơ thầu', 'rejected'],
  ['NV0026', '2026-08-28', 'weekday', '2', '0', 'Hỗ trợ đối tác', 'cancelled'],
  ['NV0016', '2026-09-01', 'holiday', '8', '0', 'Bảo trì hệ thống dịp lễ', 'approved'],
  ['NV0038', '2026-09-02', 'holiday', '6', '0', 'Giao hàng dịp Quốc khánh', 'approved'],
  ['NV0039', '2026-09-02', 'holiday', '6', '0', 'Giao hàng dịp Quốc khánh', 'partial'],
  ['NV0029', '2026-09-12', 'weekly_off', '5', '0', 'Nhập hàng', 'approved'],
  ['NV0019', '2026-09-17', 'weekday', '0', '2', 'Theo dõi sao lưu dữ liệu', 'approved'],
  ['NV0025', '2026-09-24', 'weekday', '3', '0', 'Hoàn thành báo cáo quý', 'approved'],
  ['self:NV0013', '2026-09-24', 'weekday', '2', '0', 'Gặp khách hàng ngoài giờ', 'approved'],
  ['self:NV0013', '2026-10-02', 'weekday', '2', '0', 'Gặp khách hàng ngoài giờ', 'pending'],
  ['NV0017', '2026-10-03', 'weekly_off', '8', '0', 'Di chuyển phòng máy', 'pending'],
  ['NV0031', '2026-10-05', 'weekday', '2', '0', 'Kiểm kê kho', 'pending'],
  ['NV0010', '2026-10-09', 'weekday', '3', '0', 'Chuẩn bị hội nghị khách hàng', 'draft'],
]

const baseSalary: Record<Unit, number> = { bgd: 30e6, hcns: 14e6, kt: 16e6, kdhn: 13e6, kythuat: 22e6, kdhcm: 13e6, kho: 10e6, dp: 12e6, dx: 11e6 }

const iso = (d: Date) => d.toISOString().slice(0, 10)
const day = (s: string) => new Date(`${s}T00:00:00Z`)
const addDays = (s: string, n: number) => iso(new Date(day(s).getTime() + n * 86_400_000))
const addMonths = (s: string, n: number) => {
  const d = day(s)
  d.setUTCMonth(d.getUTCMonth() + n)
  return iso(d)
}
const max = (a: string, b: string) => (a > b ? a : b)
const round = (n: number) => Math.round(n / 100_000) * 100_000
const half = (n: number) => String(Math.round(n * 2) / 2)
const pad = (n: number, w: number) => String(n).padStart(w, '0')

test('seed the Demo demo company', async ({ baseURL }) => {
  test.setTimeout(20 * 60_000)
  const admin = await signedInApi(baseURL!)
  const ok = async (res: Promise<APIResponse>, what: string) => {
    const r = await res
    expect([200, 202, 204], `${what}: ${await r.text()}`).toContain(r.status())
    return r
  }
  const get = async <T>(as: APIRequestContext, url: string) => (await (await ok(as.get(url), url)).json()) as T
  const existing = await get<{ total: number }>(admin, '/api/hrm/employees')
  expect(existing.total, 'the database must be empty: run make reset-db first').toBe(0)

  // Org chart.
  const unit = async (data: Record<string, unknown>) => created(admin.post('/api/org-units', { data }))
  const group = await unit({ parent_id: null, kind: 'group', name: 'Tập đoàn Demo' })
  const cp = await unit({ parent_id: group, kind: 'company', name: 'Công ty CP Demo', tax_code: '0101234567', legal_name: 'Công ty Cổ phần Demo', address: '12 Láng Hạ, Đống Đa, Hà Nội' })
  const lg = await unit({ parent_id: group, kind: 'company', name: 'Công ty TNHH Demo Logistics', tax_code: '0312345678', legal_name: 'Công ty TNHH Demo Logistics', address: '45 Nguyễn Thị Minh Khai, Quận 1, TP.HCM' })
  const hn = await unit({ parent_id: cp, kind: 'branch', name: 'Chi nhánh Hà Nội' })
  const hcm = await unit({ parent_id: cp, kind: 'branch', name: 'Chi nhánh TP.HCM' })
  const dept = async (parent: number, name: string) => unit({ parent_id: parent, kind: 'department', name })
  const units: Record<Unit, number> = {
    bgd: await dept(cp, 'Ban Giám đốc'),
    hcns: await dept(hn, 'Hành chính – Nhân sự'),
    kt: await dept(hn, 'Kế toán'),
    kdhn: await dept(hn, 'Kinh doanh (HN)'),
    kythuat: await dept(hn, 'Kỹ thuật'),
    kdhcm: await dept(hcm, 'Kinh doanh (HCM)'),
    kho: await dept(hcm, 'Kho vận'),
    dp: await dept(lg, 'Điều phối vận tải'),
    dx: await dept(lg, 'Đội xe'),
  }
  const company = (u: Unit) => (u === 'dp' || u === 'dx' ? lg : cp)

  // Both companies are in wage region 1; Logistics works Saturdays.
  for (const c of [cp, lg]) {
    await ok(admin.put(`/api/legal-entities/${c}/settings/hrm.wage_region`, { data: { value: '1' } }), 'wage region')
    for (const [date, name] of holidays) await ok(admin.put(`/api/hrm/calendars/${c}/holidays/${date}`, { data: { name } }), `holiday ${date}`)
  }
  await ok(admin.put(`/api/hrm/calendars/${lg}/weeks/2026-01-01`, { data: { off_days: [0] } }), 'work week')
  const offDays = (c: number) => (c === lg ? [0] : [0, 6])
  const isHoliday = new Set(holidays.map(([d]) => d))
  const workdays = (c: number, from: string, to: string) => {
    const out: string[] = []
    for (let d = from; d <= to; d = addDays(d, 1)) if (!offDays(c).includes(day(d).getUTCDay()) && !isHoliday.has(d)) out.push(d)
    return out
  }

  // Accounts. Heads see their department; HR officers their branch; the deputy director everything.
  const grants: Record<string, [string, number | null][]> = {
    giamdoc: [['hr', null], ['payroll_viewer', null]],
    'pgd.nhansu': [['hr', null], ['payroll', null], ['sensitive_viewer', null], ['leave_admin', null]],
    'nhansu.hn': [['hr', hn], ['sensitive_viewer', hn], ['leave_admin', hn]],
    'nhansu.hcm': [['hr', hcm], ['hr', lg], ['sensitive_viewer', hcm], ['sensitive_viewer', lg]],
    'ketoan.luong': [['payroll_viewer', null]],
    'tp.ketoan.hn': [['viewer', units.kt]],
    'tp.kinhdoanh.hn': [['viewer', units.kdhn]],
    'tp.kythuat.hn': [['viewer', units.kythuat]],
    'tp.kinhdoanh.hcm': [['viewer', units.kdhcm]],
    'tp.khovan.hcm': [['viewer', units.kho]],
    'tp.dieuphoi': [['viewer', units.dp]],
    'tp.doixe': [['viewer', units.dx]],
    'an.tq': [],
  }
  const userIds: Record<string, number> = {}
  const sessions: Record<string, APIRequestContext> = {}
  for (const [, name, , , , , , login] of people) {
    if (!login) continue
    const id = await created(admin.post('/api/users', { data: { login, name, password } }))
    for (const [role, org_unit_id] of grants[login]!) await created(admin.post(`/api/users/${id}/roles`, { data: { product: 'hrm', role, org_unit_id } }))
    userIds[login] = id
    sessions[login] = await signedInApi(baseURL!, login, password)
  }

  // Employees, with sensitive data except for the newest hire, who has not handed it in yet.
  const streets = { hn: ['Cầu Giấy', 'Đống Đa', 'Thanh Xuân', 'Hoàng Mai', 'Long Biên'], hcm: ['Quận 3', 'Bình Thạnh', 'Gò Vấp', 'Thủ Đức', 'Quận 7'] }
  const ascii = (s: string) => s.normalize('NFD').replace(/[̀-ͯ]/g, '').replace(/đ/g, 'd').replace(/Đ/g, 'D').toLowerCase()
  type Emp = { id: number; code: string; unit: Unit; company: number; head: boolean; hire: string; end?: string; n: number }
  const emps: Record<string, Emp> = {}
  const heads: Partial<Record<Unit, number>> = {}
  for (const [code, full_name, gender, dob, u, hire, end, login] of people) {
    const n = Number(code.slice(2))
    const words = ascii(full_name).split(' ')
    const head = !heads[u]
    const city = company(u) === lg || u === 'kdhcm' || u === 'kho' ? 'hcm' : 'hn'
    const manager_id = code === 'NV0001' ? null : head ? emps.NV0001!.id : heads[u]!
    const data = {
      code,
      full_name,
      date_of_birth: dob,
      gender,
      phone: `09${pad((n * 7_654_321) % 100_000_000, 8)}`,
      email: `${words.at(-1)}.${words.slice(0, -1).map((w) => w[0]).join('')}${n}@example.com`,
      address: `${n * 3 + 1} ${city === 'hn' ? 'phố' : 'đường'} số ${n}, ${streets[city][n % 5]}, ${city === 'hn' ? 'Hà Nội' : 'TP.HCM'}`,
      org_unit_id: units[u],
      manager_id,
      user_login: login ?? null,
      hire_date: hire,
      termination_date: end ?? null,
      ...(code === 'NV0021'
        ? {}
        : {
            sensitive: {
              national_id: `001${gender === 'male' ? 0 : 1}${dob.slice(2, 4)}${pad(n * 1_237, 6)}`,
              social_insurance_no: `01${pad(n * 2_345_671, 8).slice(-8)}`,
              tax_code: `80${pad(n * 3_456_789, 8).slice(-8)}`,
              bank_account: `${['Vietcombank', 'Techcombank', 'BIDV', 'MB Bank'][n % 4]} – ${pad(n * 98_765_432, 13)}`,
            },
          }),
    }
    const id = await created(admin.post('/api/hrm/employees', { data }))
    if (head) heads[u] = id
    emps[code] = { id, code, unit: u, company: company(u), head, hire, end, n }
  }
  for (const [code, full_name, relationship, date_of_birth, deduction_from, deduction_to] of dependents) {
    await created(
      admin.post(`/api/hrm/employees/${emps[code]!.id}/dependents`, {
        data: { full_name, relationship, date_of_birth, id_number: `0${pad(Number(date_of_birth.replace(/-/g, '')) * 7, 11)}`, deduction_from, deduction_to: deduction_to ?? null },
      }),
    )
  }

  // Leave types and the 2026 balances: 12 days, a day more for each five years, prorated for
  // those who joined or left in 2026, and some days carried over from 2025.
  const leaveType: Record<string, number> = {}
  for (const [name, deducts_balance, paid, active] of [
    ['Phép năm', true, true, true],
    ['Nghỉ ốm', false, false, true],
    ['Việc riêng có lương', false, true, true],
    ['Nghỉ không lương', false, false, true],
    ['Thai sản', false, false, true],
    ['Nghỉ bù (cũ)', false, true, false],
  ] as const) {
    leaveType[name] = await created(admin.post('/api/hrm/leave-types', { data: { name, deducts_balance, paid, active } }))
  }
  const adjust = (e: Emp, delta: string, reason: string) => ok(admin.post(`/api/hrm/employees/${e.id}/leave-balances/2026/adjustments`, { data: { delta, reason } }), `balance ${e.code}`)
  for (const e of Object.values(emps)) {
    if (e.end && e.end < '2026-01-01') continue
    const from = max(e.hire, '2026-01-01')
    const to = e.end ?? '2026-12-31'
    const months = (day(to).getUTCFullYear() - day(from).getUTCFullYear()) * 12 + day(to).getUTCMonth() - day(from).getUTCMonth() + 1
    await adjust(e, half((12 * months) / 12), 'Phép năm 2026')
    const years = Math.floor((day('2026-01-01').getTime() - day(e.hire).getTime()) / (365.25 * 86_400_000))
    if (years >= 5) await adjust(e, String(Math.floor(years / 5)), 'Phép thâm niên')
    if (e.hire < '2025-01-01' && e.n % 4 === 0) await adjust(e, '2.5', 'Phép năm 2025 chuyển sang')
  }

  // Approval rules: heads decide leave and overtime, the deputy director long leave, holiday
  // overtime and timesheets, the director contracts and payrolls.
  const rule = (type: DocType, steps: unknown[]) =>
    ok(admin.put(`/api/approval-rules/${type}`, { data: { steps, max_levels: 3, fallback_product: 'hrm', fallback_role: 'hr' } }), `rule ${type}`)
  const user = (login: string) => ({ kind: 'user', user_id: userIds[login] })
  await rule('hrm.leave_request', [{ approver: { kind: 'module' } }, { condition: { field: 'days', op: 'gt', value: '3' }, approver: user('pgd.nhansu') }])
  await rule('hrm.overtime_request', [{ approver: { kind: 'module' } }, { condition: { field: 'day_kind', op: 'eq', value: 'holiday' }, approver: user('pgd.nhansu') }])
  await rule('hrm.contract', [{ approver: user('giamdoc') }])
  await rule('hrm.timesheet', [{ approver: user('pgd.nhansu') }])
  await rule('hrm.payroll', [{ approver: user('giamdoc') }])

  // Workflow: send, then whoever has it in their inbox decides, step by step.
  const version = async (as: APIRequestContext, type: DocType, id: number) => (await get<{ version: number }>(as, `/api/hrm/${docPath[type]}/${id}`)).version
  const move = async (as: APIRequestContext, type: DocType, id: number, to: 'posted' | 'cancelled') =>
    ok(as.post(`/api/documents/${type}/${id}/transitions`, { data: { to, version: await version(as, type, id) } }), `${to} ${type} ${id}`)
  async function decide(id: number, steps: number, reject?: string) {
    for (let k = 0; k < steps; k++) {
      let item: { instance_id: number; doc_id: number; step: number } | undefined
      let by: APIRequestContext | undefined
      for (const s of [admin, ...Object.values(sessions)]) {
        item = (await get<{ items: { instance_id: number; doc_id: number; step: number }[] }>(s, '/api/approvals/inbox')).items.find((i) => i.doc_id === id)
        if (item) {
          by = s
          break
        }
      }
      if (!item) return
      const data = reject ? { step: item.step, reason: reject } : { step: item.step }
      await ok(by!.post(`/api/approvals/${item.instance_id}/${reject ? 'reject' : 'approve'}`, { data }), `decide ${id}`)
      if (reject) return
    }
  }
  async function play(as: APIRequestContext, type: DocType, id: number, outcome: Outcome, why = 'Không phù hợp kế hoạch công việc') {
    if (outcome === 'draft') return
    await move(as, type, id, 'posted')
    if (outcome === 'approved' || outcome === 'cancelled') await decide(id, 5)
    if (outcome === 'partial') await decide(id, 1)
    if (outcome === 'rejected') await decide(id, 1, why)
    if (outcome === 'cancelled') await move(admin, type, id, 'cancelled')
  }

  // Contracts, in đồng; heads get a position allowance.
  const kinds: Record<string, number> = {}
  for (const [name, fixed_term, active] of [
    ['Không xác định thời hạn', false, true],
    ['Xác định thời hạn 12 tháng', true, true],
    ['Xác định thời hạn 36 tháng', true, true],
    ['Thử việc', true, true],
    ['Hợp đồng mùa vụ', true, false],
  ] as const) {
    kinds[name] = await created(admin.post('/api/hrm/contract-types', { data: { name, fixed_term, active } }))
  }
  function terms(e: Emp, raise = 0) {
    const salary = e.code === 'NV0001' ? 60e6 : round(baseSalary[e.unit] * (e.head ? 1.6 : 1) * (1 + (e.n % 5) * 0.06) * (1 + raise))
    const lines = [{ kind: 'support', name: 'Ăn ca', amount: 730_000, taxable: false }]
    if (e.head) lines.unshift({ kind: 'allowance', name: 'Phụ cấp chức vụ', amount: e.unit === 'bgd' ? 8_000_000 : 3_000_000, taxable: true })
    if (e.unit === 'kho') lines.unshift({ kind: 'allowance', name: 'Phụ cấp độc hại', amount: 1_000_000, taxable: true })
    if (e.unit === 'dx') lines.unshift({ kind: 'allowance', name: 'Phụ cấp lái xe', amount: 1_500_000, taxable: true })
    if (e.unit === 'kdhn' || e.unit === 'kdhcm') {
      lines.push({ kind: 'support', name: 'Xăng xe', amount: 800_000, taxable: true }, { kind: 'support', name: 'Điện thoại', amount: 300_000, taxable: true })
    }
    return { salary, lines }
  }
  const contract = (e: Emp, k: string, start: string, end: string | null, raise = 0) =>
    created(admin.post('/api/hrm/contracts', { data: { employee_id: e.id, contract_type_id: kinds[k], start_date: start, end_date: end, terms: terms(e, raise) } }))
  const appendix = (e: Emp, parent: number, start: string, raise: number) =>
    created(admin.post('/api/hrm/contracts', { data: { employee_id: e.id, parent_id: parent, contract_type_id: 0, start_date: start, end_date: null, terms: terms(e, raise) } }))
  const post = async (id: number) => play(admin, 'hrm.contract', id, 'approved')

  const today = iso(new Date())
  let j = 0
  for (const e of Object.values(emps)) {
    if (e.end) {
      // Left: a fixed-term contract that ran to the last day.
      await post(await contract(e, 'Xác định thời hạn 36 tháng', max(e.hire, addDays(addMonths(e.end, -36), 1)), e.end))
      continue
    }
    if (e.hire >= '2026-08-01') {
      // New: probation, then for the earlier one a 12-month contract still drafted, for the later one probation awaits approval.
      const after = addMonths(e.hire, 2)
      const probation = await contract(e, 'Thử việc', e.hire, addDays(after, -1), -0.15)
      if (e.hire < today) {
        await post(probation)
        await contract(e, 'Xác định thời hạn 12 tháng', after, addDays(addMonths(after, 12), -1))
      } else await play(admin, 'hrm.contract', probation, 'pending')
      continue
    }
    switch (j++ % 5) {
      case 0: {
        // Probation, then open-ended.
        const after = addMonths(e.hire, 2)
        await post(await contract(e, 'Thử việc', e.hire, addDays(after, -1), -0.15))
        await post(await contract(e, 'Không xác định thời hạn', after, null))
        break
      }
      case 1: {
        // A one-year contract ending within the next four weeks; the first one's renewal is drafted.
        const end = addDays(today, 3 + ((e.n * 5) % 25))
        await post(await contract(e, 'Xác định thời hạn 12 tháng', addDays(addMonths(end, -12), 1), end))
        if (j === 2) await contract(e, 'Xác định thời hạn 36 tháng', addDays(end, 1), addMonths(end, 36))
        break
      }
      case 2: {
        // Open-ended, raised twice by appendices.
        const id = await contract(e, 'Không xác định thời hạn', e.hire, null, -0.1)
        await post(id)
        await post(await appendix(e, id, max('2025-01-01', addMonths(e.hire, 6)), 0))
        await post(await appendix(e, id, '2026-07-01', 0.08))
        break
      }
      case 3: {
        // Three years to the end of 2026, raised mid-way; some get a raise from mid-October waiting for approval.
        const start = max(e.hire, '2024-01-01')
        const id = await contract(e, 'Xác định thời hạn 36 tháng', start, '2026-12-31')
        await post(id)
        await post(await appendix(e, id, max('2025-07-01', addMonths(start, 6)), 0.1))
        if (e.n % 2 === 0) await play(admin, 'hrm.contract', await appendix(e, id, '2026-10-15', 0.18), 'pending')
        break
      }
      case 4: {
        // A contract with a wrong salary, cancelled and replaced; or one sent back by the approver.
        if (e.n % 3 === 0) {
          const wrong = await contract(e, 'Xác định thời hạn 12 tháng', e.hire, addDays(addMonths(e.hire, 12), -1), 0.5)
          await play(admin, 'hrm.contract', wrong, 'cancelled')
        }
        const id = await contract(e, 'Không xác định thời hạn', e.hire, null)
        await post(id)
        if (e.n % 3 === 1) await play(admin, 'hrm.contract', await appendix(e, id, '2026-11-01', 0.3), 'rejected', 'Mức tăng vượt ngân sách năm 2026')
        break
      }
    }
  }

  // Leave and overtime requests; "self:" ones are filed by the employee's own account.
  const filer = (who: string) => (who.startsWith('self:') ? sessions['an.tq']! : admin)
  const emp = (who: string) => emps[who.replace('self:', '')]!
  const onLeave = new Map<number, Set<string>>()
  for (const [who, type, start_date, end_date, reason, outcome] of leaves) {
    const e = emp(who)
    const days = workdays(e.company, start_date, end_date)
    const as = filer(who)
    const data = { ...(as === admin ? { employee_id: e.id } : {}), leave_type_id: leaveType[type], start_date, end_date, days: String(days.length), reason }
    await play(as, 'hrm.leave_request', await created(as.post('/api/hrm/leaves', { data })), outcome, 'Bộ phận đang thiếu người')
    if (outcome === 'approved') onLeave.set(e.id, new Set([...(onLeave.get(e.id) ?? []), ...days]))
  }
  for (const [who, date, day_kind, day_hours, night_hours, reason, outcome] of overtimes) {
    const as = filer(who)
    const data = { ...(as === admin ? { employee_id: emp(who).id } : {}), date, day_kind, day_hours, night_hours, reason }
    await play(as, 'hrm.overtime_request', await created(as.post('/api/hrm/overtimes', { data })), outcome, 'Không có kế hoạch tăng ca được duyệt trước')
  }

  // Timesheets: July and August approved; September approved in Hà Nội, waiting in HCM, a draft at Logistics.
  // Every workday not on approved leave is worked, but for an absence or a half day now and then.
  for (const u of Object.keys(units) as Unit[]) {
    for (const month of ['2026-07', '2026-08', '2026-09']) {
      const id = await created(admin.post('/api/hrm/timesheets', { data: { org_unit_id: units[u], month } }))
      const ts = await get<{ version: number; period_start: string; period_end: string; employees: { id: number; hire_date: string; termination_date: string | null }[] }>(admin, `/api/hrm/timesheets/${id}`)
      const lines = []
      for (const e of ts.employees) {
        for (const date of workdays(company(u), max(ts.period_start, e.hire_date), e.termination_date && e.termination_date < ts.period_end ? e.termination_date : ts.period_end)) {
          const k = e.id + day(date).getUTCDate()
          if (onLeave.get(e.id)?.has(date) || k % 37 === 0) continue
          lines.push({ employee_id: e.id, date, days: k % 23 === 0 ? '0.5' : '1' })
        }
      }
      await ok(admin.put(`/api/hrm/timesheets/${id}`, { data: { version: ts.version, month, lines } }), `timesheet ${u} ${month}`)
      const outcome = month !== '2026-09' ? 'approved' : company(u) === lg ? 'draft' : ['kdhcm', 'kho'].includes(u) ? 'pending' : 'approved'
      await play(admin, 'hrm.timesheet', id, outcome)
    }
  }

  // Payrolls: July posted for both companies; August computed with July corrections at Demo,
  // sent to the director at Logistics.
  const pgd = sessions['pgd.nhansu']!
  const waitJob = (id: number) =>
    expect.poll(async () => (await get<{ state: string }>(pgd, `/api/jobs/${id}`)).state, { timeout: 180_000, intervals: [500] }).toBe('completed')
  for (const [c, month, outcome] of [
    [cp, '2026-07', 'approved'],
    [lg, '2026-07', 'approved'],
    [cp, '2026-08', 'draft'],
    [lg, '2026-08', 'pending'],
  ] as const) {
    const res = await pgd.post('/api/hrm/payrolls', { data: { legal_entity_id: c, month } })
    expect(res.status(), `payroll ${c} ${month}: ${await res.text()}`).toBe(201)
    const { id, job_id } = (await res.json()) as { id: number; job_id: number }
    await waitJob(job_id)
    if (c === cp && month === '2026-08') {
      const items = [
        { employee_id: emps.NV0010!.id, amount: 800_000, source_period: '2026-07', reason: 'Truy lĩnh hỗ trợ xăng xe tháng 7' },
        { employee_id: emps.NV0028!.id, amount: -500_000, source_period: '2026-07', reason: 'Thu hồi phụ cấp tính thừa tháng 7' },
      ]
      const r = await ok(pgd.put(`/api/hrm/payrolls/${id}/adjustments`, { data: { version: await version(pgd, 'hrm.payroll', id), items } }), 'adjustments')
      await waitJob(((await r.json()) as { job_id: number }).job_id)
    }
    await play(pgd, 'hrm.payroll', id, outcome)
    if (month === '2026-07' && c === cp) await ok(pgd.post('/api/exports/hrm.payroll', { data: { params: { payroll_id: id } } }), 'export')
  }

  // Demo closed its books up to June.
  await ok(admin.put(`/api/period-locks/${cp}`, { data: { locked_until: '2026-06-30' } }), 'period lock')
})
