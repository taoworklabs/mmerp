# HRM menu in HR's words, and a personal space for every employee

Date: 2026-10-27 · Status: draft, awaiting approval

## Intent

"Finishing HRM" means one goal: **HR closes a month without opening Excel.** We checked HRM against an SMB HR person's month, Odoo, Frappe HR and the Vietnamese competitor MISA AMIS. Payroll calculation is already done. The gaps are what HR does every month for employees and for the state. The work is split into four parts, each with its own design and PR, in this order:

1. **This document:** the HRM menu renamed and regrouped in HR's own words, and a personal space ("Cá nhân") where every employee sees their timesheet days and payslips.
2. The first day on the software: a company-wide leave balance page ("Quỹ phép") and importing employee profiles from Excel.
3. Insurance and tax: the monthly social insurance join/leave list (báo tăng/giảm BHXH, form D02-LT under Decision 366/QĐ-BHXH of 2026), PIT returns (05/KK-TNCN) and the yearly PIT finalisation (05/QTT-TNCN). Read the current forms before designing it.
4. Salary bank transfer files.

Why part 1 comes first: employees seeing their own payslip is the most certain need on the list (MISA and Odoo both have it, and HRM does not). And today an employee with no HR role gets **no tile at all** on the home page (`HomePage` requires a product permission), so self-service filing has no front door.

The product owner is not an HR practitioner, and nobody on the team is. Menu names below are our best reading of what Vietnamese HR people say. They are not checked with a real HR user.

## Goals

### Principles

- **A menu name is what HR says out loud**, not a developer's word ("Tham số pháp lý" said nothing to the product owner).
- **The menu lists things; actions live on a record.** Terminating an employee, for example, is an action on the profile and never a menu item.
- **Employees get their own space**, designed around what they come to do. It is not a trimmed copy of the HR menu.
- Inside the personal space, names carry no "của tôi" suffix: the tile already says whose it is.

### The "Nhân sự" area (HR, payroll accountant, administrators)

The home tile is renamed from "Quản lý nhân sự" to **"Nhân sự"**. The menu becomes the following (Vietnamese labels are final; the English ones are the builder's call, keeping CONTEXT.md terms):

| Group | Item | Was |
| --- | --- | --- |
| (none) | Tổng quan | unchanged name, new content (below) |
| Nhân sự | Hồ sơ nhân viên | Nhân viên |
| | Hợp đồng lao động | Hợp đồng |
| | Sơ đồ tổ chức | Cơ cấu tổ chức (also in the personal space) |
| Chấm công & nghỉ phép | Bảng chấm công | Bảng công |
| | Đơn nghỉ phép | Đơn nghỉ |
| | Đơn tăng ca | Tăng ca |
| Tiền lương | Bảng lương | unchanged |
| Thiết lập (collapsible, folded by default) | Loại hợp đồng | unchanged |
| | Loại nghỉ phép | Loại nghỉ |
| | Lịch làm việc & ngày lễ | Lịch làm việc |
| | Mức đóng bảo hiểm & thuế | Tham số pháp lý |
| | Quy trình duyệt | Quy tắc duyệt |

- Group "Cấu hình" is renamed "Thiết lập".
- Every Vietnamese label a user reads for these things follows the new names: menu, page titles, breadcrumbs, record-type names (approval inbox, notifications, audit), buttons. E2E selectors are updated to match *(my call)*. English labels follow the same rule where they change.
- Leave and overtime list items stay in this menu for HR. People without HR permissions see no HRM tile and use the personal space instead.
- Items for parts 2 to 4 ("Quỹ phép", a "Bảo hiểm & thuế" group, "Chuyển khoản lương") are **not** added now.

**Tổng quan** keeps its counts and gains a "cần xử lý" block. Each entry is a count that links to the matching filtered list, and it is shown only to users who may open that list:

- leave and overtime requests waiting for **this user's** approval (they are an approver of the pending step), linking to the core approval inbox *(my call: actionable beats a scope-wide count)*;
- contracts expiring within 30 days (the contract list already has this filter);
- org units with no closed timesheet for the previous month;
- the previous month with no closed payroll for a legal entity.

An entry with a count of zero is hidden *(my call)*. Which counts the existing overview keeps is the builder's call.

### The "Cá nhân" area (every user linked to an employee profile)

A new home tile, **"Cá nhân"**. It is shown to every signed-in user whose account is linked to an employee profile, with no role needed. HR staff who are also employees see both tiles. Accounts not linked to a profile (e.g. `root`) do not see it. A terminated employee whose account is still linked keeps the tile and their old payslips; cutting access is done by locking the account in `iam`, as for anyone leaving *(my call)*. Core's home page must not learn HRM data: the tile decides its own visibility through the HRM client (the existing self-service endpoint already returns the actor's employee id), e.g. an optional visibility hook on the manifest *(my call; mechanism is the builder's)*. Like any HRM screen, it stays readable when the HRM product is disabled, and filing is refused through the usual write gate.

Menu (no groups):

1. **Chấm công.** A calendar of the employee's own days, with a Month / Week switch (weeks run Monday to Sunday and may cross months). Each day shows the days worked (1 or ½), leave (with its type), public holidays and weekly rest days, and overtime hours. A summary shows days worked, paid leave days, overtime hours and the month's standard workdays.
   - Days from a timesheet that is not closed yet are shown with a **"chưa chốt"** (not closed) label, so the employee can spot mistakes while HR can still fix them *(my call)*.
   - If the employee's unit has no timesheet for the month yet, the calendar still shows approved leave, approved overtime, holidays and rest days.
2. **Phiếu lương.** A list of months, newest first, each with the net pay. Opening a month shows the payslip: pay by workdays, allowances and benefits, overtime, BHXH/BHYT/BHTN deductions, PIT, adjustments and net pay. There is also a PDF download with the same layout HR prints today. The PDF is cut from the payslip print that `printing` froze when the payroll was posted (the employee's own part only), not re-rendered from live lines, so it can never drift from HR's copy *(my call)*. This needs an "own part" access path in core `printing`, whose check today requires View and Print on the whole payroll; write an ADR for it before the code.
   - A month appears only once its payroll is **posted** (closed), so an employee never sees a figure that later changes *(my call)*. If that payroll is cancelled, the month disappears until a new one is posted.
3. **Nghỉ phép & tăng ca.** The employee's own leave and overtime requests, the leave days left this year per leave type, and buttons to file a new request. Filing reuses today's forms and flow.
4. **Hồ sơ cá nhân.** The employee's own profile, read-only, ordinary fields only (name, date of birth, contact, department, manager, hire date, current contract type and term). Sensitive fields (ID number, tax code, bank account, salary) are **not** shown here *(my call: showing them would weaken the sensitive-field rule, and part 1 doesn't need them)*.
5. **Sơ đồ tổ chức.** The same page HRM has today, which every signed-in user may already read.

### Permissions and audit

- An employee reads **their own** timesheet days, requests, balance and payslips with no role. This is new for payslips: today every per-person amount needs `hrm.salary.view`. The new rule is narrow: the actor's own employee record only, posted payrolls only.
- Every payslip read and PDF download by the employee is audited, like any sensitive read.
- "Own" means the employee profile linked to the signed-in account. Nothing in the personal space takes an employee id from the client.

## Non-goals

- Self clock-in/out: time clocks stay out of scope; employees **view** days that HR enters or imports.
- Editing one's own profile, or requests to change it.
- Reporting a payslip error from the payslip (MISA has this). Employees ask HR as they do today.
- Notifications when a payslip is published.
- A mobile app. The personal space is the same web app and must work on a phone-sized screen.
- A manager's view of their team in the personal space.
- Anything from parts 2 to 4.

## Anti-goals

- The personal space must never show one employee anything about another employee (amounts, days, requests), whatever the URL or API call.
- A payslip the employee sees must match, figure for figure, the payslip HR prints for the same posted payroll.
- The rename must not change English domain terms or API names. It is a label change.

## Constraints

- Areas never import each other. The personal space belongs to the HRM frontend area and calls only the HRM client. How an area exposes a second home tile (a second manifest from `web/src/hrm`, or a manifest option) is the builder's call. Since the area manifest is "the only thing app knows about an area", check whether that needs an ADR.
- Every screen follows `docs/ui.md` and the `mmerp-ui` skill. Page layouts come from `shared/ui/page` templates.
- Payroll lines are encrypted: the payslip is decrypted in the service per request, never stored decrypted.
- All user-facing text goes in i18n files, `vi` and `en`.

## Left to the builder

- API shape of the new "own" endpoints (calendar data, payslip list, payslip detail and PDF, own requests and balance), and whether existing list endpoints with a self filter serve the requests page.
- The personal space's base path and the English labels.
- How the calendar renders on narrow screens.
- Test split. At minimum, a backend test that one employee cannot read another's payslip or days through any new endpoint, and a Playwright flow: an employee with no role signs in, sees only the "Cá nhân" tile, and opens a posted payslip.
