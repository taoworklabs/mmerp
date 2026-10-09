import { Navigate, Route, Routes } from 'react-router'
import { useCan } from '@/shared/auth/me'
import { ApprovalRules } from '@/shared/document'
import { ForbiddenPage, NotFoundPage } from '@/shared/ui/states'
import { CustomerPage } from './pages/CustomerPage'
import { CustomersPage } from './pages/CustomersPage'
import { DocPage } from './pages/DocPage'
import { DocsPage } from './pages/DocsPage'
import { ItemsPage } from './pages/ItemsPage'
import { NewCustomerPage } from './pages/NewCustomerPage'
import { NewDocPage } from './pages/NewDocPage'

export default function SalesRoutes() {
  const canCustomers = useCan('sales.customer.view')
  const canQuotes = useCan('sales.quote.view')
  const canOrders = useCan('sales.order.view')
  const canItems = useCan('sales.item.view')
  const canRules = useCan(['sales.approval.manage', 'core.approval.manage'])
  return (
    <Routes>
      <Route index element={<Navigate to={canQuotes ? 'quotes' : 'customers'} replace />} />
      <Route path="customers" element={canCustomers ? <CustomersPage /> : <ForbiddenPage />} />
      <Route path="customers/new" element={canCustomers ? <NewCustomerPage /> : <ForbiddenPage />} />
      {/* The API decides per record; one outside the user's reach answers 404. */}
      <Route path="customers/:id" element={canCustomers ? <CustomerPage /> : <ForbiddenPage />} />
      <Route path="quotes" element={canQuotes ? <DocsPage key="quote" kind="quote" /> : <ForbiddenPage />} />
      <Route path="quotes/new" element={canQuotes ? <NewDocPage key="quote" kind="quote" /> : <ForbiddenPage />} />
      <Route path="quotes/:id" element={canQuotes ? <DocPage key="quote" kind="quote" /> : <ForbiddenPage />} />
      <Route path="orders" element={canOrders ? <DocsPage key="order" kind="order" /> : <ForbiddenPage />} />
      <Route path="orders/new" element={canOrders ? <NewDocPage key="order" kind="order" /> : <ForbiddenPage />} />
      <Route path="orders/:id" element={canOrders ? <DocPage key="order" kind="order" /> : <ForbiddenPage />} />
      <Route path="items" element={canItems ? <ItemsPage /> : <ForbiddenPage />} />
      <Route path="approval-rules/*" element={canRules ? <ApprovalRules product="sales" basePath="/sales/approval-rules" /> : <ForbiddenPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
