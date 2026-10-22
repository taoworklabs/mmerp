import type { Meta, StoryObj } from '@storybook/react-vite'
import { KeyFacts } from './KeyFacts.tsx'

const meta: Meta<typeof KeyFacts> = {
  title: 'Blocks/KeyFacts',
  component: KeyFacts,
}

export default meta

type Story = StoryObj<typeof KeyFacts>

export const Default: Story = {
  args: {
    facts: [
      ['Quản lý trực tiếp', 'Trần Thị B'],
      ['Thâm niên', '2 năm'],
      ['Phép năm còn lại', '8 ngày'],
      ['Làm thêm giờ', '12 giờ'],
    ],
  },
}
