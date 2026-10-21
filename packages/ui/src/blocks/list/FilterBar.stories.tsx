import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { Button } from '../../primitives/input/Button.tsx'
import { Select } from '../../primitives/input/Select.tsx'
import { BulkBar } from './BulkBar.tsx'
import { FilterBar } from './FilterBar.tsx'
import { FilterChip } from './FilterChip.tsx'

const statusData = [
  { value: 'open', label: 'Open' },
  { value: 'closed', label: 'Closed' },
]

const kindData = [
  { value: 'invoice', label: 'Invoice' },
  { value: 'quote', label: 'Quote' },
]

const meta: Meta<typeof FilterBar> = {
  title: 'Blocks/FilterBar',
  component: FilterBar,
}

export default meta

type Story = StoryObj<typeof FilterBar>

export const Default: Story = {
  args: {
    label: 'Filters',
    filters: (
      <>
        <FilterBar.Search aria-label="Search" placeholder="Search" value="" onChange={() => {}} />
        <FilterChip label="Status" icon="●" data={statusData} value={null} onChange={() => {}} valueText={(l, v) => `${l}: ${v}`} clearLabel="Clear status" nothingFound="No options" />
        <Select aria-label="Kind" data={kindData} value={null} onChange={() => {}} placeholder="Kind" />
      </>
    ),
    tools: <Button variant="default">Columns</Button>,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const body = within(canvasElement.ownerDocument.body)

    // A filter reads like the tool buttons beside it, down to its primary edge once open — on the
    // control itself for a FilterChip, on its bordered box for a Select (whose own trigger is a
    // plain child button with no border of its own).
    const probe = document.createElement('div')
    probe.style.borderColor = 'var(--mmerp-color-primary-6)'
    canvasElement.appendChild(probe)
    const primaryEdge = getComputedStyle(probe).borderColor

    const chip = canvas.getByRole('combobox', { name: 'Status' })
    await userEvent.click(chip)
    await waitFor(() => expect(body.getByRole('listbox')).toBeInTheDocument())
    // The border colour transitions, so the computed value only settles a beat after the attribute lands.
    await waitFor(() => expect(getComputedStyle(chip).borderColor).toBe(primaryEdge))
    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(body.queryByRole('listbox')).not.toBeInTheDocument())

    const select = canvas.getByRole('combobox', { name: 'Kind' })
    const selectBox = select.closest('[data-mmerp-control]') as HTMLElement
    await userEvent.click(select)
    await waitFor(() => expect(body.getByRole('listbox')).toBeInTheDocument())
    await waitFor(() => expect(getComputedStyle(selectBox).borderColor).toBe(primaryEdge))

    probe.remove()
  },
}

export const WithBulk: Story = {
  args: {
    ...Default.args,
    bulk: (
      <BulkBar countText="2 selected" label="Bulk actions" clearLabel="Clear selection" onClear={() => {}}>
        <Button variant="default">Export</Button>
      </BulkBar>
    ),
  },
}
