import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
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
