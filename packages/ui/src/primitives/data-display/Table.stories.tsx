import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { Table, type TableAppearance, type TableProps } from './Table.tsx'
import { tableEmployees } from './tableSamples.ts'
import { ScrollArea } from '../layout/ScrollArea.tsx'

const meta: Meta<typeof Table> = {
  title: 'Data display/Table',
  component: Table,
}

export default meta

type Story = StoryObj<typeof Table>

function Demo(props: TableProps) {
  return (
    <Table {...props}>
      <Table.Thead>
        <Table.Tr>
          <Table.Th>Họ tên</Table.Th>
          <Table.Th>Chức danh</Table.Th>
        </Table.Tr>
      </Table.Thead>
      <Table.Tbody>
        {tableEmployees.map((employee) => (
          <Table.Tr key={employee.id}>
            <Table.Td>{employee.full_name}</Table.Td>
            <Table.Td>{employee.role}</Table.Td>
          </Table.Tr>
        ))}
      </Table.Tbody>
    </Table>
  )
}

export const Default: Story = {
  render: () => <Demo />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const header = canvas.getByRole('columnheader', { name: 'Họ tên' })

    const probe = document.createElement('div')
    probe.style.backgroundColor = 'var(--mmerp-color-table-header)'
    canvasElement.appendChild(probe)
    expect(getComputedStyle(header.closest('thead')!).backgroundColor).toBe(getComputedStyle(probe).backgroundColor)
    probe.remove()

    const row = canvas.getAllByText('Kế toán viên')[0]!.closest('tr')!
    expect(row.getBoundingClientRect().height).toBe(40)
  },
}

export const HighlightOnHover: Story = {
  render: () => <Demo highlightOnHover />,
}

export const StickyHeader: Story = {
  render: () => (
    <ScrollArea style={{ height: 360 }}>
      <Table stickyHeader>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Họ tên</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {Array.from({ length: 60 }, (_, i) => (
            <Table.Tr key={i}>
              <Table.Td>Dòng {i + 1}</Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </ScrollArea>
  ),
}

type SortDir = 'ascending' | 'descending'

function SortableDemo() {
  const [sort, setSort] = useState<{ key: string; dir: SortDir } | undefined>()
  const state = (key: string): SortDir | undefined => (sort?.key === key ? sort.dir : undefined)
  const toggle = (key: string) => setSort((s) => ({ key, dir: s?.key === key && s.dir === 'ascending' ? 'descending' : 'ascending' }))
  return (
    <Table>
      <Table.Thead>
        <Table.Tr>
          <Table.SortHeader label="Họ tên" state={state('name')} onSort={() => toggle('name')} />
          <Table.SortHeader label="Chức danh" state={state('role')} onSort={() => toggle('role')} />
          <Table.SortHeader label="Ngày vào làm" state={state('start')} onSort={() => toggle('start')} />
        </Table.Tr>
      </Table.Thead>
      <Table.Tbody>
        {[...tableEmployees].sort((a, b) => {
          if (!sort) return 0
          const key = sort.key === 'name' ? 'full_name' : sort.key === 'start' ? 'hire_date' : 'role'
          return a[key].localeCompare(b[key], 'vi') * (sort.dir === 'ascending' ? 1 : -1)
        }).map((employee) => (
          <Table.Tr key={employee.id}>
            <Table.Td>{employee.full_name}</Table.Td>
            <Table.Td>{employee.role}</Table.Td>
            <Table.Td look="secondary">{employee.hire_date}</Table.Td>
          </Table.Tr>
        ))}
      </Table.Tbody>
    </Table>
  )
}

export const Sortable: Story = {
  render: () => <SortableDemo />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const name = canvas.getByRole('columnheader', { name: 'Họ tên' })
    const role = canvas.getByRole('columnheader', { name: 'Chức danh' })
    const start = canvas.getByRole('columnheader', { name: 'Ngày vào làm' })

    await userEvent.click(within(name).getByRole('button'))
    expect(name).toHaveAttribute('aria-sort', 'ascending')
    expect(role).not.toHaveAttribute('aria-sort')
    expect(start).not.toHaveAttribute('aria-sort')

    await userEvent.keyboard('{Enter}')
    expect(name).toHaveAttribute('aria-sort', 'descending')
    expect(role).not.toHaveAttribute('aria-sort')
  },
}

export const CellLooks: Story = {
  render: () => (
    <Table>
      <Table.Tbody>
        {tableEmployees.map((employee, index) => (
          <Table.Tr key={employee.id}>
            <Table.Td>{employee.full_name}</Table.Td>
            <Table.Td look="numeric">{new Intl.NumberFormat('vi-VN').format(1234000 + index * 250000)}</Table.Td>
            <Table.Td look="secondary">HĐ-{String(employee.id).padStart(4, '0')}</Table.Td>
          </Table.Tr>
        ))}
      </Table.Tbody>
    </Table>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const numeric = canvas.getByText('1.234.000')
    expect(getComputedStyle(numeric).textAlign).toBe('right')
    expect(getComputedStyle(numeric).fontVariantNumeric).toContain('tabular-nums')

    const secondary = canvas.getByText('HĐ-0001')
    const probe = document.createElement('div')
    probe.style.color = 'var(--mmerp-color-dimmed)'
    canvasElement.appendChild(probe)
    expect(getComputedStyle(secondary).color).toBe(getComputedStyle(probe).color)
    probe.remove()
  },
}

export const StickyFirstColumn: Story = {
  render: () => (
    <ScrollArea style={{ width: 600, maxWidth: '100%', height: 360 }} data-testid="scrollbox">
      <Table style={{ width: 1200 }}>
        <Table.Thead>
          <Table.Tr>
            <Table.Th sticky>Họ tên</Table.Th>
            {['Mã', 'Chức danh', 'Phòng ban', 'Ngày vào làm', 'Trạng thái', 'Hợp đồng'].map((label) => (
              <Table.Th key={label}>{label}</Table.Th>
            ))}
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {tableEmployees.map((employee) => (
            <Table.Tr key={employee.id}>
              <Table.Td sticky>{employee.full_name}</Table.Td>
              {[employee.code, employee.role, employee.org_unit, employee.hire_date, employee.status === 'active' ? 'Đang làm' : 'Đã nghỉ', `HĐ-${String(employee.id).padStart(4, '0')}`].map((value, index) => (
                <Table.Td key={index}>{value}</Table.Td>
              ))}
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </ScrollArea>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const box = canvas.getByTestId('scrollbox').querySelector<HTMLElement>('[data-scroll-area-viewport]')!
    const cell = canvas.getByText(tableEmployees[0]!.full_name)

    box.scrollLeft = 200
    await waitFor(() => expect(Math.abs(cell.getBoundingClientRect().left - box.getBoundingClientRect().left)).toBeLessThanOrEqual(1))

    // The sticky header cell still reads as header, not as a plain elevated surface.
    const header = canvas.getByRole('columnheader', { name: 'Họ tên' })
    const probe = document.createElement('div')
    probe.style.backgroundColor = 'var(--mmerp-color-table-header)'
    canvasElement.appendChild(probe)
    expect(getComputedStyle(header).backgroundColor).toBe(getComputedStyle(probe).backgroundColor)
    probe.remove()
  },
}

export const SelectedRow: Story = {
  render: () => (
    <Table>
      <Table.Tbody>
        {tableEmployees.map((employee, index) => (
          <Table.Tr key={employee.id} aria-selected={index === 3 ? true : undefined}>
            <Table.Td sticky>{index === 3 ? 'Dòng đang chọn' : employee.full_name}</Table.Td>
            <Table.Td>{employee.role}</Table.Td>
          </Table.Tr>
        ))}
      </Table.Tbody>
    </Table>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const row = canvas.getByText('Dòng đang chọn').closest('tr')!
    const stickyCell = canvas.getByText('Dòng đang chọn').closest('td')!

    const probe = document.createElement('div')
    probe.style.backgroundColor = 'var(--mmerp-color-primary-light)'
    canvasElement.appendChild(probe)
    expect(getComputedStyle(row).backgroundColor).toBe(getComputedStyle(probe).backgroundColor)
    // The sticky cell paints its own background, so it must match the row's tint, not the
    // plain elevated surface it normally uses to stay opaque while scrolling.
    expect(getComputedStyle(stickyCell).backgroundColor).toBe(getComputedStyle(probe).backgroundColor)
    probe.remove()
  },
}

export const Striped: Story = { render: () => <Demo appearance={{ striped: true }} /> }
export const Grid: Story = { render: () => <Demo appearance={{ borders: 'grid', withBorder: true }} /> }
export const ColumnBorders: Story = { render: () => <Demo appearance={{ borders: 'columns', withBorder: true }} /> }
export const Borderless: Story = { render: () => <Demo appearance={{ borders: 'none', headerColor: 'plain' }} /> }
export const DashedBorders: Story = { render: () => <Demo appearance={{ borderStyle: 'dashed', borderColor: 'primary' }} /> }
export const DottedGrid: Story = { render: () => <Demo appearance={{ borders: 'grid', borderStyle: 'dotted', withBorder: true }} /> }
export const StrongBorder: Story = { render: () => <Demo appearance={{ borders: 'grid', borderWidth: 2, borderColor: 'gray', withBorder: true }} /> }
export const TintedHeader: Story = { render: () => <Demo appearance={{ headerColor: 'primary' }} /> }
export const TintedStripes: Story = { render: () => <Demo highlightOnHover appearance={{ headerColor: 'primary', striped: true, stripeColor: 'primary', hoverColor: 'primary', borderColor: 'primary' }} /> }
export const Compact: Story = { render: () => <Demo appearance={{ density: 'compact', striped: true }} /> }
export const Comfortable: Story = { render: () => <Demo appearance={{ density: 'comfortable', borders: 'none', striped: true }} /> }

export const AppearancePlayground: StoryObj<TableAppearance> = {
  args: { borders: 'grid', withBorder: true, borderColor: 'primary', borderStyle: 'solid', borderWidth: 1, headerColor: 'primary', striped: true, stripeColor: 'gray', hoverColor: 'primary', density: 'standard' },
  argTypes: {
    borders: { control: 'select', options: ['rows', 'columns', 'grid', 'none'] },
    withBorder: { control: 'boolean' },
    borderColor: { control: 'select', options: ['gray', 'primary', 'info', 'success', 'warning', 'danger', 'discovery'] },
    borderStyle: { control: 'select', options: ['solid', 'dashed', 'dotted'] },
    borderWidth: { control: 'select', options: [1, 2] },
    headerColor: { control: 'select', options: ['neutral', 'plain', 'gray', 'primary', 'info', 'success', 'warning', 'danger', 'discovery'] },
    striped: { control: 'boolean' },
    stripeColor: { control: 'select', options: ['gray', 'primary', 'info', 'success', 'warning', 'danger', 'discovery'] },
    hoverColor: { control: 'select', options: ['gray', 'primary', 'info', 'success', 'warning', 'danger', 'discovery'] },
    density: { control: 'select', options: ['compact', 'standard', 'comfortable'] },
  },
  render: (appearance) => <Demo highlightOnHover appearance={appearance} />,
}
