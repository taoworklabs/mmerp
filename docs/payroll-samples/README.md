# Payroll samples

Payroll acceptance test (M6): `internal/modules/hrm/payroll_samples_test.go` builds a payroll for all 28 cases and compares it to the dong with the "Bộ mẫu" (Samples) sheet of `payroll-input-samples.xlsx`, in the columns for standard workdays, paid workdays (I–J), pay by workdays (K) and N–AF (overtime, untaken leave, adjustments, total income, tax-exempt income, insurance salary, insurance, deductions, taxable income, tax, net pay, employer contributions, total cost). After editing the file, rerun this test. **Not yet reviewed by an accountant**: the company-level choices are assumptions drawn from public regulations, recorded in the "Giả định" (Assumptions) sheet. Accountant review belongs to the [production-readiness gate](../../roadmap.md#production-readiness-gate), before real data enters the system.

`payroll-input-samples.xlsx` has these sheets:

- **Bộ mẫu** (Samples): 28 cases, period 04/2026 (one row in period 07/2026), legal entity in region I. S rows are full-month attendance; C rows are hire or termination mid-period (with untaken leave), salary changes, leave types, overtime (including over 40 hours/month), back pay and recovery. Employer contributions and total payroll cost are included. Pay by workdays, overtime pay, tax-exempt income, total income, net pay and total cost are Excel formulas so they can be checked.
- **Giả định** (Assumptions): the method chosen for each point the company decides itself, with its basis.
- **Tham số** (Parameters): legal parameters and their effective dates.
- **Nguồn** (Sources): which part was cross-checked against which tool.

Cross-checked against two independently written open-source tools, matching to the dong: every row's tax matches both; every row's insurance (employee and employer) and taxable income match thue-2026, and S rows also match pit; overtime pay and its taxable part match thue-2026's overtime calculator. Pay by workdays and untaken leave have no cross-check source. Both tools are personal projects, not official sources.

## Rebuilding the file

```sh
cd /tmp
git clone https://github.com/thangtd-0050/pit && git -C pit checkout 2844c5c
git clone https://github.com/googlesky/thue-2026 && git -C thue-2026 checkout 2e67d2e
npm init -y && npm i esbuild exceljs
node /path/to/mmerp/docs/payroll-samples/build.mjs   # ORACLE_ROOT=/tmp by default
```

The script stops with a `mismatch` error if a sample figure differs from one of the tools.
