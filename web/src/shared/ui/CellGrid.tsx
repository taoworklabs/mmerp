import { Paper, ScrollArea, VisuallyHidden } from '@mantine/core'
import type { KeyboardEvent, ReactNode } from 'react'
import classes from './CellGrid.module.css'

export type GridAxis = { key: string; header: ReactNode; label: string }

type Props = {
  label: string // accessible name of the grid
  corner: string // header of the row-title column
  rows: GridAxis[]
  columns: GridAxis[]
  value: (row: string, column: string) => string
  // Without onChange the grid is read-only.
  onChange?: (row: string, column: string, value: string) => void
  invalid?: (value: string) => boolean
  // Given to the first invalid cell, so a form can focus it (fieldId of the grid's field).
  invalidId?: string
  // The id of the message explaining what is invalid, for the cells' aria-describedby.
  errorId?: string
  // A cell that takes no value is greyed and read-only; a value it holds anyway still shows.
  off?: (row: string, column: string) => boolean
  // A cell to look at again, with the reason: tinted, with the reason as its tooltip and read
  // out after the cell's value.
  warning?: (row: string, column: string) => string | undefined
  total?: { header: string; value: (row: string) => ReactNode }
}

const moves: Record<string, [number, number]> = { ArrowUp: [-1, 0], ArrowDown: [1, 0], Enter: [1, 0], ArrowLeft: [0, -1], ArrowRight: [0, 1] }

// CellGrid is a rows × columns grid of small cells, e.g. days worked per employee and day:
// the row titles stay in view when it scrolls sideways, and arrow keys move between cells.
export function CellGrid({ label, corner, rows, columns, value, onChange, invalid, invalidId, errorId, off, warning, total }: Props) {
  const first = invalidId && invalid ? firstInvalid(rows, columns, value, invalid, off) : null
  function move(e: KeyboardEvent<HTMLInputElement>, r: number, c: number) {
    const step = moves[e.key]
    if (!step) return
    const input = e.currentTarget
    // Left and right move only when the caret is already at the edge of the text.
    if (e.key === 'ArrowLeft' && input.selectionStart !== 0) return
    if (e.key === 'ArrowRight' && input.selectionEnd !== input.value.length) return
    // Skip the cells that take no value, up to the edge of the grid.
    const table = input.closest('table')
    for (let nr = r + step[0], nc = c + step[1]; nr >= 0 && nr < rows.length && nc >= 0 && nc < columns.length; nr += step[0], nc += step[1]) {
      const next = table?.querySelector<HTMLInputElement>(`input[data-cell="${nr}:${nc}"]`)
      if (!next) continue
      e.preventDefault()
      next.focus()
      next.select()
      return
    }
  }

  return (
    <Paper withBorder>
      <ScrollArea type="auto">
        <table className={classes.grid} aria-label={label}>
          <thead>
            <tr>
              <th scope="col" className={classes.sticky}>
                {corner}
              </th>
              {columns.map((c) => (
                <th key={c.key} scope="col">
                  {c.header}
                </th>
              ))}
              {total && (
                <th scope="col" className={classes.total}>
                  {total.header}
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {rows.map((r, ri) => (
              <tr key={r.key}>
                <th scope="row" className={classes.sticky}>
                  {r.header}
                </th>
                {columns.map((c, ci) => {
                  const isOff = off?.(r.key, c.key) ?? false
                  const v = value(r.key, c.key)
                  const warn = warning?.(r.key, c.key)
                  return (
                    <td key={c.key} className={[isOff && classes.off, warn && classes.warn].filter(Boolean).join(' ') || undefined} title={warn}>
                      {onChange && !isOff ? (
                        <input
                          className={classes.cell}
                          data-cell={`${ri}:${ci}`}
                          aria-label={`${r.label}, ${c.label}`}
                          id={first === `${r.key}:${c.key}` ? invalidId : undefined}
                          aria-invalid={invalid?.(v) || undefined}
                          aria-describedby={invalid?.(v) ? errorId : undefined}
                          inputMode="decimal"
                          maxLength={4}
                          value={v}
                          onChange={(e) => onChange(r.key, c.key, e.currentTarget.value)}
                          onKeyDown={(e) => move(e, ri, ci)}
                          aria-description={warn}
                        />
                      ) : (
                        <>
                          {v}
                          {warn && <VisuallyHidden>{warn}</VisuallyHidden>}
                        </>
                      )}
                    </td>
                  )
                })}
                {total && <td className={classes.total}>{total.value(r.key)}</td>}
              </tr>
            ))}
          </tbody>
        </table>
      </ScrollArea>
    </Paper>
  )
}

// firstInvalid is the key ("row:column") of the first invalid editable cell in reading order.
function firstInvalid(rows: GridAxis[], columns: GridAxis[], value: (r: string, c: string) => string, invalid: (v: string) => boolean, off?: (r: string, c: string) => boolean) {
  for (const r of rows) for (const c of columns) if (!off?.(r.key, c.key) && invalid(value(r.key, c.key))) return `${r.key}:${c.key}`
  return null
}
