// What every failed API call throws: the stable code and parameters from the server.
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly params: Record<string, unknown>

  constructor(status: number, code: string, params: Record<string, unknown> = {}) {
    super(code)
    this.status = status
    this.code = code
    this.params = params
  }
}
