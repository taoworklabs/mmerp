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

  it('keeps the identifier column on one line and scrolls a table too wide for its frame sideways', async () => {
    type Param = { key: string; value: string }
    const params: Column<Param>[] = [
      { key: 'key', header: 'Tham số', role: 'title', render: (p) => <span data-testid="key">{p.key}</span> },
      { key: 'value', header: 'Giá trị', role: 'meta', kind: 'number', render: (p) => p.value },
    ]
    const value = Array.from({ length: 8 }, (_, i) => `đến ${i + 1}.000.000: ${i + 5}%`).join(' · ')
    const { container } = await render(
      <div style={{ width: 600 }}>
        <DataTable columns={params} rows={[{ key: 'Biểu thuế thu nhập cá nhân lũy tiến', value }]} rowKey={(p) => p.key} label="Tham số" labels={labels} />
      </div>,
    )

    await page.viewport(1300, 900)
    await expect.poll(() => container.querySelector('[data-testid="key"]')).not.toBeNull()
    const key = container.querySelector('[data-testid="key"]') as HTMLElement
    const lineHeight = parseFloat(getComputedStyle(key).lineHeight)
    expect(key.getBoundingClientRect().height).toBeLessThanOrEqual(lineHeight)
    const viewport = container.querySelector('[data-scroll-area-viewport]') as HTMLElement
    expect(viewport.scrollWidth).toBeGreaterThan(viewport.clientWidth)
  })

  it('renders card lines as blocks below 64em, so a column that renders a block stays valid HTML', async () => {
    const blocky: Column<Employee>[] = [
      { key: 'full_name', header: 'Họ tên', role: 'title', render: (e) => <div>{e.full_name}</div> },
      { key: 'code', header: 'Mã', role: 'meta', render: (e) => <div>{e.code}</div> },
      { key: 'status', header: 'Trạng thái', role: 'status', render: (e) => <div>{e.status}</div> },
    ]
    const { container } = await render(<DataTable columns={blocky} rows={rows} rowKey={(e) => e.id} label="Nhân viên" labels={labels} />)

    await page.viewport(375, 800)
    await expect.poll(() => container.querySelector('[role="list"]')).not.toBeNull()
    expect(container.querySelectorAll('p div')).toHaveLength(0)
  })
})
