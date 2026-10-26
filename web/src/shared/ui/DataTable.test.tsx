import type { DataTableProps } from '@mmerp/ui'
import { renderToStaticMarkup } from 'react-dom/server'
import { expect, test, vi } from 'vitest'
import { DataTable, type Column } from './DataTable'

// The kit block is replaced by a stub that keeps the props it was given, so the test can fire its callbacks.
const kit = vi.hoisted(() => ({ props: undefined as DataTableProps<unknown> | undefined }))
vi.mock('@mmerp/ui', async (original) => ({
  ...(await original<typeof import('@mmerp/ui')>()),
  DataTable: (props: DataTableProps<unknown>) => {
    kit.props = props
    return null
  },
}))

type Row = { id: number; name: string; code: string }
const columns: Column<Row>[] = [
  { key: 'name', header: 'Name', role: 'title', render: (r) => r.name },
  { key: 'code', header: 'Code', role: 'meta', kind: 'code', render: (r) => r.code },
]

test('reports a resized column width to the list view', () => {
  const widths: Record<string, number> = {}
  renderToStaticMarkup(<DataTable label="Rows" columns={columns} rows={[{ id: 1, name: 'A', code: 'X' }]} rowKey={(r) => r.id} onResize={(key, w) => (widths[key] = w)} />)
  kit.props?.onResize?.('name', 180)
  expect(widths).toEqual({ name: 180 })
})
