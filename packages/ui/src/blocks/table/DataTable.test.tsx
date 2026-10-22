import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import { describe, expect, it } from 'vitest'
import { DataTable, type Column, type DataTableLabels } from './DataTable.tsx'
import '../../styles/index.css'

type Employee = { id: number; full_name: string; code: string; status: string }

const columns: Column<Employee>[] = [
  { key: 'full_name', header: 'Họ tên', role: 'title', kind: 'person', render: (e) => <span data-testid="title">{e.full_name}</span> },
  { key: 'code', header: 'Mã', role: 'meta', kind: 'code', render: (e) => e.code },
  { key: 'status', header: 'Trạng thái', role: 'status', render: (e) => <span data-testid="status">{e.status}</span> },
]

const rows: Employee[] = [{ id: 1, full_name: 'Nguyễn Văn A', code: 'NV001', status: 'Đang làm' }]

const labels: DataTableLabels = { actions: 'Thao tác khác', selectAll: 'Chọn mọi dòng', selectRow: 'Chọn dòng', resize: (c) => `Đổi độ rộng cột ${c}` }

describe('DataTable', () => {
  it('renders role="list" cards with the title left of the status column below 64em, and a table from it', async () => {
    const { container } = await render(<DataTable columns={columns} rows={rows} rowKey={(e) => e.id} label="Nhân viên" labels={labels} />)

    await page.viewport(800, 900)
    await expect.poll(() => container.querySelector('[role="list"]')).not.toBeNull()
    const title = container.querySelector('[data-testid="title"]') as HTMLElement
    const status = container.querySelector('[data-testid="status"]') as HTMLElement
    expect(title.getBoundingClientRect().left).toBeLessThan(status.getBoundingClientRect().left)

    await page.viewport(1300, 900)
    await expect.poll(() => container.querySelector('table')).not.toBeNull()
    await expect.poll(() => container.querySelector('[role="list"]')).toBeNull()
  })

  it('carries data-table, merges a caller className, and forwards ref to the root div', async () => {
    let rootEl: HTMLDivElement | null = null
    const { container } = await render(
      <DataTable
        columns={columns}
        rows={rows}
        rowKey={(e) => e.id}
        label="Nhân viên"
        labels={labels}
        className="extra"
        ref={(el) => {
          rootEl = el
        }}
      />,
    )

    await page.viewport(1300, 900)
    const root = container.firstElementChild as HTMLElement
    await expect.poll(() => root.hasAttribute('data-table')).toBe(true)
    expect(root.classList.contains('extra')).toBe(true)
    expect(rootEl).toBe(root)
  })
})
