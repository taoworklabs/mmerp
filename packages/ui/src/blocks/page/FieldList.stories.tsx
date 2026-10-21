import type { Meta, StoryObj } from '@storybook/react-vite'
import { FieldList } from './FieldList.tsx'

const meta: Meta<typeof FieldList> = {
  title: 'Blocks/FieldList',
  component: FieldList,
}

export default meta

type Story = StoryObj<typeof FieldList>

export const SideBySide: Story = {
  args: {
    rows: [
      ['Name', 'Nguyễn Văn A'],
      ['Email', 'a@example.com'],
      ['Phone', '090 123 4567'],
    ],
  },
}

export const Stacked: Story = {
  args: {
    stacked: true,
    rows: [
      ['Name', 'Nguyễn Văn A'],
      ['Phone', '090 123 4567'],
      ['Address', '123 Some Street, District 1, Ho Chi Minh City', 'wide'],
    ],
  },
}
