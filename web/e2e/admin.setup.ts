import { test as setup } from '@playwright/test'
import { signedInApi } from './api'

// The install-time admin only administers; give it HRM roles to set up test data and see the HRM menu.
setup('admin can use HRM', async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const { id } = (await (await api.get('/api/me')).json()) as { id: number }
  for (const role of ['hr', 'sensitive_viewer', 'leave_admin', 'payroll']) {
    const res = await api.post(`/api/users/${id}/roles`, { data: { product: 'hrm', role, org_unit_id: null } })
    if (![201, 409].includes(res.status())) throw new Error(`grant ${role}: ${res.status()} ${await res.text()}`)
  }
})
