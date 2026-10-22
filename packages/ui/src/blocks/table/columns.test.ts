import { describe, expect, it } from 'vitest'
import { bandStart, bandWidth, clampWidth, grouped, type Column } from './columns.ts'

type Row = { id: number; team: string }

describe('grouped', () => {
  it('keeps runs in order and splits on a name change', () => {
    const rows: Row[] = [
      { id: 1, team: 'A' },
      { id: 2, team: 'A' },
      { id: 3, team: 'B' },
      { id: 4, team: 'A' },
    ]
    const runs = grouped(rows, (r) => r.team)
    expect(runs).toEqual([
      { name: 'A', rows: [rows[0], rows[1]] },
      { name: 'B', rows: [rows[2]] },
      { name: 'A', rows: [rows[3]] },
    ])
  })

  it('makes one unnamed run without a grouping function', () => {
    const rows: Row[] = [{ id: 1, team: 'A' }, { id: 2, team: 'B' }]
    expect(grouped(rows)).toEqual([{ name: '', rows }])
  })
})

describe('clampWidth', () => {
  it('clamps to 48-800', () => {
    expect(clampWidth(10)).toBe(48)
    expect(clampWidth(1000)).toBe(800)
    expect(clampWidth(200)).toBe(200)
  })
})

describe('bandWidth / bandStart', () => {
  const columns: Column<Row>[] = [
    { key: 'a', header: 'A', role: 'meta', render: () => '', band: 'X' },
    { key: 'b', header: 'B', role: 'meta', render: () => '', band: 'X' },
    { key: 'c', header: 'C', role: 'meta', render: () => '' },
  ]

  it('counts consecutive columns of a band', () => {
    expect(bandWidth(columns, 0)).toBe(2)
    expect(bandWidth(columns, 2)).toBe(1)
  })

  it('marks the first column of a band, not the ones after it', () => {
    expect(bandStart(columns, 0)).toBe(false)
    expect(bandStart(columns, 1)).toBe(false)
    expect(bandStart(columns, 2)).toBe(true)
  })
})
