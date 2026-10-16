import type { Middleware } from 'openapi-fetch'
import type { components } from '@/shared/api/schema.gen'

type OrgUnit = components['schemas']['OrgUnit']
type ListSettings = components['schemas']['ListSettings']

export const orgUnits: OrgUnit[] = [
  { id: 1, kind: 'company', name: 'Công ty CP Demo', parent_id: null },
  { id: 2, kind: 'department', name: 'Kế toán', parent_id: 1 },
  { id: 3, kind: 'branch', name: 'Chi nhánh Hà Nội', parent_id: 1 },
  { id: 4, kind: 'department', name: 'Kinh doanh', parent_id: 3 },
]

// Saved columns and views live only as long as the tab.
const lists = new Map<string, ListSettings>()

const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

// offline answers every API call of the catalogue in the browser, so it never reaches a server:
// the reads its components make come from samples, anything else fails as if offline.
export const offline: Middleware = {
  async onRequest({ request, schemaPath }) {
    if (schemaPath === '/org-units') return json(orgUnits)
    if (schemaPath === '/me/lists/{key}') {
      const key = new URL(request.url).pathname.split('/').pop() ?? ''
      if (request.method === 'PUT') lists.set(key, await request.json())
      return json(lists.get(key) ?? { columns: [], views: [] })
    }
    return json({ code: 'network_error' }, 503)
  },
}
