import type { FieldValues, Path } from 'react-hook-form'
import type { ApiError } from '@/shared/api/error'

// ErrorFields maps an API error code to the field it belongs to. A field of a line holds `*`
// where the line goes, e.g. "lines.*.quantity"; the error names the line in its `line`
// parameter (`line_no` from the stock ledger).
export type ErrorFields<T extends FieldValues> = Record<string, Path<T> | `${string}.*.${string}`>

// fieldOfError is the field an API error shows at, or undefined for the form's banner.
export function fieldOfError<T extends FieldValues>(map: ErrorFields<T> | undefined, err: ApiError): Path<T> | undefined {
  const field = map && Object.hasOwn(map, err.code) ? map[err.code] : undefined
  if (!field?.includes('*')) return field as Path<T> | undefined
  const line = err.params.line ?? err.params.line_no
  if (typeof line !== 'number' || !Number.isInteger(line) || line < 1) return undefined
  return field.replace('*', String(line - 1)) as Path<T>
}
