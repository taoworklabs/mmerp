import { Navigate, Route, Routes } from 'react-router'
import { useCan } from '@/shared/auth/me'
import { ApprovalRules } from '@/shared/document'
import { ForbiddenPage, NotFoundPage } from '@/shared/ui/states'
import { GoodsReceiptPage } from './pages/GoodsReceiptPage'
import { GoodsReceiptsPage } from './pages/GoodsReceiptsPage'
import { InvoicePage } from './pages/InvoicePage'
import { InvoicesPage } from './pages/InvoicesPage'
import { NewGoodsReceiptPage } from './pages/NewGoodsReceiptPage'
import { NewInvoicePage } from './pages/NewInvoicePage'
import { NewOrderPage } from './pages/NewOrderPage'
import { NewPaymentPage } from './pages/NewPaymentPage'
import { NewSupplierPage } from './pages/NewSupplierPage'
import { OrderPage } from './pages/OrderPage'
import { OrdersPage } from './pages/OrdersPage'
import { PaymentPage } from './pages/PaymentPage'
import { PaymentsPage } from './pages/PaymentsPage'
import { SupplierPage } from './pages/SupplierPage'
import { SuppliersPage } from './pages/SuppliersPage'

export default function PurchaseRoutes() {
  const canSuppliers = useCan('purchase.supplier.view')
  const canOrders = useCan('purchase.order.view')
  const canRules = useCan(['purchase.approval.manage', 'core.approval.manage'])
  return (
    <Routes>
      <Route index element={<Navigate to={canOrders ? 'orders' : 'suppliers'} replace />} />
      <Route path="suppliers" element={canSuppliers ? <SuppliersPage /> : <ForbiddenPage />} />
      <Route path="suppliers/new" element={canSuppliers ? <NewSupplierPage /> : <ForbiddenPage />} />
      {/* The API decides per record; one outside the user's reach answers 404. */}
      <Route path="suppliers/:id" element={canSuppliers ? <SupplierPage /> : <ForbiddenPage />} />
      <Route path="orders" element={canOrders ? <OrdersPage /> : <ForbiddenPage />} />
      <Route path="orders/new" element={canOrders ? <NewOrderPage /> : <ForbiddenPage />} />
      <Route path="orders/:id" element={canOrders ? <OrderPage /> : <ForbiddenPage />} />
      <Route path="goods-receipts" element={canOrders ? <GoodsReceiptsPage /> : <ForbiddenPage />} />
      {/* Without create_goods_receipt on the order the new page itself answers forbidden. */}
      <Route path="goods-receipts/new" element={canOrders ? <NewGoodsReceiptPage /> : <ForbiddenPage />} />
      <Route path="goods-receipts/:id" element={canOrders ? <GoodsReceiptPage /> : <ForbiddenPage />} />
      <Route path="invoices" element={canOrders ? <InvoicesPage /> : <ForbiddenPage />} />
      {/* Without create_invoice on the order the new page itself answers forbidden. */}
      <Route path="invoices/new" element={canOrders ? <NewInvoicePage /> : <ForbiddenPage />} />
      <Route path="invoices/:id" element={canOrders ? <InvoicePage /> : <ForbiddenPage />} />
      <Route path="payments" element={canOrders ? <PaymentsPage /> : <ForbiddenPage />} />
      {/* Without create (or create_payment on the invoice) the new page itself answers forbidden. */}
      <Route path="payments/new" element={canOrders ? <NewPaymentPage /> : <ForbiddenPage />} />
      <Route path="payments/:id" element={canOrders ? <PaymentPage /> : <ForbiddenPage />} />
      <Route
        path="approval-rules/*"
        element={canRules ? <ApprovalRules product="purchase" basePath="/purchase/approval-rules" /> : <ForbiddenPage />}
      />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
