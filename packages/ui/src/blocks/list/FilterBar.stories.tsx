import type { Meta, StoryObj } from '@storybook/react-vite'
import { Button } from '../../primitives/input/Button.tsx'
import { BulkBar } from './BulkBar.tsx'
import { FilterBar } from './FilterBar.tsx'
import { FilterChip } from './FilterChip.tsx'

const statusData = [
  { value: 'open', label: 'Open' },
  { value: 'closed', label: 'Closed' },
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
      </>
    ),
    tools: <Button variant="default">Columns</Button>,
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
