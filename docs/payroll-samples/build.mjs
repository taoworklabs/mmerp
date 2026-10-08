// Dựng file bộ mẫu lương nháp. Cách tính theo các giả định ở sheet "Giả định";
// mỗi phần có công cụ ngoài thì đối chiếu và dừng nếu lệch dù một đồng.
// Cách chạy: xem README.md cùng thư mục.
import { createRequire } from 'node:module';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = process.env.ORACLE_ROOT ?? '/tmp';
const require = createRequire(join(root, 'noop.js'));
const esbuild = require('esbuild');
const ExcelJS = require('exceljs');
const here = dirname(fileURLToPath(import.meta.url));

async function load(entry, absWorkingDir) {
  const out = await esbuild.build({
    entryPoints: [entry], absWorkingDir, bundle: true, format: 'esm', platform: 'node', write: false, logLevel: 'error',
  });
  return import('data:text/javascript;base64,' + Buffer.from(out.outputFiles[0].text).toString('base64'));
}

const pit = await load('./src/lib/tax.ts', join(root, 'pit'));
const pitConst = await load('./src/config/constants.ts', join(root, 'pit'));
const thue = await load('./src/lib/taxCalculator.ts', join(root, 'thue-2026'));
const thueOt = await load('./src/lib/overtimeCalculator.ts', join(root, 'thue-2026'));

// ---- Tham số và lịch của công ty mẫu ----

// Làm tròn nửa xa số 0, giống ROUND của Excel.
const round = (x) => Math.sign(x) * Math.round(Math.abs(x));

const PERIODS = {
  '03/2026': { year: 2026, month: 3 },
  '04/2026': { year: 2026, month: 4, capSI: 46_800_000, calcDate: new Date(2026, 3, 15) },
  '07/2026': { year: 2026, month: 7, capSI: 50_600_000, calcDate: new Date(2026, 6, 15) },
};
const CAP_UI = 106_200_000; // vùng I
const PERSONAL = 15_500_000;
const DEPENDENT = 6_200_000;
const BRACKETS = [[10e6, 0.05], [30e6, 0.1], [60e6, 0.2], [100e6, 0.3], [Infinity, 0.35]];
const HOURS_PER_DAY = 8;
const OT_MONTH_LIMIT = 40; // giờ/tháng được miễn thuế; phần vượt chịu thuế
// Doanh nghiệp đóng: BHXH (gồm TNLĐ-BNN 0,5%), BHYT, BHTN, kinh phí công đoàn.
const EMPLOYER = { bhxh: 0.175, bhyt: 0.03, bhtn: 0.01, kpcd: 0.02 };
// Hệ số theo Điều 98 BLLĐ 2019, Điều 55–57 NĐ 145/2020; đêm ngày thường khi ban ngày không tăng ca.
const OT_RATE = { 'weekday/day': 1.5, 'weekend/day': 2, 'holiday/day': 3, 'weekday/night': 2 };

// Công ty làm thứ Hai–thứ Sáu; ngày lễ rơi vào ngày làm việc vẫn là ngày công hưởng lương.
function workdays(period) {
  const { year, month } = PERIODS[period];
  let n = 0;
  for (let d = new Date(year, month - 1, 1); d.getMonth() === month - 1; d.setDate(d.getDate() + 1)) {
    if (d.getDay() !== 0 && d.getDay() !== 6) n++;
  }
  return n;
}

function pitOf(taxable) {
  let tax = 0, prev = 0;
  for (const [top, rate] of BRACKETS) {
    if (taxable <= prev) break;
    tax += (Math.min(taxable, top) - prev) * rate;
    prev = top;
  }
  return round(tax);
}

// ---- Các trường hợp ----
// paid: số ngày công hưởng lương (mặc định đủ công); segments: đổi lương giữa kỳ.

const P = '04/2026';
const rows = [
  { code: 'S01', title: 'Lương thấp, không phát sinh thuế', salary: 8_000_000 },
  { code: 'S02', title: 'Một bậc thuế', salary: 18_000_000 },
  { code: 'S03', title: 'Hai bậc thuế', salary: 30_000_000 },
  { code: 'S04', title: 'Hai bậc thuế, 2 người phụ thuộc', salary: 30_000_000, deps: 2 },
  { code: 'S05', title: 'Ba bậc thuế, 1 người phụ thuộc', salary: 45_000_000, deps: 1 },
  { code: 'S06', title: 'Chạm trần BHXH/BHYT', salary: 55_000_000 },
  { code: 'S07', title: 'Chạm trần BHXH/BHYT, bốn bậc thuế', salary: 90_000_000 },
  { code: 'S08', title: 'Chạm trần cả BHTN, năm bậc thuế', salary: 120_000_000 },
  { code: 'S09', title: 'Năm bậc thuế, 3 người phụ thuộc', salary: 150_000_000, deps: 3 },
  { code: 'S10', title: 'Hỗ trợ xăng xe (không tính BH)', salary: 22_000_000, nonInsA: 8_000_000,
    note: 'Khoản 8.000.000 là hỗ trợ xăng xe, không phải phụ cấp lương: không tính BH, vẫn chịu thuế' },
  { code: 'S11', title: 'Phụ cấp trách nhiệm (tính BH), 1 người phụ thuộc', salary: 22_000_000, insA: 3_000_000, deps: 1,
    note: 'Phụ cấp lương có mức cụ thể, trả đều mỗi kỳ: tính vào lương đóng BH' },
  { code: 'C01', title: 'Vào làm giữa kỳ, nghỉ dưới 14 ngày', salary: 20_000_000, paid: 11,
    event: 'Vào làm 16/04/2026. Công hưởng lương 11 = 9 ngày làm + lễ 27/04 (nghỉ bù Giỗ Tổ) và 30/04',
    note: 'Không làm việc 11 ngày < 14 nên đóng BH đủ trên lương hợp đồng. CẦN HỎI cơ quan BHXH quản lý đơn vị: có nơi cho người vào làm sau ngày 15 đóng từ tháng sau' },
  { code: 'C02', title: 'Vào làm giữa kỳ, nghỉ từ 14 ngày', salary: 20_000_000, paid: 7,
    event: 'Vào làm 22/04/2026. Công hưởng lương 7 = 5 ngày làm + 2 ngày lễ',
    note: 'Không làm việc 15 ngày ≥ 14 nên tháng này không đóng BHXH, BHYT, BHTN' },
  { code: 'C03', title: 'Nghỉ việc giữa kỳ, còn phép chưa nghỉ', salary: 18_000_000, paid: 8,
    leave: { days: 4, prev: '03/2026' },
    event: 'Ngày làm cuối 10/04/2026. Công hưởng lương 8. Còn 4 ngày phép năm chưa nghỉ',
    note: 'Không làm việc 14 ngày ≥ 14 nên tháng này không đóng BH; lễ sau ngày nghỉ việc không được hưởng. '
      + 'Phép chưa nghỉ = lương HĐ tháng 03/2026 / 22 công × 4 (Điều 67 NĐ 145/2020); miễn thuế (khoản 8 Điều 4 Luật 109/2025), không tính BH' },
  { code: 'C04', title: 'Đổi lương giữa kỳ bằng phụ lục', salary: 25_000_000,
    segments: [[20_000_000, 11], [25_000_000, 11]],
    event: 'Phụ lục: lương 20.000.000 → 25.000.000 từ 16/04/2026. 01–15/04: 11 công; 16–30/04: 11 công',
    note: 'Lương chia hai đoạn theo công. Lương đóng BH lấy mức hiệu lực cuối tháng (25.000.000)' },
  { code: 'C05', title: 'Nghỉ không lương', salary: 25_000_000, paid: 19, event: 'Nghỉ không lương 3 ngày',
    note: 'Nghỉ 3 ngày < 14 nên đóng BH đủ' },
  { code: 'C06', title: 'Nghỉ ốm (BHXH chi trả)', salary: 25_000_000, paid: 17, event: 'Nghỉ ốm 5 ngày có giấy',
    note: 'Công ty không trả lương ngày ốm; trợ cấp ốm đau do BHXH chi, không qua bảng lương' },
  { code: 'C07', title: 'Nghỉ phép năm', salary: 25_000_000, event: 'Nghỉ phép năm 2 ngày',
    note: 'Phép năm hưởng nguyên lương nên vẫn đủ công' },
  { code: 'C08', title: 'Tăng ca ngày thường', salary: 20_000_000, ot: ['weekday/day', 10],
    event: '10 giờ tăng ca ngày thường, ban ngày', note: 'Tiền tăng ca miễn thuế toàn bộ, không tính BH' },
  { code: 'C09', title: 'Tăng ca ngày nghỉ hằng tuần', salary: 20_000_000, ot: ['weekend/day', 8],
    event: '8 giờ tăng ca chủ nhật' },
  { code: 'C10', title: 'Tăng ca ngày lễ', salary: 20_000_000, ot: ['holiday/day', 8],
    event: '8 giờ tăng ca ngày 30/04/2026', note: '300% chưa gồm lương ngày lễ (đã nằm trong lương tháng)' },
  { code: 'C11', title: 'Tăng ca ban đêm', salary: 20_000_000, ot: ['weekday/night', 4],
    event: '4 giờ tăng ca ngày thường 22h–2h, ban ngày không tăng ca',
    note: '150% + 30% + 20% × 100% = 200% (Điều 57 NĐ 145/2020)' },
  { code: 'C12', title: 'Tăng ca, chạm trần BH, có người phụ thuộc', salary: 60_000_000, deps: 2, ot: ['weekday/day', 12],
    event: '12 giờ tăng ca ngày thường; 2 người phụ thuộc' },
  { code: 'C13', title: 'Truy lĩnh sai sót kỳ đã chốt', salary: 30_000_000, adj: 2_000_000,
    event: 'Kỳ 03/2026 đã chốt bị thiếu 2.000.000 phụ cấp xăng xe (không tính BH)',
    note: 'Trả bù ở kỳ đang mở; chịu thuế ở kỳ chi trả. Không sửa bảng lương 03/2026' },
  { code: 'C14', title: 'Truy thu sai sót kỳ đã chốt', salary: 30_000_000, adj: -1_500_000,
    event: 'Kỳ 03/2026 đã chốt bị trả thừa 1.500.000 phụ cấp xăng xe (không tính BH)',
    note: 'Thu lại ở kỳ đang mở và giảm thu nhập chịu thuế kỳ này; quyết toán năm tự cân. Không sửa bảng lương 03/2026' },
  { code: 'C15', title: 'Kỳ sau khi tăng lương cơ sở', period: '07/2026', salary: 60_000_000,
    event: 'Kỳ 07/2026: lương cơ sở 2.530.000, trần BHXH/BHYT 50.600.000' },
  { code: 'C16', title: 'Tăng ca, có phụ cấp lương', salary: 20_000_000, insA: 3_000_000, ot: ['weekday/day', 10],
    event: 'Phụ cấp trách nhiệm 3.000.000; 10 giờ tăng ca ngày thường',
    note: 'Đơn giá giờ tính trên lương + phụ cấp lương (Điều 55 NĐ 145/2020)' },
  { code: 'C17', title: 'Tăng ca vượt 40 giờ/tháng', salary: 20_000_000, ot: ['weekday/day', 44],
    event: '44 giờ tăng ca ngày thường (11 ngày × 4 giờ)',
    note: '40 giờ đầu miễn thuế, 4 giờ vượt giới hạn chịu thuế. Hệ thống phải cảnh báo khi vượt' },
];

function compute(r) {
  const period = r.period ?? P;
  const { capSI, calcDate } = PERIODS[period];
  const std = workdays(period);
  const insA = r.insA ?? 0, nonInsA = r.nonInsA ?? 0, deps = r.deps ?? 0, adj = r.adj ?? 0;
  const paid = r.segments ? r.segments.reduce((s, [, d]) => s + d, 0) : r.paid ?? std;

  const earned = r.segments
    ? r.segments.reduce((s, [sal, d]) => s + round((sal * d) / std), 0)
    : round(((r.salary + insA + nonInsA) * paid) / std);
  const earnedF = r.segments
    ? r.segments.map(([sal, d]) => `ROUND(${sal}*${d}/I{r},0)`).join('+')
    : 'ROUND((D{r}+E{r}+F{r})*J{r}/I{r},0)';

  // Đơn giá giờ tăng ca gồm phụ cấp lương, không gồm khoản hỗ trợ.
  const [otKind, otHours] = r.ot ?? [null, 0];
  const otRate = otKind ? OT_RATE[otKind] : 0;
  const hourly = (r.salary + insA) / std / HOURS_PER_DAY;
  const ot = round(hourly * otRate * otHours);
  const otExempt = round(hourly * otRate * Math.min(otHours, OT_MONTH_LIMIT));

  const leaveStd = r.leave ? workdays(r.leave.prev) : 0;
  const leavePay = r.leave ? round((r.salary / leaveStd) * r.leave.days) : 0;
  const leaveF = r.leave ? `ROUND(D{r}/${leaveStd}*${r.leave.days},0)` : '0';

  const insured = std - paid < 14;
  const base = insured ? r.salary + insA : 0;
  const bhxh = round(Math.min(base, capSI) * 0.08);
  const bhyt = round(Math.min(base, capSI) * 0.015);
  const bhtn = round(Math.min(base, CAP_UI) * 0.01);
  const ins = bhxh + bhyt + bhtn;
  const er = {
    bhxh: round(Math.min(base, capSI) * EMPLOYER.bhxh),
    bhyt: round(Math.min(base, capSI) * EMPLOYER.bhyt),
    bhtn: round(Math.min(base, CAP_UI) * EMPLOYER.bhtn),
    kpcd: round(Math.min(base, capSI) * EMPLOYER.kpcd),
  };

  const total = earned + ot + leavePay + adj;
  const exempt = otExempt + leavePay;
  const taxableGross = total - exempt;
  const taxable = Math.max(0, taxableGross - ins - PERSONAL - deps * DEPENDENT);
  const tax = pitOf(taxable);
  const net = total - ins - tax;
  const cost = total + er.bhxh + er.bhyt + er.bhtn + er.kpcd;

  // Đối chiếu bảo hiểm và thuế với thue-2026, thuế với pit.
  const opts = { bhxh: insured, bhyt: insured, bhtn: insured };
  const b = thue.calculateNewTax({
    grossIncome: taxableGross, declaredSalary: base, dependents: deps, region: 1, calculationDate: calcDate, insuranceOptions: opts,
  });
  const e = thue.calculateEmployerInsurance(base, 1, opts, true, calcDate);
  check(r.code, {
    bhxh: [bhxh, round(b.insuranceDetail.bhxh)], bhyt: [bhyt, round(b.insuranceDetail.bhyt)],
    bhtn: [bhtn, round(b.insuranceDetail.bhtn)], taxable: [taxable, round(b.taxableIncome)],
    tax: [tax, round(b.taxAmount)], 'tax(pit)': [tax, pit.calcPit(taxable, pitConst.REGIME_2026).total],
    'er.bhxh': [er.bhxh, round(e.bhxh)], 'er.bhyt': [er.bhyt, round(e.bhyt)],
    'er.bhtn': [er.bhtn, round(e.bhtn)], 'er.kpcd': [er.kpcd, round(e.unionFee)],
  });
  if (r.code.startsWith('S')) {
    const a = pit.calcAll({ gross: taxableGross, insuranceBase: base, dependents: deps, region: 'I', regime: pitConst.REGIME_2026 });
    check(r.code, { 'ins(pit)': [ins, a.insurance.total], 'net(pit)': [net, a.net] });
  }
  if (otKind) {
    const [type, shift] = otKind.split('/');
    const o = thueOt.calculateOvertime({
      monthlySalary: r.salary + insA, workingDaysPerMonth: std, hoursPerDay: HOURS_PER_DAY, includeHolidayBasePay: false,
      entries: [{ id: '1', type, shift, hours: otHours }], dependents: 0, otherDeductions: 0, hasInsurance: true,
      insuranceOptions: { bhxh: true, bhyt: true, bhtn: true }, region: 1, useNewLaw: true,
    });
    check(r.code, {
      'ot(thue-2026)': [ot, round(o.breakdowns[0].grossAmount)],
      'ot taxable(thue-2026)': [ot - otExempt, round(o.breakdowns[0].taxableAmount)],
    });
  }

  const sources = ['thuế, BH người lao động và doanh nghiệp: khớp thue-2026; thuế khớp pit'];
  if (otKind) sources.push('tăng ca: khớp thue-2026');
  if (r.segments || r.paid != null || r.leave) sources.push('chia lương theo công, phép chưa nghỉ: tự tính');
  return {
    period, std, paid, earned, earnedF, otHours, otRate, ot, exempt, leavePay, leaveF, adj, total, base, bhxh, bhyt, bhtn,
    personal: PERSONAL, dependent: deps * DEPENDENT, taxable, tax, net, er, cost, deps, insA, nonInsA, sources: sources.join('; '),
  };
}

function check(code, pairs) {
  for (const [k, [x, y]] of Object.entries(pairs)) {
    if (x !== y) throw new Error(`${code} mismatch ${k}: ours=${x} other=${y}`);
  }
}

// ---- Ghi file ----

const wb = new ExcelJS.Workbook();
const money = '#,##0';
const ws = wb.addWorksheet('Bộ mẫu', { views: [{ state: 'frozen', xSplit: 2, ySplit: 1 }] });
ws.columns = [
  ['Mã', 6], ['Tình huống', 34], ['Kỳ', 9], ['Lương hợp đồng', 14], ['Phụ cấp lương (tính BH)', 12],
  ['Khoản hỗ trợ (không tính BH, chịu thuế)', 12], ['Sự kiện trong kỳ', 44], ['Số NPT', 7], ['Công chuẩn', 8],
  ['Công hưởng lương', 9], ['Lương theo công', 14], ['Giờ tăng ca', 8], ['Hệ số tăng ca', 8], ['Tiền tăng ca', 13],
  ['Phép chưa nghỉ được trả', 13], ['Truy lĩnh (+) / truy thu (−)', 13], ['Tổng thu nhập', 14], ['Thu nhập miễn thuế', 13],
  ['Lương đóng BH', 14], ['BHXH 8%', 11], ['BHYT 1,5%', 11], ['BHTN 1%', 11], ['Giảm trừ bản thân', 13],
  ['Giảm trừ NPT', 13], ['Thu nhập tính thuế', 14], ['Thuế TNCN', 12], ['Thực lĩnh', 14],
  ['DN: BHXH 17,5%', 12], ['DN: BHYT 3%', 11], ['DN: BHTN 1%', 11], ['DN: KPCĐ 2%', 11], ['Tổng chi phí lương của DN', 15],
  ['Giả định riêng của dòng', 48], ['Nguồn số liệu', 40],
].map(([header, width]) => ({ header, width }));
ws.getRow(1).font = { bold: true };
ws.getRow(1).alignment = { wrapText: true, vertical: 'top' };

for (const r of rows) {
  const c = compute(r);
  const n = ws.rowCount + 1;
  const f = (formula, result) => ({ formula: formula.replaceAll('{r}', n), result });
  ws.addRow([
    r.code, r.title, c.period, r.salary, c.insA, c.nonInsA, r.event ?? 'Đủ công cả tháng', c.deps, c.std, c.paid,
    f(c.earnedF, c.earned), c.otHours, c.otRate, f('ROUND((D{r}+E{r})/I{r}/8*M{r}*L{r},0)', c.ot), f(c.leaveF, c.leavePay), c.adj,
    f('K{r}+N{r}+O{r}+P{r}', c.total), f(`ROUND((D{r}+E{r})/I{r}/8*M{r}*MIN(L{r},${OT_MONTH_LIMIT}),0)+O{r}`, c.exempt),
    c.base, c.bhxh, c.bhyt, c.bhtn, c.personal, c.dependent, c.taxable, c.tax, f('Q{r}-T{r}-U{r}-V{r}-Z{r}', c.net),
    c.er.bhxh, c.er.bhyt, c.er.bhtn, c.er.kpcd, f('Q{r}+AB{r}+AC{r}+AD{r}+AE{r}', c.cost), r.note ?? '', c.sources,
  ]);
}
const textCols = [7, 8, 9, 10, 12, 13, 33, 34];
for (let col = 4; col <= 32; col++) if (!textCols.includes(col)) ws.getColumn(col).numFmt = money;
for (const col of [7, 33]) ws.getColumn(col).alignment = { wrapText: true, vertical: 'top' };

const g = wb.addWorksheet('Giả định');
g.columns = [{ header: '#', width: 4 }, { header: 'Điểm cần chốt', width: 40 }, { header: 'Giả định của bộ mẫu', width: 70 }, { header: 'Căn cứ', width: 50 }];
g.getRow(1).font = { bold: true };
[
  ['Lịch làm việc', 'Thứ Hai–thứ Sáu, 8 giờ/ngày; nghỉ thứ Bảy, Chủ nhật. Pháp nhân ở vùng I.', 'Giả định của công ty mẫu'],
  ['Công chuẩn', 'Số ngày thứ Hai–thứ Sáu của tháng, kể cả ngày lễ rơi vào ngày làm việc (03/2026 và 04/2026: 22; 07/2026: 23).', 'Điều 54 NĐ 145/2020: lương ngày = lương tháng / số ngày làm việc bình thường do doanh nghiệp chọn, tối đa 26'],
  ['Ngày công hưởng lương', 'Ngày làm việc + ngày lễ, nghỉ bù + phép năm. Ngày lễ trước ngày vào làm hoặc sau ngày nghỉ việc không tính.', 'Điều 112, 113 BLLĐ 2019'],
  ['Chia lương', 'Lương theo công = ROUND((lương + phụ cấp lương + khoản hỗ trợ) × công hưởng lương / công chuẩn). Đổi lương giữa kỳ: chia từng đoạn theo công rồi cộng.', 'Thông lệ; NĐ 145/2020 Điều 54'],
  ['Phụ cấp lương và khoản hỗ trợ', 'Phụ cấp lương (chức vụ, trách nhiệm, độc hại…) và khoản bổ sung có mức cụ thể, trả đều mỗi kỳ: tính BH. Khoản hỗ trợ (xăng xe, điện thoại, ăn ca, nhà ở…): không tính BH. Khoản hỗ trợ trong mẫu đều chịu thuế.', 'Luật BHXH 2024'],
  ['Đơn giá giờ tăng ca', '(Lương hợp đồng + phụ cấp lương) / công chuẩn / 8. Không gồm khoản hỗ trợ.', 'Điều 55 NĐ 145/2020: tiền lương thực trả theo công việc đang làm'],
  ['Hệ số tăng ca', 'Ngày thường 150%, nghỉ tuần 200%, lễ 300% (chưa gồm lương ngày lễ), đêm ngày thường không tăng ca ban ngày 200%.', 'Điều 98 BLLĐ 2019; Điều 55, 57 NĐ 145/2020'],
  ['Thuế tiền tăng ca', 'Miễn toàn bộ phần trong giới hạn: 40 giờ/tháng và 200 giờ/năm (300 giờ với ngành được phép). Phần vượt chịu thuế toàn bộ. Hệ thống phải cộng dồn giờ trong năm và cảnh báo khi vượt; bộ mẫu chỉ có trường hợp vượt giới hạn tháng.', 'Luật Thuế TNCN 109/2025/QH15 khoản 8 Điều 4; Điều 107 BLLĐ 2019; Điều 60 NĐ 145/2020'],
  ['Phép năm chưa nghỉ khi thôi việc', 'Lương HĐ của tháng liền trước tháng thôi việc / số ngày làm việc bình thường của tháng đó × số ngày chưa nghỉ. Miễn thuế, không tính BH.', 'Khoản 3 Điều 113 BLLĐ 2019; Điều 67 NĐ 145/2020; khoản 8 Điều 4 Luật 109/2025 (Bộ Tài chính trả lời: gồm cả phép chưa nghỉ khi thôi việc)'],
  ['Lương đóng BH', 'Lương hợp đồng + phụ cấp lương, không chia theo công. Đổi lương giữa tháng: lấy mức hiệu lực cuối tháng.', 'Luật BHXH 2024; mức cuối tháng là giả định'],
  ['Tháng nghỉ nhiều', 'Không làm việc và không hưởng lương từ 14 ngày làm việc trở lên (gồm ngày trước khi vào làm, sau khi nghỉ việc, nghỉ ốm): không đóng BHXH, BHYT, BHTN, KPCĐ tháng đó.', 'Luật BHXH 2024. Tính ngày trước khi vào làm là giả định: CẦN HỎI cơ quan BHXH quản lý đơn vị'],
  ['Phần doanh nghiệp đóng', 'BHXH 17,5% (gồm TNLĐ-BNN 0,5%), BHYT 3%, KPCĐ 2% trên lương đóng BH, trần 20 × lương cơ sở; BHTN 1%, trần 20 × lương tối thiểu vùng.', 'Luật BHXH 2024; Luật Việc làm 2025; Luật Công đoàn 2024 Điều 29'],
  ['Nghỉ ốm', 'Công ty không trả lương; BHXH chi trợ cấp 75% ngoài bảng lương.', 'Điều 45 Luật BHXH 2024'],
  ['Truy lĩnh, truy thu cùng năm', 'Lập ở kỳ đang mở như một khoản cộng/trừ, chịu thuế ở kỳ chi trả; không sửa kỳ đã chốt. Khoản điều chỉnh trong mẫu không tính BH.', 'Thời điểm xác định thu nhập chịu thuế là lúc trả; phần trừ vào kỳ sau là giả định'],
  ['Truy thu khác năm (CHƯA CHỐT)', 'Ví dụ 01/2027 thu lại khoản trả thừa của 12/2026: trừ vào kỳ hiện tại thì quyết toán năm 2026 không tự cân. Cần quy tắc riêng; chưa có dòng mẫu.', 'Cần tìm hướng dẫn của cơ quan thuế'],
  ['Khấu trừ thuế', 'Mọi người là cá nhân cư trú, HĐ từ 3 tháng, khấu trừ theo biểu lũy tiến từng tháng.', 'Luật Thuế TNCN 109/2025/QH15'],
  ['Làm tròn', 'Từng khoản của từng người làm tròn tới đồng, nửa xa số 0 (chế độ line). Thực lĩnh không làm tròn thêm.', 'ADR-0006'],
  ['Chưa có trong mẫu', 'Đoàn phí công đoàn của người lao động, khoản hỗ trợ miễn thuế (ăn ca…), người không cư trú, HĐ dưới 3 tháng, truy thu khác năm, vượt giới hạn tăng ca năm.', 'Bổ sung khi cần'],
].forEach((r, i) => g.addRow([i + 1, ...r]));
for (const col of [2, 3, 4]) g.getColumn(col).alignment = { wrapText: true, vertical: 'top' };

const p = wb.addWorksheet('Tham số');
p.columns = [{ header: 'Tham số', width: 45 }, { header: 'Giá trị', width: 22 }, { header: 'Hiệu lực', width: 22 }, { header: 'Căn cứ', width: 60 }];
p.getRow(1).font = { bold: true };
[
  ['Lương tối thiểu vùng I', 5_310_000, 'từ 01/01/2026', 'Nghị định 293/2025/NĐ-CP'],
  ['Lương cơ sở', 2_340_000, 'tới 30/06/2026', 'Nghị định 73/2024/NĐ-CP'],
  ['Lương cơ sở', 2_530_000, 'từ 01/07/2026', 'Nghị định 161/2026/NĐ-CP'],
  ['Trần lương đóng BHXH, BHYT (20 × lương cơ sở)', 46_800_000, 'tới 30/06/2026', 'Luật BHXH 2024'],
  ['Trần lương đóng BHXH, BHYT', 50_600_000, 'từ 01/07/2026', 'Luật BHXH 2024'],
  ['Trần lương đóng BHTN vùng I (20 × lương tối thiểu vùng)', 106_200_000, 'từ 01/01/2026', 'Luật Việc làm 2025'],
  ['Người lao động: BHXH / BHYT / BHTN', '8% / 1,5% / 1%', '', 'Luật BHXH 2024, Luật BHYT, Luật Việc làm 2025'],
  ['Giảm trừ bản thân', 15_500_000, 'kỳ tính thuế 2026', 'Nghị quyết 110/2025/UBTVQH15'],
  ['Giảm trừ mỗi người phụ thuộc', 6_200_000, 'kỳ tính thuế 2026', 'Nghị quyết 110/2025/UBTVQH15'],
  ['Biểu thuế (tháng)', '≤10tr 5%; ≤30tr 10%; ≤60tr 20%; ≤100tr 30%; >100tr 35%', 'kỳ tính thuế 2026', 'Luật Thuế TNCN 109/2025/QH15'],
  ['Ngày lễ 04/2026', '27/04 (nghỉ bù Giỗ Tổ 26/04), 30/04', '', 'Lịch nghỉ lễ 2026 của Chính phủ'],
].forEach((r) => p.addRow(r));
p.getColumn(2).numFmt = money;


const s = wb.addWorksheet('Nguồn');
s.columns = [{ header: 'Mục', width: 30 }, { header: 'Chi tiết', width: 110 }];
s.getRow(1).font = { bold: true };
[
  ['Người lập', 'Claude (vai kế toán), từ quy định công khai. Đã qua một lượt soát của người dùng; CHƯA có kế toán thật duyệt.'],
  ['Đối chiếu', 'Mọi dòng: BH người lao động và doanh nghiệp, thu nhập tính thuế và thuế khớp từng đồng với thue-2026; thuế khớp với pit. Dòng S: BH và thực lĩnh khớp thêm với pit. Dòng tăng ca: tiền tăng ca và phần chịu thuế khớp với bộ tính tăng ca của thue-2026. Chia lương theo công, phép chưa nghỉ và các giả định ở sheet "Giả định" không có nguồn đối chiếu.'],
  ['Công cụ 1', 'https://github.com/thangtd-0050/pit @ 2844c5c'],
  ['Công cụ 2', 'https://github.com/googlesky/thue-2026 @ 2e67d2e'],
  ['Giới hạn', 'Hai công cụ là dự án cá nhân, không phải nguồn chính thức. Kế toán cần duyệt bộ mẫu trước khi dùng dữ liệu thật của khách.'],
].forEach((r) => s.addRow(r));
s.getColumn(2).alignment = { wrapText: true, vertical: 'top' };

const file = join(here, 'payroll-input-samples.xlsx');
await wb.xlsx.writeFile(file);
console.log(`ok: ${rows.length} rows cross-checked -> ${file}`);
