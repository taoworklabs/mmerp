# 0039. Shared master data: partners, items and tax in `shared/`, with their own namespace

Status: accepted · 2026-10-23 · supersedes the "Who owns customers and items" section and the VAT sentence of [ADR-0027](./0027-sales-product.md); carries out the `shared/tax` plan of [ADR-0006](./0006-money-rounding.md)

## Context

The suite is heading the way Odoo went: many products (purchasing, inventory, invoicing) on one data model, reusing the same companies and goods. [ADR-0027](./0027-sales-product.md) kept customers and items in `sales` and deferred promotion to `shared/` until a second product needed them. The product owner wants them in `shared/` before that product arrives, as first-class master data with their own permissions, their own page with files and discussion, and an owner for their correctness.

Doing it now matters because of the shape, not the schema: a customer owned by one product cannot become "a company that is both customer and supplier" by `SET SCHEMA`. Identity has to be separated from the product's relation to it first.

Today `iam` and `record` treat every permission and record-type prefix as a product: `record.Register` panics on an empty product, `iam.Scope` and `Registered` look up `(product, permission)`, the frontend `can()` assumes the same. A `shared` module therefore cannot own permissions or record types. That is the core change this decision needs.

## Decision

### Modules and ownership

- `shared/partner` (schema `partner`): one company or person known to the whole tenant, **identity only**: code, name, tax code, address, phone, email, contact person. Record type `partner.partner`, with its own attachments, discussion and history.
- `shared/item` (schema `item`): the item catalogue as it is (code, name, unit, default price, default VAT rate, active). Not a record type; audited as today.
- `shared/tax`: VAT rates and VAT computation (per line or per total, following the legal entity's `setting.rounding`). Pure Go, no routes, tables or permissions. Sales calls it.
- `shared` modules never import or name `sales`. Employees are not partners (sensitive encrypted fields); `hrm` and `iam.org_units` and `setting` stay where they are.
- **Customer** becomes Sales' **partner role**: `sales.customers` keeps its rows, ids, `code`, owning org unit, payment terms and active flag, gains a required partner reference and loses the identity columns. It stays the record type `sales.customer` with the same permissions, so quotations and orders keep their `customer_id` and Sales visibility is unchanged.
- **Several customers may point to one partner, one per org unit at most.** Two branches selling to one company each keep their own customer (own code, terms, active flag) over one identity. A second customer for the same partner in the same org unit is refused with a stable error code.

### Namespaces in the core

A `shared` module registers roles, permissions and record types under its **own namespace** (`partner.`, `item.`), the way `core` does with `core.`. A namespace is not a product.

- Record types carry two flags. `Shared`: the namespace is not a product, so `ProductGate` never refuses it and `products_with_data` ignores it. `Anywhere`: a permission of the type held at any org unit covers every record (the type has no org-unit scope); `Locate` returns org unit 0 and no error when the record exists, `platform.ErrNotFound` otherwise.
- `iam` already treats role prefixes as namespaces; it and the frontend `can()` learn that a namespace may be a shared one.
- No role grants another namespace's permissions: partner and item permissions come from partner and item roles.

| Permission | Granted at | Allows |
| --- | --- | --- |
| `partner.view` | Anywhere | Reading the identity of every partner |
| `partner.edit` | Anywhere | Creating partners and editing any partner's identity |
| `item.view` | Anywhere | Reading the item catalogue |
| `item.manage` | Tenant-wide | Creating and editing items |

Roles: `partner.viewer`, `partner.editor`, `item.viewer`, `item.admin` (tenant-wide only). The Sales role `catalog_admin` and the permissions `sales.item.view` and `sales.item.manage` are removed.

### Frontend

Screens stay in the products: Sales keeps "Customers" (the customer role and the partner's identity fields) and "Items", looking and behaving as today. Identity fields are editable only for holders of `partner.edit`. Reusable master-data UI (partner identity fields, partner picker, item picker) lives in `web/src/shared/master`, importable by every area, calling only `shared/api/core`; dependency-cruiser enforces it with a planted violation in `web/lint-fixtures`. There is no separate "Master data" area or home tile. Partner record links open in the Sales area for now: Sales declares `partner.partner` in its `recordTypes`.

### Data and migration calls

- **Grants are migrated so nobody loses an action:** holders of a Sales role that grants customer view get `partner.viewer`; customer edit, `partner.editor`; item view, `item.viewer`; `sales.catalog_admin`, `item.admin`. Scopes are copied as granted.
- **Partner code** is issued from the series `partner.partner` (prefix `DT`); existing partners get generated codes. Customer codes (`KH…`) stay on the customer and keep printing.
- **Tax code is unique among partners** when present, compared trimmed and case-insensitive. Migration: customers sharing a tax code collapse onto one partner, identity from the most recently updated; customers without a tax code each get their own partner. Data is simulated only, so no merge UI.
- Existing files, comments and history stay on the `sales.customer` records; the partner page starts empty and the Sales customer page shows both.
- Documents copy the customer's code and the partner's identity at save and send, the same fields as today; prints and totals do not change.
- Disabled Sales: customer-role writes stay gated by `sales`. Partner and item writes are never product-gated, so another enabled product can still edit them.

### Deliberate behaviour changes

- Holders of `partner.view` see the identity of **every** partner, including those whose customers belong to other org units. They still see only the customers, quotations and orders of their own scope.
- Holders of `partner.edit` may edit any partner's identity; the change shows in every org unit and product using it, and is audited on the partner.
- Creating a customer with a tax code that already belongs to a partner attaches it to that partner (identity prefilled) instead of creating a second company.

## Rejected alternatives

- **Keep master data under Sales permissions in `core`/`sales`:** no owner, and every product would need Sales permissions to read a company.
- **A `res.partner` holding employees, users and company settings:** mixes sensitive and unrelated data; the partner is identity only.
- **A new tier or a `core` master-data module:** `core` is business-agnostic, and master data is exactly what `shared` is for.
- **Moving tables with `SET SCHEMA` later:** cannot split identity from role.

## Consequences

- Each later product adds a role table in its own schema pointing to `partner.partners` (Purchasing: supplier), with its own owner, terms and flags.
- Next round: units of measure, payment terms, currencies, bank accounts, administrative units, supplier role. Not built here.
- `shared/master` is the one frontend folder areas share; it must not grow product logic.
- How two areas share the link to one `partner.partner` record is decided with Purchasing.
- `backend.md` (layer table, promotion rule), `products/sales.md` and `CONTEXT.md` follow this ADR.
