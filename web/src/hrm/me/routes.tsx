import { Navigate, Route, Routes } from 'react-router'
import { NotFoundPage } from '@/shared/ui/states'
import { DaysPage } from './DaysPage'

export default function MeRoutes() {
  return (
    <Routes>
      <Route index element={<Navigate to="days" replace />} />
      <Route path="days" element={<DaysPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
