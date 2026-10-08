import { expect, request, type APIRequestContext } from '@playwright/test'
import { admin } from './helpers'

// signedInApi is an API client with its own session, for setting up data without the UI.
export async function signedInApi(baseURL: string, login = admin.login, password = admin.password): Promise<APIRequestContext> {
  const api = await request.newContext({ baseURL })
  const res = await api.post('/api/auth/login', { data: { login, password } })
  expect(res.status(), `login ${login}`).toBe(204)
  return api
}

export async function created(res: Promise<{ status(): number; json(): Promise<unknown>; text(): Promise<string> }>): Promise<number> {
  const r = await res
  expect(r.status(), await r.text()).toBe(201)
  return ((await r.json()) as { id: number }).id
}

export const password = 'e2e password'

export async function createUser(api: APIRequestContext, login: string, grants: { role: string; unit: number | null }[]) {
  const id = await created(api.post('/api/users', { data: { login, name: login, password } }))
  for (const g of grants) {
    const [product, role] = g.role.split('.')
    await created(api.post(`/api/users/${id}/roles`, { data: { product, role, org_unit_id: g.unit } }))
  }
  return id
}

export function employee(code: string, unit: number, extra: Record<string, unknown> = {}) {
  return {
    code,
    full_name: `Nhân viên ${code}`,
    date_of_birth: null,
    gender: null,
    phone: null,
    email: null,
    address: null,
    org_unit_id: unit,
    manager_id: null,
    user_login: null,
    hire_date: '2026-01-05',
    termination_date: null,
    ...extra,
  }
}
