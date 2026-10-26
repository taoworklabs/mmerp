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
      ['Họ và tên', 'Nguyễn Văn A'],
      ['Email', 'a@example.com'],
      ['Điện thoại', '090 123 4567'],
    ],
  },
}

export const Stacked: Story = {
  args: {
    stacked: true,
    rows: [
      ['Họ và tên', 'Nguyễn Văn A'],
      ['Điện thoại', '090 123 4567'],
      ['Địa chỉ', '123 Nguyễn Thị Minh Khai, Phường Bến Thành, Quận 1, Thành phố Hồ Chí Minh', 'wide'],
    ],
  },
}
