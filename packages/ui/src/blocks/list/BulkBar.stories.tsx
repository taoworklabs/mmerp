import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import { Button } from '../../primitives/input/Button.tsx'
import { BulkBar } from './BulkBar.tsx'

const meta: Meta<typeof BulkBar> = {
  title: 'Blocks/BulkBar',
  component: BulkBar,
  args: {
    countText: 'Đã chọn 3',
    label: 'Thao tác hàng loạt',
    clearLabel: 'Bỏ chọn',
    onClear: fn(),
  },
  // React elements, not values to edit.
  argTypes: { children: { control: false } },
}

export default meta

type Story = StoryObj<typeof BulkBar>

export const Default: Story = {
  args: {
    children: (
      <>
        <Button variant="default">Xuất file</Button>
        <Button color="danger" variant="outline">
          Xóa
        </Button>
      </>
    ),
  },
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('toolbar', { name: 'Thao tác hàng loạt' })).toBeInTheDocument()

    const clear = canvas.getByRole('button', { name: 'Bỏ chọn' })
    await userEvent.click(clear)
    expect(args.onClear).toHaveBeenCalledTimes(1)
  },
}
