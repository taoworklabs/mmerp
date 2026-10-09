# 0027. The Sales product

Status: accepted · 2026-10-09 · replaces the "probe product" row of the roadmap's "Not in this roadmap" table; amends [ADR-0023](./0023-build-core-first.md) ("a second product remains outside the roadmap") and the sentence "VAT is computed in `shared/tax`" of [ADR-0006](./0006-money-rounding.md)

## Context

M11 to M13 finished the shared document features. The roadmap planned a throwaway "probe product" with one unreleased document type, to find what in the core still assumes HRM; issues #1 to #15 list the suspects. A real second product does that job better: every suspected limit is checked against a concrete need instead of a guess, and what is built stays. Sales is the product almost every SMB needs, whatever its trade: customers, a catalogue of goods and services, quotations and sales orders.

The product owner decided to open Sales as the second product, generic across trades. The first release stops at the order: no CRM pipeline, stock, delivery, payment, accounting, e-invoices or foreign currencies. M10 and the production-readiness gate stay on hold; only simulated data.

## Decision

### Product and module

- Product `sales`, depending on no other product. One module `internal/modules/sales`, schema `sales`. Same reason as HRM for one module: quotations, orders, customers and items share everything, and modules in `modules/` cannot import each other.
- `deps.go`: `iam`, `record`, `audit`, `setting`, `printing`. No hooks emitted. No `dataio` import or export in the first release: printing is the product's export, and lists are read on screen.
- `depguard` gets one rule per product module denying the other product modules.

### Who owns customers and items

**Customers and items belong to `sales`**, not to `shared/`. No other product uses them; [backend.md](../../backend.md#data-ownership) already says to promote an entity to `shared/` *before* a second product starts using it, with `ALTER TABLE … SET SCHEMA`, ids unchanged. A generic master-data module (partners with roles, an item master with variants and units of measure) is not built ahead of need.

- `sales.customers`: a catalogue record type `sales.customer`. Code (unique in the tenant, case-insensitive), name, optional tax code, address, phone, email, contact person, default payment terms, an **owning org unit** and an active flag. The org unit decides who sees the customer.
- `sales.items`: code (unique, case-insensitive), name, unit of measure (free text), default unit price, default VAT rate, active flag. Tenant-wide: one catalogue for every org unit. Not a record type: no lifecycle, attachments or discussion are needed yet; changes are audited.
- No goods/service distinction, no stock fields: nothing reads them until inventory exists.

### Documents

| Type | Prefix | `posted` means | Date for period lock | `OnTransition` | Approval fields |
| --- | --- | --- | --- | --- | --- |
| `sales.quote` | `BG` | Quotation approved, may be sent to the customer and turned into an order | Quotation date | Into `posted`: the customer is still active. Into `cancelled`: refused while a non-cancelled order comes from it | `max_discount` |
| `sales.order` | `DH` | Order confirmed | Order date | Into `posted`: the customer is still active | `max_discount` |

- Both types share one header table `sales.headers` (`kind` with a `CHECK` on `quote`, `order`) and one line table `sales.lines`, keyed by the document id; the columns that only one kind uses (`valid_until`, `delivery_date`, `quote_id`) have `CHECK`s tying them to the kind. One code path computes, saves and prints both.
- `record.documents.amount` is the document total including VAT; `max_discount` is the largest line discount percentage. Approval rules can therefore say "orders over 500 million" or "any discount above 10 %".
- Numbering, gaps on deleted drafts, and the period lock are the core's, unchanged: quotations and orders need no gap-free numbers (invoices will).
- **No business lines.** Revenue arises on invoicing or delivery, not on an order; `posting` is not called.

### Progress is derived, not stored

`record` status says only whether the document is in effect. Sales progress is not stored in any status column:

- A quotation is **ordered** while a non-cancelled order points to it (`quote_id`), and **expired** once its `valid_until` is before today in the tenant time zone. Both are computed on read.
- An order has no progress yet. Delivery and payment progress arrive with those products, as module status columns ([documents.md](../../documents.md#progress-status-owned-by-the-module)).

So there is no module status column to keep in step and no transition function to guard; issue #11 holds.

### Creating an order from a quotation

`POST /sales/quotes/{id}/order`, in one transaction:

1. `record.Lock` on the quotation (period lock `FOR SHARE`, then the quotation row `FOR UPDATE`). Every conversion of one quotation, and its cancellation, therefore run one at a time.
2. If a non-cancelled order already points to the quotation, return that order's id: a retried or doubled click gets the same order.
3. Otherwise check the quotation is `posted` and not expired, and that the actor may create orders at its org unit, then create a draft order with the same org unit, today's date, and the quotation's customer details, terms and lines exactly as quoted.

Only the order is written; the quotation is only locked, so "one transaction writes one document" holds. Deleting the draft order, or cancelling the order, frees the quotation for a new conversion. Cancelling the quotation is refused while an order from it is not cancelled.

### Retried creates do not duplicate

Every create body (customer excepted, whose unique code already guards it) carries a client-generated `request_id` (UUID), stored with a unique index. A create whose `request_id` already exists returns the existing document instead of a new one; two concurrent creates with the same id leave one document, the loser reading the winner's id after its unique violation. The frontend generates the id when the form opens.

### Money

- Quantity: `numeric(15,3)`, positive. Discount: a percentage `numeric(5,2)` from 0 to 100, per line. Unit price: `bigint` in đồng, zero or more. VAT rate: one of `none` (not subject to VAT), `0`, `5`, `8`, `10`, with a `CHECK`.
- Per line: `amount = round(quantity × unit_price)`, `discount = round(amount × discount / 100)`, `net = amount − discount`. Every rounding goes through `platform.Round`; intermediate values are `decimal`.
- VAT follows the legal entity's `setting.rounding` ([ADR-0006](./0006-money-rounding.md)): `line` rounds `net × rate` per line; `total` computes the VAT of each rate over the document once and allocates it to the lines with `platform.Allocate`. The VAT of each line is stored, so lines always add up to the totals.
- Header totals are sums of stored line values: `subtotal` (Σ amount), `discount_total`, `vat_total`, `total = subtotal − discount_total + vat_total`.
- The server computes every amount on each save of a draft; the frontend shows what the server returned and computes nothing. A posted document is never recomputed.
- **VAT is computed in `sales`**, not in a `shared/tax` module as ADR-0006 planned: one product computes VAT, and a shared module for one caller is a guess. It moves to `shared/tax` when a second product computes VAT.

### What a document keeps

On each save of a draft, the document copies from the catalogues what it shows: the customer's code, name, tax code, address, phone, email and contact person; for each line the item's code and unit, plus the line description (prefilled from the item name, editable), the unit price and the VAT rate (prefilled from the item, editable). Payment terms (prefilled from the customer), delivery terms, validity date and note are the document's own fields. Once a document leaves `draft` it cannot be edited, so these copies are what the quotation or order said; later edits of the customer or item never reach it. The print of a posted document is additionally frozen by `printing` when it posts.

Saving refuses an inactive customer (`customer_inactive`) or item (`item_inactive`); posting refuses a customer deactivated since.

### Permissions and data scope

| Permission | Granted at | Allows |
| --- | --- | --- |
| `sales.customer.view`, `sales.customer.edit` | Org unit | Customers whose owning org unit is in scope |
| `sales.quote.view`, `sales.quote.edit` | Org unit | Quotations whose org unit is in scope; edit also sends, withdraws, cancels and attaches |
| `sales.order.view`, `sales.order.edit` | Org unit | The same for orders |
| `sales.item.view` | Anywhere | Reading the item catalogue |
| `sales.item.manage` | Tenant-wide | Creating and editing items |
| `sales.approval.manage` | Tenant-wide | Sales approval rules ([ADR-0022](./0022-approval-rule-permissions-per-product.md)) |

Roles: `staff` and `manager` (customers, quotations and orders, view and edit; read items: the same permissions, two names so approval rules can route to managers), `viewer` (read everything), `catalog_admin` (items; tenant-wide only), `approval_admin` (tenant-wide only).

- Nobody sees every customer or document by default: a user sees what lies under the org units where they hold a role. A sales team is an org unit (`department` or `branch`); a person who should see only their own work is granted at a node of their own.
- Creating a document needs edit permission at its org unit **and** view permission on the customer. A document's org unit may differ from the customer's; the document then shows the customer details it copied, even to someone who cannot open the customer.
- `print`, `view_files` are answered with view; `attach` with edit. Customers answer `view`, `edit`, `view_files`, `attach`.

### Printing

Two templates, `sales.quote` and `sales.order`, registered with `printing`, sharing one layout: seller (the document's legal entity), customer as copied, lines, totals by VAT rate, terms, and two text blocks (`terms`, `footer`) tenants can reword. Printing needs `print`, which is answered with view: a quotation is meant to be handed to the customer.

### Personal data

A customer may be a person: name, address, phone, email, tax code and contact person are ordinary personal data under the Law on Personal Data Protection. None is a sensitive field: no column encryption; reading goes by the customer's org-unit scope, and every change is audited. Documents copy the same fields, readable by whoever may view the document.

### Core limits checked against Sales (issues #1 to #15)

| Issue | Sales need | Verdict |
| --- | --- | --- |
| #1 posting lines fit only payroll | Orders write no business lines | Not needed now; reopen with invoicing |
| #2 every document needs an org unit | Quotations and orders belong to a sales team, which is an org unit | Holds |
| #3 one-dimensional scope | Teams are org units; nothing scopes by channel or warehouse yet | Holds |
| #4 number format and issue time | `BG-2026-00001`, gaps accepted | Holds; reopen with invoices |
| #5 one period lock per legal entity | Sales documents follow the common lock | Holds until accounting |
| #7 amounts without currency | VND only | Holds |
| #8, #9 approval by org unit, field kinds | Roles at the team; `amount` and `max_discount` | Hold |
| #10 closed org unit kinds | Teams are `department` or `branch` | Holds |
| #11 four document statuses | Progress is derived, not stored | Holds |
| #12 posting hooks per instance | No posting | Not needed now |
| #14 web client knows only `/hrm/` | The sales area needs its client | **Fixed** with this product |
| #15 small limits | Approval rules per document type are enough | Holds |

## Rejected alternatives

- **A probe product**: tests guesses about a product nobody uses; Sales tests needs.
- **Customers and items in `shared/` now**: no second user; the promotion rule already handles the day one arrives.
- **Quotation and order progress as status columns** (`open`, `ordered`, `expired`): each would need a transition function kept in step with orders and the calendar; deriving them cannot drift.
- **Converting by editing the quotation into an order**: one record type changing kind breaks numbering, history and approval per type.
- **Idempotency keys in an HTTP header and a generic table**: one unique column per document table does it with no new core concept.
- **Document-level discount**: per-line discount covers it (same percentage on every line); a header discount needs its own allocation rule for VAT.
- **Recomputing totals in the browser**: two implementations of the same money rule would disagree on rounding.

## Consequences

- HRM keeps working unchanged; enabling `PRODUCTS=hrm,sales` shows both areas on the home page.
- Issues #2 to #15 stay open as "checked, holds for Sales"; #1, #4, #5 and #12 return with invoicing and accounting.
- The customer and item tables will move to `shared/` the day another product needs them.
- Users see totals only after saving a draft.
