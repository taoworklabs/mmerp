# Shared master data: partners, items, tax

## Intent

mmerp is heading toward a suite in the spirit of Odoo: many products (purchasing, inventory, invoicing, …) on one data model. Those products will reuse the same companies and goods. Today the only master data more than one product will need, customers and items, belongs to `sales` (ADR-0027 decided that, waiting for a second product before moving them).

The product owner wants that master data in `shared/` **before** the next product arrives, as first-class master data: its own permissions, its own page with files and discussion, an owner for its correctness. Products add their own **roles** on top (Sales: customer; later Purchasing: supplier). Doing it now rather than with a `SET SCHEMA` later matters because of the shape: a customer owned by one product cannot become "a company that is both customer and supplier" by moving a table.

This round moves the master data that exists today and makes the core able to host master data in `shared`. New master data (units of measure, payment terms, currencies, bank accounts, administrative units, supplier role) comes in the next round, on top of this one.

## Inventory of existing master data and the verdict

| Today | Verdict |
| --- | --- |
| `sales.customers` | Identity moves to `shared/partner`; `sales.customers` stays as the **customer role**, pointing to a partner |
| `sales.items` | Moves to `shared/item` |
| VAT computation in `sales` (`calc.go`) | Moves to `shared/tax`, as ADR-0006 planned |
| `hrm.holidays`, `hrm.work_weeks` | Stay for now. Company-wide in nature; move by `SET SCHEMA` when a second product needs them, no reshape needed |
| `iam.org_units`, `setting` | Stay in `core` |
| `hrm.employees`, `hrm.dependents` | Stay in HRM. Employees are **not** partners (sensitive encrypted fields). A future "salesperson" refers to `iam.users` |
| `hrm.legal_params`, `hrm.contract_types`, `hrm.leave_types` | Stay in HRM |

## Goals

1. **Core can host master data in `shared`.** A `shared` module registers roles, permissions and record types under its **own namespace** (`partner.`, `item.`), the way `core` does with `core.`. A namespace is not a product: `ProductGate` never refuses it, `products_with_data` ignores it, and the frontend permission check (`can()` in `web/src/shared/auth/me.tsx`) understands it. Today `iam` and `record` treat every prefix as a product (`record.Register` panics on an empty product; `iam.Scope`/`Registered` look up `(product, permission)`); this is the core change.
2. **`shared/partner`.** A partner is one company or person, shared by the whole tenant: code, name, tax code, address, phone, email, contact person. It is a catalogue record type (`partner.partner` or similar) with its own attachments, discussion and history. Its own permissions: view (granted anywhere, sees every partner) and edit (granted anywhere, edits any partner's identity); its own roles.
3. **Customer role in Sales.** `sales.customers` keeps its rows, ids, `code`, owning org unit, payment terms and active flag, gains a required partner reference, and loses the identity columns. **Several customers may point to one partner**, one per org unit at most: two branches selling to the same company each keep their own customer (own code, terms, active flag) over one shared identity. `sales.customer` stays a record type with the same permissions, so Sales visibility is unchanged and quotations and orders keep their `customer_id`.
4. **`shared/item`.** The item catalogue moves as it is (code, name, unit, default price, default VAT rate, active), with its own namespace permissions (view anywhere, manage tenant-wide) and roles, replacing `sales.item.*`. Items stay a non-record-type catalogue, audited as today.
5. **`shared/tax`.** VAT rates and the VAT computation (per line or per total, following the legal entity's `setting.rounding`) move out of `sales`; Sales calls them. Pure Go, no routes or permissions.
6. **Screens stay in the products.** Each product area draws its own screens over shared master data. Sales keeps "Customers" (customer role plus the partner's identity fields) and "Items"; they look and behave as today, apart from the changes listed below. Identity fields are editable only for holders of partner edit.
7. **Reusable master-data UI** (partner identity fields, partner picker, item picker) lives in one frontend folder under `web/src/shared/` that every area may import; it calls only `shared/api/core`. dependency-cruiser enforces it, with a planted violation in `web/lint-fixtures` and its `expected.json`.
8. **Docs.** A new ADR supersedes ADR-0027's "Who owns customers and items" section and records the namespace change in the core and the frontend rule. `CONTEXT.md` gains Partner and Partner role and redefines Customer as Sales' role of a partner. `backend.md` (layer table, promotion rule) and `products/sales.md` are updated.

## Non-goals

- No supplier role, units of measure catalogue, payment terms catalogue, currencies, purchase price, item variants, goods/service distinction. Next round.
- No separate "Master data" area or home tile.
- No change to employees, the org tree, settings or the work calendar.
- No change to how quotations and orders compute, copy customer details or print.
- No role that grants another namespace's permissions: partner and item permissions come from partner and item roles.

## Anti-goals (a failure even if it works)

- An existing quotation or order prints, totals or reads differently.
- Anyone loses an action they have today.
- The partner turns into an Odoo-style `res.partner` that also holds employees, users or company settings.
- A `shared` module importing or naming Sales.

## Deliberate behaviour changes

- Holders of partner view see the identity (name, tax code, address, contact) of **every** partner, including those whose only customers belong to other org units. They still see only the customers, quotations and orders of their own scope.
- Holders of partner edit may edit the identity of any partner; a change shows in every org unit and every product using it. Changes are audited on the partner.
- Creating a customer with a tax code that already belongs to a partner attaches the new customer to that partner (identity prefilled) instead of creating a second company. Creating a second customer for the same partner in the same org unit is refused with a stable error code.

## Decisions made with the product owner

- `shared/`, one module per kind of master data; not `core`, not a new tier.
- Accept a core change so `shared` master data has its own namespace, permissions, roles and record type (option chosen over keeping everything under Sales permissions).
- Partner identity is tenant-wide; ownership by org unit belongs to the role.
- One customer per org unit per partner (several branches may each have their own customer of one company).
- Each product area draws its own screens; the Sales "Customers" screen edits identity and customer-role fields together.

## Calls made by the designer (check these)

- **Grants are migrated so nobody loses an action:** every user holding a Sales role that grants customer view gets the partner viewer role; customer edit gets partner editor; item view gets item viewer; `sales.catalog_admin` gets item admin. Scopes are copied as granted (partner and item permissions are "anywhere", so scope only matters for revocation).
- **Partner code** is new, issued from the partner module's code series (prefix is the builder's call); existing partners get generated codes. Customer codes (`KH…`) stay on the customer and keep printing.
- **Tax code is unique among partners** when present, compared trimmed and case-insensitive. Migration: customers sharing a tax code collapse onto one partner, identity taken from the most recently updated one; customers without a tax code each get their own partner. Data is simulated only (ADR-0027), so no merge UI is needed.
- **Existing files, comments and history stay on the `sales.customer` records** they were attached to; nothing becomes more visible than it is. The partner page starts empty. The Sales customer page shows both.
- **Partner record links** open in the Sales area this round (Sales declares `partner.partner` in its `recordTypes`). How two areas share one record type's link is settled when Purchasing arrives.
- **Documents copy** the customer's code and the partner's identity at save and send, exactly the fields they copy today; prints are unchanged.
- **Disabled Sales:** customer-role writes stay gated by `sales` (refused when disabled, as today). Partner and item writes are never product-gated, so another enabled product can still edit them; with only HRM enabled there is simply no screen for them.

## Constraints

- All of `CLAUDE.md`: layering `modules → shared → core → platform`, data engine only, one schema per module, SQL migrations only, never edit a released migration, money in `int64`, rounding through `platform`, lock order for document writes.
- `platform.Module` must still not reference `core` types.
- `shared` modules never import or name `sales`.
- `sales.customer` and the partner record type pass the `recordtest` contract suite.
- Every existing Go test, Vitest and Playwright test keeps passing (adjusted only where a deliberate behaviour change above requires it); `make seed` keeps working and creates partners.
- Areas never import each other; the shared master-data UI folder calls only `shared/api/core`.

## Left to the builder

- How the namespace concept is expressed in `iam`, `record`, `ProductGate` and the frontend, within the rule that a `shared` namespace is never product-gated.
- Table and column layout of `partner` and `item` schemas; migration steps, as long as `sales.customers` ids, item ids, document foreign keys and existing history survive.
- API paths of the partner and item routes, and role names within `CONTEXT.md` naming.
- Which UI pieces move to the shared master-data folder versus stay in `web/src/sales`.
- Ticket breakdown and order (the core namespace change first is the obvious order).
