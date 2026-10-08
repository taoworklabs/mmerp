import createClient, { type Client, type Middleware } from 'openapi-fetch'
import { endSession, onAuthzVersion, sessionSignal } from '@/shared/auth/session'
import { ApiError } from './error'
import type { paths } from './schema.gen'

// Wrong credentials answer 401 too; that is a form error, not an ended session.
const sessionless = new Set(['/auth/login'])

const session: Middleware = {
  onRequest({ request }) {
    return new Request(request, { signal: AbortSignal.any([request.signal, sessionSignal.signal]) })
  },
  onResponse({ response, schemaPath }) {
    onAuthzVersion(response.headers.get('X-Authz-Version'))
    if (response.status === 401 && !sessionless.has(schemaPath)) endSession()
    return response
  },
}

export const client = createClient<paths>({ baseUrl: '/api' })
client.use(session)

// Every product's routes live under /<product>/; core routes are the rest.
type ProductPrefix = '/hrm/'

// productClient is the shared client typed to one product's routes, so an area cannot call another's.
export type ProductPaths<P extends ProductPrefix> = { [K in keyof paths as K extends `${P}${string}` ? K : never]: paths[K] }
export type CorePaths = { [K in keyof paths as K extends `${ProductPrefix}${string}` ? never : K]: paths[K] }
export const productClient = <P extends ProductPrefix>() => client as unknown as Client<ProductPaths<P>>
export const coreClient = client as unknown as Client<CorePaths>

type Result<T> = { data?: T; error?: unknown; response: Response }

// unwrap returns the data of a call, or throws its error as an ApiError.
export async function unwrap<T>(call: Promise<Result<T>>): Promise<T> {
  const { data, error, response } = await call
  if (response.ok) return data as T
  const body = (error ?? {}) as { code?: string; params?: Record<string, unknown> }
  throw new ApiError(response.status, body.code ?? 'unexpected_error', body.params)
}
