// The Accounting API as the accounting area sees it.
import { productClient } from './client'

export { unwrap } from './client'
export const api = productClient<'/accounting/'>()
export { ApiError } from './error'
