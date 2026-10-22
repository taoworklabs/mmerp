import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { Badge } from '../../primitives/feedback/Badge.tsx'
import { Menu } from '../../primitives/overlay/Menu.tsx'
import { DataTable, type Column, type DataTableLabels, type Selection } from './DataTable.tsx'

type Employee = { id: number; full_name: string; code: string; org_unit: string; hire_date: string; status: 'active' | 'terminated' }

const columns: Column<Employee>[] = [
  { key: 'full_name', header: 'Họ tên', role: 'title', kind: 'person', sortable: true, render: (e) => e.full_name },
  { key: 'code', header: 'Mã', role: 'meta', kind: 'code', sortable: true, render: (e) => e.code },
  { key: 'org_unit', header: 'Phòng ban', role: 'meta', render: (e) => e.org_unit },
  { key: 'hire_date', header: 'Ngày vào làm', role: 'hidden', kind: 'date', sortable: true, render: (e) => e.hire_date },
  {
    key: 'status',
    header: 'Trạng thái',
    role: 'status',
    kind: 'status',
    render: (e) => <Badge color={e.status === 'active' ? 'success' : 'gray'}>{e.status === 'active' ? 'Đang làm' : 'Đã nghỉ'}</Badge>,
  },
]

const rows: Employee[] = [
  { id: 1, full_name: 'Nguyễn Văn A', code: 'NV001', org_unit: 'Kinh doanh', hire_date: '2023-01-10', status: 'active' },
  { id: 2, full_name: 'Trần Thị B', code: 'NV002', org_unit: 'Kế toán', hire_date: '2022-05-02', status: 'active' },
  { id: 3, full_name: 'Lê Văn C', code: 'NV003', org_unit: 'Nhân sự', hire_date: '2021-09-15', status: 'terminated' },
]

const longName = 'Nguyễn Thị Phương Thảo – Trưởng phòng Kế toán Tổng hợp, Chi nhánh Thành phố Hồ Chí Minh, phụ trách quyết toán thuế thu nhập cá nhân và bảo hiểm xã hội cho toàn bộ nhân viên khối văn phòng'

const labels: DataTableLabels = { actions: 'Thao tác khác', selectAll: 'Chọn mọi dòng', selectRow: 'Chọn dòng', resize: (c: string) => `Đổi độ rộng cột ${c}` }

const meta: Meta<typeof DataTable<Employee>> = {
  title: 'Blocks/DataTable',
  component: DataTable,
  args: { columns, rows, rowKey: (e: Employee) => e.id, label: 'Danh sách nhân viên', labels },
}

export default meta

type Story = StoryObj<typeof DataTable<Employee>>

export const Basic: StoryObj<{ onLinkClick(): void }> = {
  args: { onLinkClick: fn() },
  render: (args) => (
    <DataTable
      columns={columns}
      rows={rows}
      rowKey={(e) => e.id}
      label="Danh sách nhân viên"
      labels={labels}
      renderLink={(row: Employee, children) => (
        // A real router Link intercepts the click itself (preventDefault, client-side routing);
        // without that here, the forwarded click would navigate the page away.
        <a
          href={`#/employees/${row.id}`}
          onClick={(e) => {
            e.preventDefault()
            args.onLinkClick()
          }}
        >
          {children}
        </a>
      )}
    />
  ),
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    // Clicking a plain cell (not the link itself) still runs the rendered <a>'s own onClick: the
    // kit forwards the click to it instead of navigating itself.
    await userEvent.click(canvas.getByText('Kinh doanh'))
    expect(args.onLinkClick).toHaveBeenCalledTimes(1)
  },
}

export const RowClickWithoutLink: StoryObj<{ onInnerLinkClick(): void }> = {
  tags: ['!dev', '!autodocs'],
  args: { onInnerLinkClick: fn() },
  render: (args) => {
    // No renderLink here: the second column's own <a> must never be the one the row forwards a
    // plain-cell click to, since without renderLink the row shouldn't forward a click at all.
    const linkColumns: Column<Employee>[] = [
      columns[0]!,
      {
        key: 'codeLink',
        header: 'Mã',
        role: 'meta',
        render: (e) => (
          <a
            href="#"
            onClick={(ev) => {
              ev.preventDefault()
              args.onInnerLinkClick()
            }}
          >
            {e.code}
          </a>
        ),
      },
      columns[2]!,
    ]
    return <DataTable columns={linkColumns} rows={rows} rowKey={(e) => e.id} label="Danh sách nhân viên" labels={labels} />
  },
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByText('Kinh doanh'))
    expect(args.onInnerLinkClick).not.toHaveBeenCalled()
  },
}

export const RowClickSkipsOtherColumns: StoryObj<{ onFirstLinkClick(): void; onOtherLinkClick(): void }> = {
  tags: ['!dev', '!autodocs'],
  args: { onFirstLinkClick: fn(), onOtherLinkClick: fn() },
  render: (args) => {
    // With renderLink, the row forwards a plain-cell click to the first (sticky) cell's own link
    // only — never to another column's unrelated <a>, even though one exists in the row too.
    const linkColumns: Column<Employee>[] = [
      columns[0]!,
      {
        key: 'codeLink',
        header: 'Mã',
        role: 'meta',
        render: (e) => (
          <a
            href="#"
            onClick={(ev) => {
              ev.preventDefault()
              args.onOtherLinkClick()
            }}
          >
            {e.code}
          </a>
        ),
      },
      columns[2]!,
    ]
    return (
      <DataTable
        columns={linkColumns}
        rows={rows}
        rowKey={(e) => e.id}
        label="Danh sách nhân viên"
        labels={labels}
        renderLink={(row: Employee, children) => (
          <a
            href={`#/employees/${row.id}`}
            onClick={(e) => {
              e.preventDefault()
              args.onFirstLinkClick()
            }}
          >
            {children}
          </a>
        )}
      />
    )
  },
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByText('Kinh doanh'))
    expect(args.onFirstLinkClick).toHaveBeenCalledTimes(1)
    expect(args.onOtherLinkClick).not.toHaveBeenCalled()
  },
}

function SortableDemo({ onSort }: { onSort(sort: string): void }) {
  const [sort, setSort] = useState<string | undefined>()
  return (
    <DataTable
      columns={columns}
      rows={rows}
      rowKey={(e) => e.id}
      label="Danh sách nhân viên"
      labels={labels}
      sort={sort}
      onSort={(s) => {
        setSort(s)
        onSort(s)
      }}
    />
  )
}

export const Sortable: StoryObj<{ onSort(sort: string): void }> = {
  args: { onSort: fn() },
  render: (args) => <SortableDemo onSort={args.onSort} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const header = canvas.getByRole('columnheader', { name: 'Họ tên' })
    await userEvent.click(within(header).getByRole('button'))
    expect(args.onSort).toHaveBeenCalledWith('full_name')
    await userEvent.click(within(header).getByRole('button'))
    expect(args.onSort).toHaveBeenCalledWith('-full_name')
  },
}

function SelectableDemo() {
  const [selected, setSelected] = useState<ReadonlySet<string | number>>(new Set())
  const selection: Selection = { selected, onChange: setSelected }
  return <DataTable columns={columns} rows={rows} rowKey={(e) => e.id} label="Danh sách nhân viên" labels={labels} selection={selection} />
}

export const Selectable: Story = {
  render: () => <SelectableDemo />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const selectAll = canvas.getByRole('checkbox', { name: labels.selectAll })
    const rowCheckboxes = canvas.getAllByRole('checkbox', { name: labels.selectRow })

    await userEvent.click(rowCheckboxes[0]!)
    expect(selectAll).toHaveAttribute('aria-checked', 'mixed')

    await userEvent.click(selectAll)
    for (const box of canvas.getAllByRole('checkbox', { name: labels.selectRow })) expect(box).toHaveAttribute('aria-checked', 'true')

    await userEvent.click(selectAll)
    for (const box of canvas.getAllByRole('checkbox', { name: labels.selectRow })) expect(box).toHaveAttribute('aria-checked', 'false')
  },
}

function SelectableWithLinkAndClickDemo({ onLinkClick, onRowClick }: { onLinkClick(): void; onRowClick(row: Employee): void }) {
  const [selected, setSelected] = useState<ReadonlySet<string | number>>(new Set())
  const selection: Selection = { selected, onChange: setSelected }
  return (
    <DataTable
      columns={columns}
      rows={rows}
      rowKey={(e) => e.id}
      label="Danh sách nhân viên"
      labels={labels}
      selection={selection}
      onRowClick={onRowClick}
      renderLink={(row, children) => (
        <a
          href={`#/employees/${row.id}`}
          onClick={(e) => {
            e.preventDefault()
            onLinkClick()
          }}
        >
          {children}
        </a>
      )}
    />
  )
}

export const SelectableWithLinkAndRowClick: StoryObj<{ onLinkClick(): void; onRowClick(row: Employee): void }> = {
  tags: ['!dev', '!autodocs'],
  args: { onLinkClick: fn(), onRowClick: fn() },
  render: (args) => <SelectableWithLinkAndClickDemo onLinkClick={args.onLinkClick} onRowClick={args.onRowClick} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const rowCheckboxes = canvas.getAllByRole('checkbox', { name: labels.selectRow })

    // The checkbox's own label and control (span[role=checkbox]) both sit outside the row click
    // guard's old selector: clicking either toggled selection *and* ran the row's link/onRowClick.
    await userEvent.click(rowCheckboxes[0]!)
    expect(rowCheckboxes[0]).toHaveAttribute('aria-checked', 'true')
    expect(args.onLinkClick).not.toHaveBeenCalled()
    expect(args.onRowClick).not.toHaveBeenCalled()
  },
}

export const Resizable: StoryObj<{ onResize(key: string, width: number): void }> = {
  args: { onResize: fn() },
  render: (args) => <DataTable columns={columns} rows={rows} rowKey={(e) => e.id} label="Danh sách nhân viên" labels={labels} onResize={args.onResize} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const handle = canvas.getByRole('separator', { name: labels.resize('Mã') })
    const th = handle.closest('th')!
    const width = th.getBoundingClientRect().width

    handle.focus()
    await userEvent.keyboard('{ArrowRight}')
    expect(args.onResize).toHaveBeenCalledTimes(1)
    expect(args.onResize).toHaveBeenCalledWith('code', Math.round(width + 16))
  },
}

export const ResizeEdgeCases: StoryObj<{ onResize(key: string, width: number): void }> = {
  tags: ['!dev', '!autodocs'],
  args: { onResize: fn() },
  render: (args) => <DataTable columns={columns} rows={rows} rowKey={(e) => e.id} label="Danh sách nhân viên" labels={labels} onResize={args.onResize} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const handle = canvas.getByRole('separator', { name: labels.resize('Mã') })
    const down = (clientX: number) => handle.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, cancelable: true, clientX, pointerId: 1 }))
    const move = (clientX: number) => window.dispatchEvent(new PointerEvent('pointermove', { bubbles: true, clientX, pointerId: 1 }))
    const up = (clientX: number) => window.dispatchEvent(new PointerEvent('pointerup', { bubbles: true, clientX, pointerId: 1 }))
    const cancel = () => window.dispatchEvent(new PointerEvent('pointercancel', { bubbles: true, pointerId: 1 }))

    // A pointer down/up without any movement between them is a click, not a drag.
    down(100)
    up(100)
    expect(args.onResize).not.toHaveBeenCalled()

    // A cancelled gesture (e.g. a touch scroll takes over) removes the listeners instead of
    // committing the drag: a later pointerup must then do nothing either.
    down(100)
    move(140)
    cancel()
    up(140)
    expect(args.onResize).not.toHaveBeenCalled()
  },
}

export const Banded: Story = {
  args: {
    // bandStart never fires at index 0 (no column to its left to compare against), so the
    // identifier column here stays outside the band, as it would in a real column set.
    columns: [
      { key: 'full_name', header: 'Họ tên', role: 'title', kind: 'person', render: (e) => e.full_name },
      { key: 'code', header: 'Mã', role: 'meta', kind: 'code', band: 'Liên hệ', render: (e) => e.code },
      { key: 'org_unit', header: 'Phòng ban', role: 'meta', band: 'Liên hệ', render: (e) => e.org_unit },
    ],
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('columnheader', { name: 'Liên hệ' })).toHaveAttribute('colspan', '2')
  },
}

export const GroupedWithTotals: Story = {
  args: {
    group: {
      of: (e: Employee) => e.org_unit,
      total: (groupRows: Employee[], name: string) => ({ ...groupRows[0]!, full_name: `Tổng ${name}`, code: '', hire_date: '' }),
    },
  },
}

export const Footer: Story = {
  args: { footer: { id: 0, full_name: 'Tổng', code: '', org_unit: '', hire_date: '', status: 'active' } },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByText('Tổng').closest('tfoot')).not.toBeNull()
  },
}

export const RowMenu: StoryObj<{ onArchive(employee: Employee): void }> = {
  args: { onArchive: fn() },
  render: (args) => (
    <DataTable
      columns={columns}
      rows={rows}
      rowKey={(e) => e.id}
      label="Danh sách nhân viên"
      labels={labels}
      menu={(row: Employee) => (
        <>
          <Menu.Item onClick={() => args.onArchive(row)}>Lưu trữ</Menu.Item>
          <Menu.Item color="danger">Xóa</Menu.Item>
        </>
      )}
    />
  ),
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getAllByRole('button', { name: labels.actions })[0]!
    await userEvent.click(trigger)
    const body = within(canvasElement.ownerDocument.body)
    await userEvent.click(await body.findByRole('menuitem', { name: 'Lưu trữ' }))
    expect(args.onArchive).toHaveBeenCalledTimes(1)
  },
}

export const Cards: Story = {
  args: { cards: true },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('list')).not.toBeNull()
  },
}

export const LongFirstCell: Story = {
  render: () => (
    <div style={{ width: 500 }}>
      <DataTable
        columns={columns}
        rows={[{ ...rows[0]!, full_name: longName }]}
        rowKey={(e) => e.id}
        label="Danh sách nhân viên"
        labels={labels}
      />
    </div>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const cell = canvas.getByText(longName).closest('td')!
    expect(cell.scrollHeight).toBeLessThanOrEqual(41)
  },
}

export const MaxHeight: Story = {
  args: {
    rows: Array.from({ length: 40 }, (_, i) => ({
      id: i + 1,
      full_name: `Nhân viên ${i + 1}`,
      code: `NV${String(i + 1).padStart(3, '0')}`,
      org_unit: 'Kinh doanh',
      hire_date: '2023-01-10',
      status: 'active' as const,
    })),
    maxHeight: '300px',
  },
  play: async ({ canvasElement }) => {
    // Base UI's own fixed class on the scrollable viewport, not a CSS-module hash: stable to query.
    const viewport = canvasElement.querySelector<HTMLElement>('.base-ui-disable-scrollbar')!
    expect(viewport.scrollHeight).toBeGreaterThan(viewport.clientHeight)

    const header = canvasElement.querySelector('thead')!
    const headerTopBefore = header.getBoundingClientRect().top

    viewport.scrollTop = viewport.scrollHeight
    await waitFor(() => expect(viewport.scrollTop).toBeGreaterThan(0))

    const lastRow = within(canvasElement).getByText('Nhân viên 40').closest('tr')!
    await waitFor(() => expect(lastRow.getBoundingClientRect().bottom).toBeLessThanOrEqual(viewport.getBoundingClientRect().bottom + 1))

    // The sticky header stays docked at the viewport's top instead of scrolling away with the rows.
    expect(Math.abs(header.getBoundingClientRect().top - headerTopBefore)).toBeLessThanOrEqual(1)

    // Every header cell, the identifier's too, paints over the rows scrolling under it.
    for (const th of header.querySelectorAll('th')) {
      const r = th.getBoundingClientRect()
      expect(th.contains(document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2))).toBe(true)
    }
  },
}

type EmptyRow = { id: number; name: string | null | undefined; amount: number | string }

export const EmptyCells: StoryObj = {
  render: () => {
    const emptyColumns: Column<EmptyRow>[] = [
      { key: 'name', header: 'Tên', role: 'title', render: (r) => r.name },
      { key: 'amount', header: 'Số tiền', role: 'meta', kind: 'number', render: (r) => r.amount },
    ]
    const emptyRows: EmptyRow[] = [
      { id: 1, name: '', amount: 0 },
      { id: 2, name: null, amount: 5 },
      { id: 3, name: undefined, amount: '0' },
      { id: 4, name: 'Bình', amount: 5 },
    ]
    return (
      <DataTable
        columns={emptyColumns}
        rows={emptyRows}
        rowKey={(r) => r.id}
        label="Dòng có ô trống"
        labels={labels}
        footer={{ id: 0, name: '', amount: 5 }}
      />
    )
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    // Row 1: blank name and zero amount; row 2: null name; row 3: undefined name and '0' amount
    // (as a string, e.g. from a decimal field) — every one of those five cells shows a dash.
    expect(canvas.getAllByText('–')).toHaveLength(5)
    // The footer's blank name cell stays blank instead of a dash.
    const footerCells = canvasElement.querySelectorAll('tfoot td')
    expect(footerCells[0]!.textContent).toBe('')
  },
}
