import { describe, expect, it } from 'vitest'
import { parseList, serializeList } from './listParams'

const defaults = { filters: { q: '', status: '' }, sort: 'code', sorts: ['code', '-code', 'full_name', '-full_name'], pageSize: 50 }

describe('list params on the URL', () => {
  it('reads defaults from a bare URL and writes nothing back', () => {
    const p = parseList(new URLSearchParams(), defaults)
    expect(p).toEqual({ filters: { q: '', status: '' }, sort: 'code', page: 1, pageSize: 50 })
    expect(serializeList(p, defaults).toString()).toBe('')
  })

  it('round-trips filters, sort, page and page size', () => {
    const url = 'page=3&page_size=20&q=an&sort=-full_name&status=active'
    const p = parseList(new URLSearchParams(url), defaults)
    expect(p).toEqual({ filters: { q: 'an', status: 'active' }, sort: '-full_name', page: 3, pageSize: 20 })
    expect(serializeList(p, defaults).toString()).toBe(url)
  })

  it('falls back to the default sort when the URL names one the list does not offer', () => {
    expect(parseList(new URLSearchParams('sort=salary'), defaults).sort).toBe('code')
  })

  it('ignores a bad page, page size and unknown keys', () => {
    expect(parseList(new URLSearchParams('page=-2&page_size=7&evil=1'), defaults)).toEqual({
      filters: { q: '', status: '' },
      sort: 'code',
      page: 1,
      pageSize: 50,
    })
  })
})
