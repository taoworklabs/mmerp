import { render } from 'vitest-browser-react'
import { page, userEvent } from 'vitest/browser'
import { expect, it } from 'vitest'
import { Table, type TableAppearance } from './Table.tsx'
import { DataTable } from '../../blocks/table/DataTable.tsx'
import '../../styles/index.css'

function Sample({ appearance }: { appearance?: TableAppearance }) {
  return <Table appearance={appearance} highlightOnHover stickyHeader>
    <Table.Thead><Table.Tr><Table.Th sticky>Name</Table.Th><Table.Th>Code</Table.Th></Table.Tr></Table.Thead>
    <Table.Tbody>{[1, 2, 3].map(id => <Table.Tr key={id} aria-selected={id === 3 || undefined}><Table.Td sticky>Person {id}</Table.Td><Table.Td>{id}</Table.Td></Table.Tr>)}</Table.Tbody>
  </Table>
}

it('composes border style, width and layout and removes every rule in borderless mode', async () => {
  const { container, rerender } = await render(<Sample appearance={{ borders: 'grid', withBorder: true, borderStyle: 'dashed', borderWidth: 2 }} />)
  const table = container.querySelector('table')!
  const cell = container.querySelector('td')!
  expect(getComputedStyle(table).borderTopWidth).toBe('2px')
  expect(getComputedStyle(cell).borderBottomStyle).toBe('dashed')
  expect(getComputedStyle(cell).borderRightWidth).toBe('2px')
  const headerRule = getComputedStyle(container.querySelector('th')!, '::after')
  expect(headerRule.borderBottomWidth).toBe('2px')
  expect(headerRule.borderBottomStyle).toBe('dashed')
  await rerender(<Sample appearance={{ borders: 'none', withBorder: false }} />)
  expect(getComputedStyle(table).borderTopStyle).toBe('none')
  expect(getComputedStyle(cell).borderBottomStyle).toBe('none')
  expect(getComputedStyle(cell).borderRightStyle).toBe('none')
})

it('keeps stripes and selection opaque on sticky cells, including on hover', async () => {
  const { container } = await render(<Sample appearance={{ striped: true, stripeColor: 'primary', headerColor: 'primary', hoverColor: 'primary' }} />)
  const rows = container.querySelectorAll('tbody tr')
  expect(getComputedStyle(rows[1]!).backgroundColor).toBe(getComputedStyle(rows[1]!.firstElementChild!).backgroundColor)
  expect(getComputedStyle(rows[1]!).backgroundColor).not.toBe(getComputedStyle(rows[0]!).backgroundColor)
  const selected = getComputedStyle(rows[2]!).backgroundColor
  await userEvent.hover(page.getByText('Person 3'))
  expect(getComputedStyle(rows[2]!).backgroundColor).toBe(selected)
  expect(getComputedStyle(rows[2]!.firstElementChild!).backgroundColor).toBe(selected)
  expect(getComputedStyle(container.querySelector('thead')!).backgroundColor).toBe(getComputedStyle(container.querySelector('th')!).backgroundColor)
})

it('applies appearance to DataTable without leaking props to the DOM', async () => {
  await page.viewport(1280, 800)
  const { container } = await render(<DataTable
    appearance={{ borders: 'grid', withBorder: false, density: 'comfortable', borderStyle: 'dotted', headerColor: 'primary' }}
    columns={[{ key: 'name', header: 'Name', role: 'title', render: row => row.name }, { key: 'id', header: 'Code', role: 'meta', render: row => row.id }]}
    rows={[{ id: 1, name: 'An' }]} rowKey={row => row.id} label="People"
    labels={{ actions: 'Actions', selectAll: 'All', selectRow: 'Select', resize: name => name }}
  />)
  expect(container.querySelector('[appearance]')).toBeNull()
  expect(getComputedStyle(container.querySelector('[data-table]')!).borderTopStyle).toBe('none')
  expect(container.querySelector('tbody tr')!.getBoundingClientRect().height).toBe(48)
  expect(getComputedStyle(container.querySelector('td')!).borderRightStyle).toBe('dotted')
})
