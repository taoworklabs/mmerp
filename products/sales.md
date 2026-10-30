# Sales product

Selling goods and services, for any trade: customers, a catalogue of items, quotations and sales orders. Sales is the second product. Hard-to-reverse decisions are in [ADR-0027](../docs/adr/0027-sales-product.md); terminology is in the Sales section of [CONTEXT.md](../CONTEXT.md#sales).

## Scope

| In the first release | Not yet |
| --- | --- |
| Customers (Sales' partner role over a shared partner), with an owning org unit | CRM pipeline, leads, activities |
| Item catalogue (goods and services alike) with default price and VAT rate, stock tracking per item (lot, serial, negative stock), warehouses with on hand and stock card | Units of measure conversion, price lists |
| Quotations: approval, PDF print, turning into an order | Payment, invoices and e-invoices |
| Sales orders, from a quotation or entered directly | Accounting, business lines |
| Deliveries from a posted order, taking goods out of a warehouse | Delivery note print, transfers between warehouses, stocktakes, receipts from suppliers, stock valuation, hard reservations |
| Per-line discount, VAT per line, rounding per legal-entity setting | Foreign currencies, document-level discount, cash rounding |

Nothing is specific to a trade: an item is anything sold with a unit and a price.

## Module

A single module `internal/modules/sales`, schema `sales`, product `sales`. Depends on no other product. Uses `iam`, `record`, `audit`, `setting`, `printing` and `numbering` (customer codes) and the shared `partner`, `item` and `tax` modules directly (identity, catalogue, VAT) and `stock` (warehouses and the stock ledger, for deliveries); `approval`, document numbers, `attachment`, `discussion` and `notification` work through `record` with nothing to add. Emits no hooks.

## Entities

| Table | Meaning |
| --- | --- |
| `sales.customers` | Customers, Sales' partner role: `partner_id`, `org_unit_id` (the owning org unit that decides who sees them), payment terms, active flag; one per partner per org unit at most. Identity is in `partner.partners` |
| `partner.partners`, `item.items` | Shared, read by Sales: partner identity and the item catalogue (tenant-wide). Not Sales tables |
| `sales.headers` | Quotations and orders (`kind`), one row per document, keyed by the `record.documents` id: customer details as copied, terms, totals |
| `sales.deliveries` | Deliveries, one row per document keyed by the `record.documents` id: the order, the warehouse, org unit as copied from the order |
| `sales.delivery_lines` | Lines of a delivery: the order line position, copied item code, description and unit, lot or serial, quantity |
| `sales.lines` | Lines of a quotation or order, in order: item, copied code and unit, description, quantity, unit price, discount, VAT rate, and the computed amounts |

## Record types

| Type code | Kind | `posted` means | Date for period lock | `OnTransition` | Approval fields |
| --- | --- | --- | --- | --- | --- |
| `sales.customer` | catalogue | — | — | — | — |
| `sales.quote` | document `BG` | Approved: may be sent and turned into an order | Quotation date | Into `posted`: customer still active. Into `cancelled`: no non-cancelled order from it | `max_discount`, `items` |
| `sales.order` | document `DH` | Confirmed | Order date | Into `posted`: customer still active | `max_discount`, `items` |
| `sales.delivery` | document `GH` | Goods have left the warehouse | Delivery date | Into `posted`: takes the goods out through `stock` (refused while the order's cancel awaits approval). Into `cancelled`: puts them back | — |

Approval fields: `amount` (the header, the total including VAT), `max_discount` (the largest line discount, in percent, measured against the item's catalogue price: a unit price cut below it counts as discount, so lowering the price never slips past a discount rule) and `items` (the set of item ids on the lines). With no approval rule, sending posts at once. Sales approval rules are configured inside the Sales area (`/sales/approval-rules`), needing `sales.approval.manage` or `core.approval.manage`.

Attachments and discussion are on customers, quotations and orders. Printing is on quotations and orders.

## Statuses and actions

`record` status (`draft`, `pending_approval`, `posted`, `cancelled`) says whether the document is in effect and editable. Sales labels: a `posted` quotation is "Đã duyệt" (Approved), a `posted` order is "Đã xác nhận" (Confirmed).

| Status | Quotation actions | Order actions | Delivery actions |
| --- | --- | --- | --- |
| `draft` | Edit, delete, send, print (watermarked) | Edit, delete, send, print (watermarked) | Edit, delete, send |
| `pending_approval` | Withdraw (submitter), approve or reject (approver), print (watermarked) | Same | Withdraw, approve or reject |
| `posted` | Cancel (refused while an order from it is not cancelled), print, **create order** (when not expired and not ordered) | Cancel (refused while a non-cancelled delivery exists: `sales_order_has_delivery`), print | Cancel |
| `cancelled` | Print (watermarked), read | Print (watermarked), read | Read |

Each action needs the permission in the table of [Permissions](#permissions), and `allowed_actions` lists exactly those available, including `print` and the quotation's `create_order`. A locked period removes every write action, as for every document.

**Progress** is derived, never stored ([ADR-0027](../docs/adr/0027-sales-product.md#progress-is-derived-not-stored)):

- a quotation is **ordered** while a non-cancelled order comes from it; it then shows that order;
- a quotation is **expired** once its validity date is before today (tenant time zone). An expired quotation cannot become an order; it stays readable and printable.
- an order's **delivered** quantity per line is the sum of its posted deliveries; its **delivery state** is `none`, `partial` or `full`, or `not_applicable` when it has no stock lines;
- **committed** is what confirmed orders still owe, and **available** is on hand less committed, per legal entity and item (`GET /sales/availability`); a draft or pending order shows available per stock line as a warning only.

**Deliveries.** A delivery (`GH`) is made from a posted order for one warehouse of the order's legal entity; an order may have several. Only stock lines (tracking other than none) go on it, and a line never exceeds ordered less the other non-cancelled deliveries. A delivery reuses the order's permissions and org unit and has no approval fields of its own. `GET /sales/orders/{id}/delivery-options` gives warehouses, remaining quantities, suggested lots (first expiry first) and serials.

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

On each draft save a document copies the customer's code, name, tax code, address, phone, email and contact person, and for each line the item's code and unit. The line description, unit price and VAT rate start from the item and may be changed on the line; payment terms start from the customer's. Sending the draft for approval copies the customer's details once more, so the approver sees, and the document posts with, the customer as it is at sending. After that these copies are final: editing the customer or the item later, even while the document waits for approval, never changes a quotation or order. The print of a posted document is also frozen ([ADR-0026](../docs/adr/0026-printing.md)).

An inactive customer or item cannot be put on a draft (`customer_inactive`, `item_inactive`); a document whose customer was deactivated after saving cannot be sent or posted.

## From quotation to order

"Tạo đơn bán hàng" (Create sales order) on a posted, unexpired quotation creates a draft order with the quotation's org unit, customer, terms and lines (prices, discounts and VAT rates as quoted), dated today. Like every draft save it copies the customer's current details, and it refuses a customer or item deactivated since. The order shows its quotation and the quotation its order, each only to whoever may view the other; lists only say that a quotation is ordered or that an order came from a quotation.

- Clicking twice, or retrying after a lost response, returns the same order: under the quotation's row lock, an existing non-cancelled order is returned instead of creating another. The permission to create orders is checked first, so nobody else learns of that order this way.
- The draft order may then be edited like any order, sent, approved and printed.
- An order may also be created directly, without a quotation.

## Duplicate creates

Create forms send a `request_id` generated when the form opens. Sending the same form twice (double click, retry after a timeout) returns the document created the first time. Customer codes are unique, so a duplicate customer is refused with `customer_code_taken`; item codes and partner codes are unique in their shared modules (`item_code_taken`, `partner_code_taken`), and a customer for a partner already customer of the same org unit is refused too.

## Permissions

| Role | Permissions | Granted |
| --- | --- | --- |
| `staff`, `manager` | `sales.customer.view`, `sales.customer.edit`, `sales.quote.view`, `sales.quote.edit`, `sales.order.view`, `sales.order.edit` | At an org unit or tenant-wide |
| `viewer` | The `view` permissions above | At an org unit or tenant-wide |
| `approval_admin` | `sales.approval.manage` | Tenant-wide only |

`staff` and `manager` carry the same permissions; approval rules route steps to `manager`.

Partners and items have their own roles in the shared modules; a Sales user needs them too, as Sales roles no longer grant them:

| Role | Permissions | Granted |
| --- | --- | --- |
| `partner.viewer` | `partner.view` | Anywhere |
| `partner.editor` | `partner.view`, `partner.edit` | Anywhere |
| `item.viewer` | `item.view` | Anywhere |
| `item.admin` | `item.view`, `item.manage` | Tenant-wide only |

The migration granted them to existing holders of Sales roles (customer view: `partner.viewer`; customer edit: `partner.editor`; item view: `item.viewer`; `catalog_admin`: `item.admin`), scopes copied.

- Customers, quotations and orders are seen only under the org units where the user holds a role (subtrees included). A user granted at "Phòng kinh doanh 1" sees neither the customers nor the documents of "Phòng kinh doanh 2"; one granted at the company sees both.
- Creating a quotation or order needs `edit` at its org unit and `view` on its customer. Turning a quotation into an order needs `sales.quote.view` on the quotation and `sales.order.edit` at its org unit.
- Items are one catalogue for the tenant: whoever holds `item.view` anywhere reads all of it. Writing a quotation or order needs no item role: the line's item cell reads active items through `GET /sales/line-items`, open to whoever holds `sales.quote.edit` or `sales.order.edit` anywhere; `item.view` is for the catalogue screen. Partner identity is the same: whoever holds `partner.view` anywhere reads every partner, while customers stay scoped to their org unit. Identity fields on a customer are editable only with `partner.edit`.
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
