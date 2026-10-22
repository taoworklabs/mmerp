import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Skeleton } from './Skeleton.tsx'

const meta: Meta<typeof Skeleton> = {
  title: 'Feedback/Skeleton',
  component: Skeleton,
}

export default meta

type Story = StoryObj<typeof Skeleton>

export const Visible: Story = {
  args: { width: 200, height: 20 },
}

export const Hidden: Story = {
  args: { visible: false, children: 'Nội dung đã tải xong' },
  play: async ({ canvasElement }) => {
    const text = within(canvasElement).getByText('Nội dung đã tải xong')
    expect(text).toBeInTheDocument()
  },
}

export const Circle: Story = {
  args: { circle: true, width: 40, height: 40 },
}
