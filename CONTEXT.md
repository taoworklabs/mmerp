# Glossary

A term used by two products must have the same meaning, or the two must use different names. Each product adds its own section below the Core section.

## Core

- **Tenant** (vi: "Tenant"): an organisation (company) using the system. Each tenant has its own database.
- **User** (`iam`, vi: "Người dùng"): a sign-in account. Not synonymous with any person record in the business domain.
- **Session** (vi: "Phiên"): a still-valid sign-in of a user, stored in `iam.sessions`. Signing out or revoking ends it immediately.
- **Actor** (vi: "Người thực hiện"): the user performing an operation, carried in the request or job context; the person recorded in the audit log.
- **Authorization version** (`authz_version`, vi: "Phiên bản phân quyền"): a string that changes whenever a user's permissions change; the frontend compares it with the `X-Authz-Version` header to clear stale data.
- **Org unit** (vi: "Đơn vị tổ chức"): a node in the tenant's org tree.
- **Legal entity** (vi: "Pháp nhân"): an org unit with `kind = company`, carrying a tax code.
- **Org unit kind** (`kind`, vi: "Loại đơn vị tổ chức"): `group` (group, the root of a corporate group, above the legal entities), `company` (legal entity), `branch` (branch), `department` (department). Branches and departments always sit under exactly one legal entity.
- **Role** (vi: "Vai trò"): a set of permissions declared in code by a product (`hrm.hr`), granted to a user at an org unit or tenant-wide. A grant at a unit applies to its whole subtree. Some roles can only be granted tenant-wide (e.g. `hrm.approval_admin`).
- **Permission** (vi: "Quyền"): `<product>.<object>.<action>`, e.g. `hrm.employee.view`. Users get permissions through roles; individual permissions are never granted.
- **Scope** (`Scope`, vi: "Phạm vi"): the org units where a user has a permission, expanded to their subtrees, or the tenant-wide flag.
- **Sensitive field** (vi: "Trường nhạy cảm"): a field encrypted in the database; viewing it needs a dedicated permission and every view is audited.
- **Product** (`product` in code, vi: "Phân hệ"): a bundle of modules that can be enabled or disabled. `product = 'core'` marks tenant-level roles and has nothing to do with the `core/` directory.
- **Enabled products** (vi: "Phân hệ được bật"): the list of products the tenant may use. Read from configuration (`PRODUCTS`).
- **Locale** (`locale`, vi: "Ngôn ngữ"): `vi` or `en`, per user. Data entered by users is not translated.
- **Error code** (vi: "Mã lỗi"): a stable string returned by the API with parameters (e.g. `period_locked`); the frontend translates it. Part of the API interface.
- **Operation class** (vi: "Lớp hành động"): read, export or write. A product that is not enabled, or has been disabled, can still be read and exported, but not written.
- **Module**: a package in `core/`, `shared/` or `modules/`, with its own schema and tables.
- **Area** (frontend, vi: "Khu vực"): a product's part of the UI, the directory `web/src/<product>`, entered through its `index.ts` manifest.
- **Record type** (vi: "Loại bản ghi"): `<module>.<type>`, either a document or a catalog, registered with `record` to use the shared features.
- **Document** (vi: "Chứng từ"): a record with a number, a date, a legal entity and a lifecycle held by `record`.
- **Catalog** (`record.Catalog`, vi: "Danh mục"): a record type that is master data, with no lifecycle (e.g. the employee profile).
- **Document status** (vi: "Trạng thái chứng từ"): `draft`, `pending_approval`, `posted`, `cancelled`, held by `record`. Says only whether the document is in effect and whether it can be edited.
- **Progress status** (vi: "Trạng thái thực hiện"): the business progress of a document (delivery, payment…), held by the module that owns the document, under its own name. Does not decide whether the document can be edited.
- **Version** (`version`, vi: "Phiên bản"): a number incremented after every write to a document, used to detect conflicts when two people edit at once.
- **Dependency**: something a module requires from another product (`deps.go`). Never a no-op.
- **Hook**: an optional reaction of another module (`hooks.go`). May be a no-op.
- **Approval field** (vi: "Trường duyệt"): a field a document type declares for use in approval conditions (e.g. `days`, `leave_type`).
- **Approval instance** (vi: "Lần duyệt"): one submission of a document for approval, with its own id and submitted `version`. Withdrawal or rejection closes the instance; resubmitting creates a new one.
- **System job** (vi: "Job hệ thống"): a job type listed explicitly in code, run as the system actor. **User job** (vi: "Job thay người dùng"): every other job; runs with the requester's permissions, re-checked when it executes.
- **My background jobs** (vi: "Việc chạy nền của tôi"): the list of user jobs the user requested, with status, error rows and result file. Each user sees only their own jobs. Administrators can also see system jobs, without the job parameters.
- **Import** (vi: "Lần nhập"): one load of an Excel file into a data type (`hrm.timesheet`, `hrm.leave_balance`), run as a user job. All or nothing: if one row fails, no row is written, and the import reports each failing row with its reason.
- **Export** (vi: "Lần xuất"): one write of data to an Excel file, run as a user job and audited. The result is a **temporary export file** (vi: "File xuất tạm"): only the requester can download it, and it expires after 24 hours.
- **Period lock** (vi: "Khóa kỳ"): the closing date of a legal entity. Documents dated on or before the lock date cannot be created, edited or cancelled.
- **Document number** (vi: "Số chứng từ"): a number issued by `numbering` when a draft is created, per type × legal entity × year, in the form `NP-2026-00001`. Deleting a draft leaves a gap.
- **Approval rule** (vi: "Quy tắc duyệt"): the chain of approval steps for a document type; each step has an optional condition and an approver (by role, a specific user, or by module), plus a fallback role.
- **Approval inbox** (inbox, vi: "Hộp duyệt"): the list of approval instances waiting for the user to approve at the current step.
- **Approval gate** (vi: "Cổng phê duyệt"): an interface defined by `record` to ask whether a status transition needs approval; `approval` implements it.
- **Business line** (vi: "Dòng nghiệp vụ"): a line describing the economic meaning of a document (line type, amount, party, org unit), without an accounting account. Written to `posting.lines` exactly once when the document is posted.
- **Legal-entity setting** (vi: "Setting theo pháp nhân"): a setting with a separate value per legal entity (`setting.legal_entity_values`), e.g. the rounding mode or the minimum wage region. A missing value falls back to the default in code.
- **Rounding mode** (`setting.rounding`, vi: "Chế độ làm tròn"): `line` (each person × each item rounds on its own) or `total` (round the total of an item in the document, then distribute the difference down to the lines by largest remainder).
- **Attachment** (`attachment`, vi: "Đính kèm"): a file attached to a record (document or catalog). Viewing needs `view_files`, adding needs `attach`; the record type answers both in `Can`. Not available on record types that do not answer. See [ADR-0024](./docs/adr/0024-attachments-and-discussion.md).
- **Discussion** (`discussion`, vi: "Trao đổi"): a record's comment thread, in time order. Anyone who can view the record can read and comment.
- **Comment** (vi: "Bình luận"): one message in a discussion; plain text, never edited or deleted.
- **Mention** (vi: "Nhắc tên"): `@login` in a comment. The mentioned user gets a notification if they can view the record; otherwise the mention is ignored, without telling the author.
- **Notification** (`notification`, vi: "Thông báo"): a line telling a user about an event that concerns them (a document waiting for their approval, their document approved or rejected, being mentioned, their job finished or failed). Carries no record content; opening it re-checks view permission. May come with an email. See [ADR-0025](./docs/adr/0025-notifications.md).
- **Mail server** (vi: "Máy chủ mail"): the tenant's SMTP configuration. Without it no email is sent; in-app notifications still work.
- **Journal entry mapping** (vi: "Định khoản"): `accounting` turning business lines into debit/credit journal entries by configurable rules.

## HRM

Product description: [products/hrm.md](./products/hrm.md).

- **Employee** (`hrm.employee`, vi: "Nhân viên"): a worker's profile. Not synonymous with User; an employee may have no sign-in account.
- **Employee code** (`code`, vi: "Mã nhân viên"): a code set by HR, unique within the tenant, case-insensitive.
- **HRM roles** (vi: "Vai trò HRM"): `hr` (view and edit employee profiles and leave requests), `viewer` (view profiles only), `sensitive_viewer` (permission `hrm.employee.sensitive`: view and edit sensitive fields and dependants), `leave_admin` (grant and adjust leave balances and manage leave types), `payroll_viewer` (view payroll totals by department, without anyone's amounts), `payroll` (view, create, compute and finalize payrolls; `hrm.salary.view`: view the money items of contracts and payrolls; edit the work calendar; edit legal parameters when granted tenant-wide), `approval_admin` (`hrm.approval.manage`: configure HRM approval rules; can only be granted tenant-wide). `hr` also views and edits contracts, overtime requests and timesheets and manages contract types; entering contract amounts also needs `payroll`. See [ADR-0013](./docs/adr/0013-employee-profile.md), [ADR-0016](./docs/adr/0016-leave-requests.md), [ADR-0017](./docs/adr/0017-contracts-overtime-payroll-lock.md).
- **Employee status** (vi: "Trạng thái nhân viên"): `active` (working) or `terminated` (left, once the termination date has passed in the tenant time zone).
- **Dependant** (vi: "Người phụ thuộc"): a person counted for an employee's family deduction (giảm trừ gia cảnh), from month `deduction_from` to month `deduction_to` (`YYYY-MM`; without `deduction_from` the deduction has not started). All their data are sensitive fields.
- **Employment contract** (`hrm.contract`, vi: "Hợp đồng lao động"): a document; `posted` means in effect. Changed through an **amendment**, which is also a contract pointing to the base contract.
- **Base contract** (vi: "Hợp đồng gốc"): a contract that points to no other contract; it carries the term. Two base contracts in effect for the same employee do not have overlapping terms.
- **Amendment** (vi: "Phụ lục"): a contract pointing to a base contract, with an effective date within the base contract's term, carrying all money items in place of the base contract from that date. Does not change the term.
- **Contract type** (vi: "Loại hợp đồng"): a catalog entered by the organisation (fixed-term, probation…); the approval field `contract_kind`. An amendment takes the type of its base contract. A **fixed-term** type (vi: "có thời hạn") requires the contract to have an end date; an indefinite type has none.
- **Expiring contract** (vi: "Hợp đồng sắp hết hạn"): a `posted` base contract whose end date falls within the next 30 days, today included.
- **Contract in effect on a date** (vi: "Hợp đồng có hiệu lực tại một ngày"): a `posted` base contract whose term covers that date, with the money items of the latest `posted` amendment whose effective date is not after that date.
- **Overtime day kind** (`day_kind`, vi: "Loại ngày tăng ca"): `weekday` (normal day), `weekly_off` (weekly day off), `holiday` (public holiday); decides the overtime multiplier. Still entered by hand; deriving it from the work calendar comes later.
- **Affected date range** (vi: "Khoảng ngày bị ảnh hưởng"): the dates on which a source document changes pay; used to check against finalized pay periods.
- **Leave request** (vi: "Đơn nghỉ phép"), **overtime request** (vi: "Đơn tăng ca"): documents; `posted` means approved.
- **Leave balance** (vi: "Quỹ phép"): an employee's remaining leave days in the year.
- **Leave type** (vi: "Loại nghỉ"): a catalog entered by the organisation (annual leave, sick leave…); only types with the deduct-balance flag (`deducts_balance`) deduct from the leave balance when a request is approved.
- **Paid by the company** (`paid`, vi: "Công ty trả lương"): a leave-type flag; leave days of a flagged type are paid days (annual leave yes; sick leave and unpaid leave no).
- **Leave balance grant or adjustment** (vi: "Cấp, điều chỉnh quỹ phép"): adding or subtracting days to an employee's leave balance for a year, with a mandatory reason; needs the `leave_admin` role.
- **Timesheet** (`hrm.timesheet`, vi: "Bảng công"): attendance for a pay period of an org unit. `posted` means attendance is finalized. Each unit has at most one non-cancelled timesheet per period.
- **Timesheet line** (vi: "Dòng bảng công"): the days actually worked (0.5 or 1) by an employee on one day of the timesheet. An empty cell means not worked. By convention no attendance is recorded on leave days and holidays, but such lines are still accepted; payroll ignores lines falling on a holiday or weekly day off, and worked days plus paid leave days count at most 1 per day.
- **Payroll** (`hrm.payroll`, vi: "Bảng lương"): pay for a pay period of a legal entity. `posted` means pay is finalized. Stores all inputs used for the computation.
- **Pay period** (vi: "Kỳ lương"): the date range `period_start`–`period_end`. The first version is always a calendar month.
- **Direct manager** (vi: "Quản lý trực tiếp"): the managing employee (`manager_id`), an HRM business relationship, independent of the permission-scope tree.
- **Finalized pay period** (vi: "Kỳ lương đã chốt"): a pay period of a legal entity that has exactly one `posted` payroll (`posted_payroll_id`). Source documents whose affected date range overlaps that period can no longer change until the payroll is cancelled.
- **Legal parameters** (vi: "Tham số pháp lý"): insurance rates, ceiling multipliers, base salary (lương cơ sở), regional minimum wage, family deductions, personal income tax (TNCN) brackets, the tax-exempt overtime-hour limit, with effective dates (`hrm.legal_params`). Never hard-coded. A payroll uses the version in effect on the last day of the period.
- **Minimum wage region** (`hrm.wage_region`, vi: "Vùng lương tối thiểu"): region I–IV of a legal entity, a legal-entity setting; decides the unemployment insurance (BHTN) ceiling.
- **Work calendar** (vi: "Lịch làm việc"): the weekly days off (per **work-week version** (vi: "Phiên bản tuần làm việc") with an effective date, `hrm.work_weeks`; Saturday and Sunday by default) and the list of holidays, including compensatory days off, of a legal entity, entered by HR. The basis for standard working days and paid days, and for shading days off in the timesheet grid. The day count of a leave request (in half days, only checked not to exceed the calendar days of the leave range) and the day kind of an overtime request are still entered by hand; deriving them from the calendar comes later.
- **Standard working days** (vi: "Công chuẩn"): the number of normal working days in the month per the legal entity's work calendar, including holidays falling on working days. The denominator when prorating pay by attendance.
- **Paid days** (vi: "Công hưởng lương"): the days paid in the month: working days, holidays, compensatory days off and leave days of a type flagged as paid by the company; never more than the standard working days.
- **Salary allowance** (contract line `allowance`, vi: "Phụ cấp lương"): an extra payment tied to the job (position, responsibility, hazard…), with a fixed amount paid every period. Counts toward insurance and the overtime hourly rate.
- **Support payment** (contract line `support`, vi: "Khoản hỗ trợ"): an extra payment not tied to the job (fuel, phone, meals, housing…). Does not count toward insurance or the overtime hourly rate; flagged taxable (`taxable`) or tax-exempt.
- **Back pay** (vi: "Truy lĩnh"), **clawback** (vi: "Truy thu"): an addition or deduction on the payroll of the open period to correct an error in a finalized period. The finalized period's payroll is not cancelled.
- **Adjustment** (vi: "Khoản điều chỉnh"): one back pay (positive) or clawback (negative) for an employee on a draft payroll, with the original period and a reason; counts toward taxable income, not toward insurance.
- **Pay item** (vi: "Khoản lương"): a money column of a payroll line, e.g. `earned` (pay by attendance), `overtime`, `unused_leave` (unused leave paid out on leaving), `adjustment`, `gross`, `exempt` (tax-exempt income), `insurance_base` (insurance contribution salary), `taxable` (taxable income, after deductions), `net` (net pay), `cost` (gross plus the employer's contributions).
- **Payroll totals by department** (`totals`, vi: "Tổng bảng lương theo phòng ban"): pay items summed by department, decrypted and summed on read, not stored. The `payroll_viewer` role sees only this.
- **Overtime-hour warning** (`overtime_month_limit`, `overtime_year_limit`, vi: "Cảnh báo giờ tăng ca"): a payroll line whose overtime hours exceed the monthly or yearly tax-exempt limit; the excess is taxable.
- **Recompute** (vi: "Tính lại"): a deliberate user action on a draft payroll: gather the current inputs again and recompute every item, run as a user job. Never runs on its own.
- **Inputs hash** (`inputs_hash`, vi: "Mã băm đầu vào"): a hash of the set of source documents with their versions, legal parameters, work calendar, settings and employee data the payroll used. **Sources changed** (vi: "Nguồn đã đổi") when the recomputed hash differs from the stored one; the payroll then cannot be submitted for approval or finalized.
- **Payroll samples** (vi: "Bộ mẫu lương"): sample payroll cases with the expected value of every item, used as the payroll acceptance test ([docs/payroll-samples](./docs/payroll-samples/README.md)).
- **Initial import** (vi: "Nhập ban đầu"): loading the organisation's existing data (employees, dependants, contracts in effect) into the system when they start using it. Contracts imported this way take effect immediately, without approval. Only possible while the legal entity has no finalized pay period. Done in M10.
