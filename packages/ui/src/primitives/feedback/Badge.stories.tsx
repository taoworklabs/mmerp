import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Badge } from './Badge.tsx'

const meta: Meta<typeof Badge> = {
  title: 'Feedback/Badge',
  component: Badge,
}

export default meta

type Story = StoryObj<typeof Badge>

export const Primary: Story = {
  args: { color: 'primary', children: 'Mới' },
}

export const Success: Story = {
  args: { color: 'success', children: 'Đã duyệt' },
}

export const Warning: Story = {
  args: { color: 'warning', children: 'Chờ duyệt' },
}

export const Danger: Story = {
  args: { color: 'danger', children: 'Từ chối' },
}

export const Info: Story = {
  args: { color: 'info', children: 'Đang xử lý' },
}

export const Discovery: Story = {
  args: { color: 'discovery', children: 'Đề xuất' },
}

export const Gray: Story = {
  args: { color: 'gray', children: 'Đã hủy' },
}

export const Filled: Story = {
  args: { variant: 'filled', children: 'Đã ghi sổ' },
}

export const Outline: Story = {
  args: { variant: 'outline', children: 'Bản nháp' },
}

const longStatus = 'Chờ trưởng phòng Kinh doanh, kế toán trưởng và giám đốc chi nhánh phê duyệt điều chỉnh đơn giá'

export const Long: Story = {
  args: { children: longStatus },
  render: (args) => (
    <div style={{ width: 120 }}>
      <Badge {...args} />
    </div>
  ),
  play: async ({ canvasElement }) => {
    const badge = within(canvasElement).getByText(longStatus)
    expect(getComputedStyle(badge).textOverflow).toBe('ellipsis')
    expect(badge.scrollWidth).toBeGreaterThan(badge.clientWidth)
  },
}
