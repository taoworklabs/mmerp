import type { Meta, StoryObj } from '@storybook/react-vite'
import { VisuallyHidden } from './VisuallyHidden.tsx'

const meta: Meta<typeof VisuallyHidden> = {
  title: 'Layout/VisuallyHidden',
  component: VisuallyHidden,
}

export default meta

type Story = StoryObj<typeof VisuallyHidden>

export const Default: Story = {
  args: { children: 'Chỉ trình đọc màn hình mới đọc nội dung này' },
}
