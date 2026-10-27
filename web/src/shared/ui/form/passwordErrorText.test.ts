import { describe, expect, it } from 'vitest'
import { ApiError } from '@/shared/api/error'
import { passwordErrorText } from './passwordErrorText'

// A stand-in for i18next: echoes the key and the min parameter.
const t = ((key: string | string[], o?: { min?: number }) => `${String(key)}|${o?.min ?? ''}`) as never

describe('passwordErrorText', () => {
  it('gives one line per broken rule', () => {
    const err = new ApiError(422, 'password_weak', { rules: ['min_length', 'digit'], min: 10 })
    expect(passwordErrorText(t, err)).toEqual(['shared.error.password_rule.min_length|10', 'shared.error.password_rule.digit|10'])
  })
  it('falls back to the error text for other errors', () => {
    expect(passwordErrorText(t, new ApiError(409, 'login_taken'))).toHaveLength(1)
  })
})
