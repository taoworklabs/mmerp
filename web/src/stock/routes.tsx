import { Navigate, Route, Routes } from 'react-router'
import { useCan } from '@/shared/auth/me'
import { ForbiddenPage, NotFoundPage } from '@/shared/ui/states'
import { AdjustmentPage } from './pages/AdjustmentPage'
import { AdjustmentsPage } from './pages/AdjustmentsPage'
import { ItemsPage } from './pages/ItemsPage'
import { NewAdjustmentPage } from './pages/NewAdjustmentPage'
import { NewStocktakePage } from './pages/NewStocktakePage'
import { NewTransferPage } from './pages/NewTransferPage'
import { OnHandPage } from './pages/OnHandPage'
import { StockCardPage } from './pages/StockCardPage'
import { StocktakePage } from './pages/StocktakePage'
import { StocktakesPage } from './pages/StocktakesPage'
import { TransferPage } from './pages/TransferPage'
import { TransfersPage } from './pages/TransfersPage'
import { WarehousesPage } from './pages/WarehousesPage'

export default function StockRoutes() {
  const canItems = useCan('item.view')
  const canStock = useCan('stock.view')
  return (
    <Routes>
      <Route index element={<Navigate to={canStock ? 'on-hand' : 'items'} replace />} />
      <Route path="items" element={canItems ? <ItemsPage /> : <ForbiddenPage />} />
      <Route path="on-hand" element={canStock ? <OnHandPage /> : <ForbiddenPage />} />
      <Route path="card" element={canStock ? <StockCardPage /> : <ForbiddenPage />} />
      <Route path="adjustments" element={canStock ? <AdjustmentsPage /> : <ForbiddenPage />} />
      {/* Without create in the type's actions the new page itself answers forbidden. */}
      <Route path="adjustments/new" element={canStock ? <NewAdjustmentPage /> : <ForbiddenPage />} />
      <Route path="adjustments/:id" element={canStock ? <AdjustmentPage /> : <ForbiddenPage />} />
      <Route path="transfers" element={canStock ? <TransfersPage /> : <ForbiddenPage />} />
      <Route path="transfers/new" element={canStock ? <NewTransferPage /> : <ForbiddenPage />} />
      <Route path="transfers/:id" element={canStock ? <TransferPage /> : <ForbiddenPage />} />
      <Route path="stocktakes" element={canStock ? <StocktakesPage /> : <ForbiddenPage />} />
      <Route path="stocktakes/new" element={canStock ? <NewStocktakePage /> : <ForbiddenPage />} />
      <Route path="stocktakes/:id" element={canStock ? <StocktakePage /> : <ForbiddenPage />} />
      <Route path="warehouses" element={canStock ? <WarehousesPage /> : <ForbiddenPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
