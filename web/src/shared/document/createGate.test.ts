import { describe, expect, it } from 'vitest'
import { createGate } from './CreatePage'

const query = (over: object) => ({ isPending: false, isError: false, data: undefined, ...over })

describe('createGate', () => {
  it('waits while the actions load', () => {
    expect(createGate(query({ isPending: true }), 'create')).toBe('pending')
  })
  it('shows the error when the actions fail', () => {
    expect(createGate(query({ isError: true }), 'create')).toBe('error')
  })
  it('forbids a user the API does not allow to create', () => {
    expect(createGate(query({ data: ['import'] }), 'create')).toBe('forbidden')
    expect(createGate(['create'], 'create_contract')).toBe('forbidden')
  })
  it('opens the form when the action is allowed, from a query or a loaded list', () => {
    expect(createGate(query({ data: ['create', 'import'] }), 'create')).toBe('ok')
    expect(createGate(['create_contract'], 'create_contract')).toBe('ok')
  })
})
