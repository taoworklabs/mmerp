import { describe, expect, it } from 'vitest'
import { safeNext } from './login'

const origin = 'http://erp.local'
const next = (v: string | null) => safeNext(v, origin)

describe('safeNext', () => {
  it('keeps a local path', () => expect(next('/hrm/employees?page=2')).toBe('/hrm/employees?page=2'))
  it('rejects other sites', () => {
    for (const v of ['https://evil.example', '//evil.example', '/\\evil.example', '/\t/evil.example', '/\n/evil.example', 'hrm', ''])
      expect(next(v)).toBeNull()
  })
  it('collapses a pathname that would leave the site', () => expect(next('/.//evil.example/x')).toBe('/evil.example/x'))
  it('never returns to the login page', () => expect(next('/login?next=/x')).toBe('/'))
  it('accepts no value', () => expect(next(null)).toBeNull())
})
