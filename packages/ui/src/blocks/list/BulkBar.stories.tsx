import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import { Button } from '../../primitives/input/Button.tsx'
import { BulkBar } from './BulkBar.tsx'

const meta: Meta<typeof BulkBar> = {
  title: 'Blocks/BulkBar',
  component: BulkBar,
  args: {
    countText: '3 selected',
    label: 'Bulk actions',
    clearLabel: 'Clear selection',
    onClear: fn(),
  },
}

export default meta

type Story = StoryObj<typeof BulkBar>

export const Default: Story = {
  args: {
    children: (
      <>
        <Button variant="default">Export</Button>
        <Button color="danger" variant="outline">
          Delete
        </Button>
      </>
    ),
  },
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('toolbar', { name: 'Bulk actions' })).toBeInTheDocument()

    const clear = canvas.getByRole('button', { name: 'Clear selection' })
    await userEvent.click(clear)
    expect(args.onClear).toHaveBeenCalledTimes(1)
  },
}
