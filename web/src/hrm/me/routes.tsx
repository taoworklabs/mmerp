import { Navigate, Route, Routes } from 'react-router'
import { NotFoundPage } from '@/shared/ui/states'
import { LeavePage } from '../pages/LeavePage'
import { LeavesPage } from '../pages/LeavesPage'
import { NewLeavePage } from '../pages/NewLeavePage'
import { NewOvertimePage } from '../pages/NewOvertimePage'
import { OrgStructurePage } from '../pages/OrgStructurePage'
import { OvertimePage } from '../pages/OvertimePage'
import { OvertimesPage } from '../pages/OvertimesPage'
import { DaysPage } from './DaysPage'
import { PayslipPage } from './PayslipPage'
import { PayslipsPage } from './PayslipsPage'
import { ProfilePage } from './ProfilePage'
import { RequestsPage } from './RequestsPage'

export default function MeRoutes() {
  return (
    <Routes>
      <Route index element={<Navigate to="days" replace />} />
      <Route path="days" element={<DaysPage />} />
      <Route path="payslips" element={<PayslipsPage />} />
      <Route path="payslips/:payrollId" element={<PayslipPage />} />
      <Route path="requests" element={<RequestsPage />} />
      <Route path="requests/leave/new" element={<NewLeavePage />} />
      <Route path="requests/leave/:id" element={<LeavePage />} />
      <Route path="requests/overtime/new" element={<NewOvertimePage />} />
      <Route path="requests/overtime/:id" element={<OvertimePage />} />
      <Route path="team/leaves" element={<LeavesPage team />} />
      <Route path="team/overtimes" element={<OvertimesPage team />} />
      <Route path="profile" element={<ProfilePage />} />
      <Route path="org" element={<OrgStructurePage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
