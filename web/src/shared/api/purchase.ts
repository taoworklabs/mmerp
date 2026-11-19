// The Purchasing API as the purchase area sees it.
import { productClient } from './client'

export { unwrap } from './client'
export const api = productClient<'/purchase/'>()
export { ApiError } from './error'
