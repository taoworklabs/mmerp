import { describe, expect, it } from 'vitest'
import type { Column } from './DataTable'
import { shownColumns, viewQuery, withColumn } from './listView'

const col = (key: string): Column<unknown> => ({ key, header: key, role: 'meta', render: () => key })

describe('viewQuery', () => {
  it('drops the page and the open side panel and sorts the rest', () => {
    expect(viewQuery(new URLSearchParams('status=posted&page=3&peek=12&q=an'))).toBe('q=an&status=posted')
  })
})

describe('shownColumns', () => {
  const columns = [col('name'), col('code'), col('phone')]
  it('hides chosen columns but never the first, and applies widths', () => {
    const settings = { views: [], columns: [{ key: 'name', hidden: true, width: null }, { key: 'phone', hidden: true, width: null }, { key: 'code', hidden: false, width: 200 }] }
    expect(shownColumns(columns, settings).map((c) => [c.key, c.width])).toEqual([['name', undefined], ['code', 200]])
  })
  it('shows everything as designed without settings', () => {
    expect(shownColumns(columns, undefined).map((c) => c.key)).toEqual(['name', 'code', 'phone'])
  })
})

describe('withColumn', () => {
  it('adds a choice for a new column and changes an existing one', () => {
    const once = withColumn({ views: [], columns: [] }, 'phone', { hidden: true })
    expect(once.columns).toEqual([{ key: 'phone', hidden: true, width: null }])
    expect(withColumn(once, 'phone', { width: 160 }).columns).toEqual([{ key: 'phone', hidden: true, width: 160 }])
  })
})
