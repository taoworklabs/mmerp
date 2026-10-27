import type { TFunction } from 'i18next'
import { ApiError } from '@/shared/api/error'
import { errorText } from '@/shared/i18n'

// passwordErrorText gives one line per broken password rule, or the plain error text for any other error.
export function passwordErrorText(t: TFunction, err: unknown): string[] {
  if (err instanceof ApiError && err.code === 'password_weak' && Array.isArray(err.params.rules)) {
    return err.params.rules.map((rule) => t(`shared.error.password_rule.${String(rule)}`, { min: err.params.min }))
  }
  return [errorText(t, err)]
}
