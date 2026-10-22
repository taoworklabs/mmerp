import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Loader } from './Loader.tsx'

const meta: Meta<typeof Loader> = {
  title: 'Feedback/Loader',
  component: Loader,
}

export default meta

type Story = StoryObj<typeof Loader>

export const Xs: Story = {
  args: { size: 'xs', label: 'Đang tải' },
}

export const Sm: Story = {
  args: { size: 'sm', label: 'Đang tải' },
  play: async ({ canvasElement }) => {
    const loader = within(canvasElement).getByRole('status', { name: 'Đang tải' })
    expect(loader).toBeInTheDocument()
  },
}

export const Md: Story = {
  args: { size: 'md', label: 'Đang tải' },
}
