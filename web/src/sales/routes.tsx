import { Navigate, Route, Routes } from 'react-router'
import { useCan } from '@/shared/auth/me'
import { ApprovalRules } from '@/shared/document'
import { ForbiddenPage, NotFoundPage } from '@/shared/ui/states'
import { AdjustmentPage } from './pages/AdjustmentPage'
import { AdjustmentsPage } from './pages/AdjustmentsPage'
import { CustomerPage } from './pages/CustomerPage'
import { CustomersPage } from './pages/CustomersPage'
import { DocPage } from './pages/DocPage'
import { DocsPage } from './pages/DocsPage'
import { DeliveriesPage } from './pages/DeliveriesPage'
import { DeliveryPage } from './pages/DeliveryPage'
import { ItemsPage } from './pages/ItemsPage'
import { NewAdjustmentPage } from './pages/NewAdjustmentPage'
import { NewCustomerPage } from './pages/NewCustomerPage'
import { NewDeliveryPage } from './pages/NewDeliveryPage'
import { NewDocPage } from './pages/NewDocPage'
import { OnHandPage } from './pages/OnHandPage'
import { PartnerPage } from './pages/PartnerPage'
import { StockCardPage } from './pages/StockCardPage'
import { WarehousesPage } from './pages/WarehousesPage'

export default function SalesRoutes() {
  const canCustomers = useCan('sales.customer.view')
  const canQuotes = useCan('sales.quote.view')
  const canOrders = useCan('sales.order.view')
  const canItems = useCan('item.view')
  const canPartners = useCan('partner.view')
  const canStock = useCan('stock.view')
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
      <Route path="items" element={canItems ? <ItemsPage /> : <ForbiddenPage />} />
      <Route path="stock/on-hand" element={canStock ? <OnHandPage /> : <ForbiddenPage />} />
      <Route path="stock/card" element={canStock ? <StockCardPage /> : <ForbiddenPage />} />
      <Route path="stock/adjustments" element={canStock ? <AdjustmentsPage /> : <ForbiddenPage />} />
      {/* Without create in the type's actions the new page itself answers forbidden. */}
      <Route path="stock/adjustments/new" element={canStock ? <NewAdjustmentPage /> : <ForbiddenPage />} />
      <Route path="stock/adjustments/:id" element={canStock ? <AdjustmentPage /> : <ForbiddenPage />} />
      <Route path="stock/warehouses" element={canStock ? <WarehousesPage /> : <ForbiddenPage />} />
      <Route path="approval-rules/*" element={canRules ? <ApprovalRules product="sales" basePath="/sales/approval-rules" /> : <ForbiddenPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
