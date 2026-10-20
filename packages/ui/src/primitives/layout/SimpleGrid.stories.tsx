import type { Meta, StoryObj } from '@storybook/react-vite'
import { SimpleGrid } from './SimpleGrid.tsx'

const meta: Meta<typeof SimpleGrid> = {
  title: 'Layout/SimpleGrid',
  component: SimpleGrid,
}

export default meta

type Story = StoryObj<typeof SimpleGrid>

function Cell({ label }: { label: string }) {
  return <div>{label}</div>
}

export const Default: Story = {
  args: {
    cols: 3,
    children: (
      <>
        <Cell label="1" />
        <Cell label="2" />
        <Cell label="3" />
      </>
    ),
  },
}

export const ResponsiveCols: Story = {
  args: {
    cols: { base: 1, sm: 2, md: 3, lg: 4 },
    children: (
      <>
        <Cell label="1" />
        <Cell label="2" />
        <Cell label="3" />
        <Cell label="4" />
      </>
    ),
  },
}

export const Spacing: Story = {
  args: {
    cols: 2,
    spacing: 'xl',
    children: (
      <>
        <Cell label="1" />
        <Cell label="2" />
      </>
    ),
  },
}
