# Sales product

Selling goods and services, for any trade: customers, a catalogue of items, quotations and sales orders. Sales is the second product. Hard-to-reverse decisions are in [ADR-0027](../docs/adr/0027-sales-product.md); terminology is in the Sales section of [CONTEXT.md](../CONTEXT.md#sales).

## Scope

| In the first release | Not yet |
| --- | --- |
| Customers, with an owning org unit | CRM pipeline, leads, activities |
| Item catalogue (goods and services alike) with default price and VAT rate | Stock, warehouses, units of measure conversion, price lists |
| Quotations: approval, PDF print, turning into an order | Delivery, payment, invoices and e-invoices |
| Sales orders, from a quotation or entered directly | Accounting, business lines |
| Per-line discount, VAT per line, rounding per legal-entity setting | Foreign currencies, document-level discount, cash rounding |

Nothing is specific to a trade: an item is anything sold with a unit and a price.

## Module

A single module `internal/modules/sales`, schema `sales`, product `sales`. Depends on no other product. Uses `iam`, `record`, `audit`, `setting`, `printing` directly; `approval`, `numbering`, `attachment`, `discussion` and `notification` work through `record` with nothing to add. Emits no hooks.

## Entities

| Table | Meaning |
| --- | --- |
| `sales.customers` | Customers; `org_unit_id` is the owning org unit that decides who sees them |
| `sales.items` | Item catalogue, tenant-wide |
| `sales.headers` | Quotations and orders (`kind`), one row per document, keyed by the `record.documents` id: customer details as copied, terms, totals, `request_id` |
| `sales.lines` | Lines of a quotation or order, in order: item, copied code and unit, description, quantity, unit price, discount, VAT rate, and the computed amounts |

## Record types

| Type code | Kind | `posted` means | Date for period lock | `OnTransition` | Approval fields |
| --- | --- | --- | --- | --- | --- |
| `sales.customer` | catalogue | — | — | — | — |
| `sales.quote` | document `BG` | Approved: may be sent and turned into an order | Quotation date | Into `posted`: customer still active. Into `cancelled`: no non-cancelled order from it | `max_discount` |
| `sales.order` | document `DH` | Confirmed | Order date | Into `posted`: customer still active | `max_discount` |

Approval fields: `amount` (the header, the total including VAT) and `max_discount` (the largest line discount, in percent). With no approval rule, sending posts at once. Sales approval rules are configured inside the Sales area (`/sales/approval-rules`), needing `sales.approval.manage` or `core.approval.manage`.

Attachments and discussion are on customers, quotations and orders. Printing is on quotations and orders.

## Statuses and actions

`record` status (`draft`, `pending_approval`, `posted`, `cancelled`) says whether the document is in effect and editable. Sales labels: a `posted` quotation is "Đã duyệt" (Approved), a `posted` order is "Đã xác nhận" (Confirmed).

| Status | Quotation actions | Order actions |
| --- | --- | --- |
| `draft` | Edit, delete, send, print (watermarked) | Edit, delete, send, print (watermarked) |
| `pending_approval` | Withdraw (submitter), approve or reject (approver), print (watermarked) | Same |
| `posted` | Cancel (refused while an order from it is not cancelled), print, **create order** (when not expired and not ordered) | Cancel, print |
| `cancelled` | Print (watermarked), read | Print (watermarked), read |

Each action needs the permission in the table of [Permissions](#permissions), and `allowed_actions` lists exactly those available, including `print` and the quotation's `create_order`. A locked period removes every write action, as for every document.

**Progress** is derived, never stored ([ADR-0027](../docs/adr/0027-sales-product.md#progress-is-derived-not-stored)):

- a quotation is **ordered** while a non-cancelled order comes from it; it then shows that order;
- a quotation is **expired** once its validity date is before today (tenant time zone). An expired quotation cannot become an order; it stays readable and printable.

**Changing a document.** A draft is edited in place (with `version`). A posted quotation or order is never edited: cancel it and make a new one. Cancelling a posted order frees its quotation, which can then be turned into an order again.

## Amounts

All amounts are VND, `bigint` in đồng. Each line has a quantity (up to 3 decimals, more than 0), a unit price (whole đồng, 0 or more), a discount percentage (0 to 100, up to 2 decimals) and a VAT rate (`none` = not subject to VAT, `0`, `5`, `8`, `10` %).

| Value | Rule |
| --- | --- |
| Line amount | `round(quantity × unit price)` |
| Line discount | `round(line amount × discount % / 100)` |
| Line net | `line amount − line discount` |
| Line VAT, `line` rounding | `round(line net × rate)` per line |
| Line VAT, `total` rounding | For each rate: `round(Σ line net × rate)` once, allocated to the lines of that rate by largest remainder |
| Subtotal, discount total, VAT total | Sums of the lines |
| Total | `subtotal − discount total + VAT total` |

Rounding is half away from zero, always through `platform.Round`/`platform.Allocate`; the rounding point is the legal entity's `setting.rounding` ([ADR-0006](../docs/adr/0006-money-rounding.md)), read when the draft is saved. Example: two lines of net 10,005 at 8 % give VAT 800.4 each: `line` stores 800 and 800 (total 1,600); `total` rounds 1,600.8 to 1,601 and stores 801 and 800.

The server computes and stores every amount on each draft save; screens and prints show stored amounts. A posted document is never recomputed.

## What a document keeps

On each draft save a document copies the customer's code, name, tax code, address, phone, email and contact person, and for each line the item's code and unit. The line description, unit price and VAT rate start from the item and may be changed on the line; payment terms start from the customer's. After the draft stage these copies are final: editing the customer or the item later never changes a quotation or order. The print of a posted document is also frozen ([ADR-0026](../docs/adr/0026-printing.md)).

An inactive customer or item cannot be put on a draft (`customer_inactive`, `item_inactive`); a document whose customer was deactivated after saving cannot be posted.

## From quotation to order

"Tạo đơn bán hàng" (Create sales order) on a posted, unexpired quotation creates a draft order with the quotation's org unit, customer, terms and lines (prices, discounts and VAT rates as quoted), dated today. Like every draft save it copies the customer's current details, and it refuses a customer or item deactivated since. The order shows its quotation and the quotation its order, each only to whoever may view the other; lists only say that a quotation is ordered or that an order came from a quotation.

- Clicking twice, or retrying after a lost response, returns the same order: under the quotation's row lock, an existing non-cancelled order is returned instead of creating another. The permission to create orders is checked first, so nobody else learns of that order this way.
- The draft order may then be edited like any order, sent, approved and printed.
- An order may also be created directly, without a quotation.

## Duplicate creates

Create forms send a `request_id` generated when the form opens. Sending the same form twice (double click, retry after a timeout) returns the document created the first time. Customer and item codes are unique, so a duplicate customer or item is refused with `customer_code_taken` or `item_code_taken`.

## Permissions

| Role | Permissions | Granted |
| --- | --- | --- |
| `staff`, `manager` | `sales.customer.view`, `sales.customer.edit`, `sales.quote.view`, `sales.quote.edit`, `sales.order.view`, `sales.order.edit`, `sales.item.view` | At an org unit or tenant-wide |
| `viewer` | The `view` permissions above | At an org unit or tenant-wide |
| `catalog_admin` | `sales.item.view`, `sales.item.manage` | Tenant-wide only |
| `approval_admin` | `sales.approval.manage` | Tenant-wide only |

`staff` and `manager` carry the same permissions; approval rules route steps to `manager`.

- Customers, quotations and orders are seen only under the org units where the user holds a role (subtrees included). A user granted at "Phòng kinh doanh 1" sees neither the customers nor the documents of "Phòng kinh doanh 2"; one granted at the company sees both.
- Creating a quotation or order needs `edit` at its org unit and `view` on its customer. Turning a quotation into an order needs `sales.quote.view` on the quotation and `sales.order.edit` at its org unit.
- Items are one catalogue for the tenant: whoever holds `sales.item.view` anywhere reads all of it.
- A record the user cannot view answers not found everywhere, including from attachments, discussion, print and conversion.

With Sales disabled, everything stays readable; printing and attachment downloads still work; every write action disappears.

## Personal data

| Level | Fields | Storage |
| --- | --- | --- |
| Ordinary personal (a customer may be a person) | Customer name, tax code, address, phone, email, contact person; their copies on documents | Not encrypted; read per the org-unit scope; changes audited |

No sensitive fields.

## Frontend

Area `web/src/sales`, menu: customers, quotations, orders; configuration (collapsible): items, approval rules. Quotations and orders use `DocumentPage` with the lines table in the main column; the approval inbox shows a read-only preview of both.

## Proving flow

Customer and item created → quotation drafted with discounted lines → sent → approved by a manager from the approval inbox (notification) → printed to PDF → turned into an order (twice: same order) → order sent and confirmed. Another team's staff sees none of it. Covered by `sales` service tests (amounts in both rounding modes, permissions, lifecycle, `recordtest`) and Playwright.
