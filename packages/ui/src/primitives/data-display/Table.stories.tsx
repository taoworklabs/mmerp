import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { Table } from './Table.tsx'

const meta: Meta<typeof Table> = {
  title: 'Data display/Table',
  component: Table,
}

export default meta

type Story = StoryObj<typeof Table>

function Demo() {
  return (
    <Table>
      <Table.Thead>
        <Table.Tr>
          <Table.Th>Name</Table.Th>
          <Table.Th>Role</Table.Th>
        </Table.Tr>
      </Table.Thead>
      <Table.Tbody>
        <Table.Tr>
          <Table.Td>Nguyễn Văn A</Table.Td>
          <Table.Td>Developer</Table.Td>
        </Table.Tr>
        <Table.Tr>
          <Table.Td>Trần Thị B</Table.Td>
          <Table.Td>Designer</Table.Td>
        </Table.Tr>
      </Table.Tbody>
    </Table>
  )
}

export const Default: Story = {
  render: () => <Demo />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const header = canvas.getByRole('columnheader', { name: 'Name' })

    const probe = document.createElement('div')
    probe.style.backgroundColor = 'var(--mmerp-color-table-header)'
    canvasElement.appendChild(probe)
    expect(getComputedStyle(header.closest('thead')!).backgroundColor).toBe(getComputedStyle(probe).backgroundColor)
    probe.remove()

    const row = canvas.getByText('Developer').closest('tr')!
    expect(row.getBoundingClientRect().height).toBe(40)
  },
}

export const HighlightOnHover: Story = {
  render: () => (
    <Table highlightOnHover>
      <Table.Tbody>
        <Table.Tr>
          <Table.Td>Row 1</Table.Td>
        </Table.Tr>
      </Table.Tbody>
    </Table>
  ),
}

export const StickyHeader: Story = {
  render: () => (
    <div style={{ height: 120, overflow: 'auto' }} tabIndex={0}>
      <Table stickyHeader>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Name</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {Array.from({ length: 10 }, (_, i) => (
            <Table.Tr key={i}>
              <Table.Td>Row {i + 1}</Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </div>
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
          <Table.SortHeader label="Name" state={state('name')} onSort={() => toggle('name')} />
          <Table.SortHeader label="Role" state={state('role')} onSort={() => toggle('role')} />
          <Table.SortHeader label="Start date" state={state('start')} onSort={() => toggle('start')} />
        </Table.Tr>
      </Table.Thead>
    </Table>
  )
}

export const Sortable: Story = {
  render: () => <SortableDemo />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const name = canvas.getByRole('columnheader', { name: 'Name' })
    const role = canvas.getByRole('columnheader', { name: 'Role' })
    const start = canvas.getByRole('columnheader', { name: 'Start date' })

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
        <Table.Tr>
          <Table.Td look="numeric">1,234</Table.Td>
          <Table.Td look="secondary">INV-001</Table.Td>
        </Table.Tr>
      </Table.Tbody>
    </Table>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const numeric = canvas.getByText('1,234')
    expect(getComputedStyle(numeric).textAlign).toBe('right')
    expect(getComputedStyle(numeric).fontVariantNumeric).toContain('tabular-nums')

    const secondary = canvas.getByText('INV-001')
    const probe = document.createElement('div')
    probe.style.color = 'var(--mmerp-color-dimmed)'
    canvasElement.appendChild(probe)
    expect(getComputedStyle(secondary).color).toBe(getComputedStyle(probe).color)
    probe.remove()
  },
}

export const StickyFirstColumn: Story = {
  render: () => (
    <div style={{ width: 400, overflow: 'auto' }} data-testid="scrollbox" tabIndex={0}>
      <Table style={{ width: 1200 }}>
        <Table.Tbody>
          <Table.Tr>
            <Table.Td sticky>Row 1</Table.Td>
            {Array.from({ length: 6 }, (_, i) => (
              <Table.Td key={i}>Column {i + 2}</Table.Td>
            ))}
          </Table.Tr>
        </Table.Tbody>
      </Table>
    </div>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const box = canvas.getByTestId('scrollbox')
    const cell = canvas.getByText('Row 1')

    box.scrollLeft = 200
    await waitFor(() => expect(Math.abs(cell.getBoundingClientRect().left - box.getBoundingClientRect().left)).toBeLessThanOrEqual(1))
  },
}

export const SelectedRow: Story = {
  render: () => (
    <Table>
      <Table.Tbody>
        <Table.Tr aria-selected="true">
          <Table.Td>Selected row</Table.Td>
        </Table.Tr>
      </Table.Tbody>
    </Table>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const row = canvas.getByText('Selected row').closest('tr')!

    const probe = document.createElement('div')
    probe.style.backgroundColor = 'var(--mmerp-color-primary-light)'
    canvasElement.appendChild(probe)
    expect(getComputedStyle(row).backgroundColor).toBe(getComputedStyle(probe).backgroundColor)
    probe.remove()
  },
}
