import { Navigate, Route, Routes } from 'react-router'
import { useCan } from '@/shared/auth/me'
import { ApprovalRules } from '@/shared/document'
import { ForbiddenPage, NotFoundPage } from '@/shared/ui/states'
import { CustomerPage } from './pages/CustomerPage'
import { CustomersPage } from './pages/CustomersPage'
import { DocPage } from './pages/DocPage'
import { DocsPage } from './pages/DocsPage'
import { DeliveriesPage } from './pages/DeliveriesPage'
import { DeliveryPage } from './pages/DeliveryPage'
import { InvoicePage } from './pages/InvoicePage'
import { InvoicesPage } from './pages/InvoicesPage'
import { NewCustomerPage } from './pages/NewCustomerPage'
import { NewDeliveryPage } from './pages/NewDeliveryPage'
import { NewInvoicePage } from './pages/NewInvoicePage'
import { NewReceiptPage } from './pages/NewReceiptPage'
import { NewDocPage } from './pages/NewDocPage'
import { PartnerPage } from './pages/PartnerPage'
import { ReceiptPage } from './pages/ReceiptPage'
import { ReceiptsPage } from './pages/ReceiptsPage'

export default function SalesRoutes() {
  const canCustomers = useCan('sales.customer.view')
  const canQuotes = useCan('sales.quote.view')
  const canOrders = useCan('sales.order.view')
  const canPartners = useCan('partner.view')
  const canRules = useCan(['sales.approval.manage', 'core.approval.manage'])
  return (
    <Routes>
      <Route index element={<Navigate to={canQuotes ? 'quotes' : 'customers'} replace />} />
      <Route path="customers" element={canCustomers ? <CustomersPage /> : <ForbiddenPage />} />
      <Route path="customers/new" element={canCustomers ? <NewCustomerPage /> : <ForbiddenPage />} />
      {/* The API decides per record; one outside the user's reach answers 404. */}
      <Route path="customers/:id" element={canCustomers ? <CustomerPage /> : <ForbiddenPage />} />
      <Route path="partners/:id" element={canPartners ? <PartnerPage /> : <ForbiddenPage />} />
      <Route path="quotes" element={canQuotes ? <DocsPage key="quote" kind="quote" /> : <ForbiddenPage />} />
      <Route path="quotes/new" element={canQuotes ? <NewDocPage key="quote" kind="quote" /> : <ForbiddenPage />} />
      <Route path="quotes/:id" element={canQuotes ? <DocPage key="quote" kind="quote" /> : <ForbiddenPage />} />
      <Route path="orders" element={canOrders ? <DocsPage key="order" kind="order" /> : <ForbiddenPage />} />
      <Route path="orders/new" element={canOrders ? <NewDocPage key="order" kind="order" /> : <ForbiddenPage />} />
      <Route path="orders/:id" element={canOrders ? <DocPage key="order" kind="order" /> : <ForbiddenPage />} />
      <Route path="deliveries" element={canOrders ? <DeliveriesPage /> : <ForbiddenPage />} />
      {/* Without create_delivery on the order the new page itself answers forbidden. */}
      <Route path="deliveries/new" element={canOrders ? <NewDeliveryPage /> : <ForbiddenPage />} />
      <Route path="deliveries/:id" element={canOrders ? <DeliveryPage /> : <ForbiddenPage />} />
      <Route path="invoices" element={canOrders ? <InvoicesPage /> : <ForbiddenPage />} />
      {/* Without create_invoice on the order the new page itself answers forbidden. */}
      <Route path="invoices/new" element={canOrders ? <NewInvoicePage /> : <ForbiddenPage />} />
      <Route path="invoices/:id" element={canOrders ? <InvoicePage /> : <ForbiddenPage />} />
      <Route path="receipts" element={canOrders ? <ReceiptsPage /> : <ForbiddenPage />} />
      {/* Without create (or create_receipt on the invoice) the new page itself answers forbidden. */}
      <Route path="receipts/new" element={canOrders ? <NewReceiptPage /> : <ForbiddenPage />} />
      <Route path="receipts/:id" element={canOrders ? <ReceiptPage /> : <ForbiddenPage />} />
      <Route path="approval-rules/*" element={canRules ? <ApprovalRules product="sales" basePath="/sales/approval-rules" /> : <ForbiddenPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
