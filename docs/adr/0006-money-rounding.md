# 0006. Money rounding

Status: accepted · 2026-10-05

## Context

The first product (HRM) computes salaries, insurance and personal income tax (thuế TNCN), all with fractions. A future sales product will compute percentage discounts and value-added tax (thuế GTGT). Some organisations want tax computed per line, others on the total. Allowing free configuration would double the number of cases to test for every calculation, and the same document could produce two different results.

## Decision

**Fixed in code:**

- Money is stored as `bigint` in the currency's minor unit (per ISO 4217: VND in dong, USD in cents).
- A single rounding function in `platform`, rounding half away from zero. The function takes the rule as a parameter and does not read settings, because `platform` must not import `core`.
- Rates, coefficients and unit prices with fractions are computed with an exact decimal type, never `float`, and are only rounded to `bigint` at the rounding points below.

**Configured per legal entity** (`setting`):

| Key | Value | Default |
| --- | --- | --- |
| Rounding point for tax and insurance | `line`: each line (each item, each employee × each component) rounds on its own, and the total is the sum of the rounded lines · `total`: round once on the total, then allocate down to the lines (see below) | `line` |
| Cash rounding | Off, or a rounding step (100, 500, 1,000 VND). The difference is recorded as a separate "rounding" line, without changing the goods, tax or salary amounts | Off |

**`total` mode: scope and allocation**

- The "total" is the total of **one component within one document**: for example the employee share of social insurance (BHXH) on one payroll, or the 8% VAT on one invoice. Different components are never rounded together.
- Each line is always an integer, because payslips, invoices and business lines all need a per-line amount. The difference between the rounded total and the sum of the lines is allocated by the **largest remainder method**:
  1. Compute the unrounded total, then round it with the `platform` function.
  2. Each line takes its integer part (truncated towards zero).
  3. Subtract the sum of the integer parts from the rounded total to get a number of units. Hand out the units one at a time to the lines with the largest fractional parts; between two lines with equal fractional parts, the earlier line goes first.
- As a result, the sum of the lines always equals the document total, and payslips, payroll and business lines agree. Example with two components of 0.6 dong: the total 1.2 rounds to 1; the integer parts of the two lines are 0 and 0; 1 unit remains, given to the first line → 1 and 0.
- The algorithm lives in `platform`, next to the rounding function, with its own tests for negative numbers, several lines with equal fractional parts, and a zero total.

**Rules so that configuration cannot corrupt data:**

- Changing the configuration only affects `draft` documents. `posted` documents have already stored their amounts and business lines ([ADR-0003](./0003-business-lines-are-data.md)) and are never recomputed.
- The calculating module reads the configuration of the document's legal entity, then passes it to the rounding function. VAT is computed in `shared/tax`; salary, insurance and personal income tax are computed in `hrm`.

## Consequences

- Tests for every calculation run with both `line` and `total`.
- An accountant must reconfirm the `line` default against the current e-invoice and personal income tax finalisation rules before production use.
- An organisation that switches from `line` to `total` mid-period may have documents in the same period using two different methods. Accepted, because each document remains internally consistent.
