import type { ReactNode } from 'react'

// Role decides where a column goes on the card list below 1024px.
export type ColumnRole = 'title' | 'status' | 'meta' | 'hidden'

// Kind is what a column holds; it sets the column's look and width (see looks). The first column
// keeps the identifier's look whatever its kind.
export type ColumnKind = 'text' | 'person' | 'code' | 'date' | 'number' | 'status'

export type Column<T> = {
  key: string // also the sort key sent to the API
  header: string
  render: (row: T) => ReactNode
  role: ColumnRole
  // Left out for text.
  kind?: ColumnKind
  sortable?: boolean
  // Columns sharing a band sit under one heading, the band set off by a rule on its left.
  band?: string
  // A result column (net pay, cost): bold.
  strong?: boolean
  // Width in pixels, from the user's list settings; the table sizes the column otherwise.
  width?: number
}

// grouped splits rows into runs of the same group, in order; one run without a name when not grouped.
export function grouped<T>(rows: T[], of?: (row: T) => string): { name: string; rows: T[] }[] {
  const out: { name: string; rows: T[] }[] = []
  for (const row of rows) {
    const name = of ? of(row) : ''
    const last = out[out.length - 1]
    if (last && last.name === name) last.rows.push(row)
    else out.push({ name, rows: [row] })
  }
  return out
}

// bandStart: the column opens a band (or leaves one), so it carries the rule on its left.
export function bandStart<T>(columns: Column<T>[], i: number) {
  return i > 0 && columns[i - 1]?.band !== columns[i]?.band
}

export function bandWidth<T>(columns: Column<T>[], i: number) {
  let n = 1
  // Bounded by length: past the array, every column's band reads undefined, which would
  // otherwise match a bandless column (undefined === undefined) forever.
  while (i + n < columns.length && columns[i + n]?.band === columns[i]?.band) n++
  return n
}

// How each kind looks: tabular figures, the secondary colour, right alignment, and whether it stays
// on one line; only text wraps when the table runs out of room.
type Look = { tabular?: boolean; secondary?: boolean; right?: boolean; fit: boolean }
export const looks: Record<ColumnKind, Look> = {
  text: { fit: false },
  person: { fit: true },
  code: { tabular: true, secondary: true, fit: true },
  date: { tabular: true, secondary: true, fit: true },
  number: { tabular: true, right: true, fit: true },
  status: { fit: true },
}
export const look = <T,>(c: Column<T>) => looks[c.kind ?? 'text']

// A column the user has not resized stays on one line unless it is text. The first column is
// sticky and on one line anyway; marking it fit too, while a text column after it can give room
// instead, keeps a block inside it (a name with an avatar) at its full width rather than truncated.
export function fits<T>(columns: Column<T>[], i: number) {
  const c = columns[i]
  if (!c || c.width !== undefined) return false
  return look(c).fit || (i === 0 && columns.some((o, j) => j > 0 && !look(o).fit))
}

export const minWidth = 48
export const maxWidth = 800
export const clampWidth = (w: number) => Math.round(Math.min(maxWidth, Math.max(minWidth, w)))
