import type { Meta, StoryObj } from '@storybook/react-vite'
import { IconAlertTriangle, IconCircleCheck, IconInfoCircle, IconSparkles } from '@tabler/icons-react'
import { expect, within } from 'storybook/test'
import { icon } from '../../tokens/tokens.ts'
import { Alert } from './Alert.tsx'

const meta: Meta<typeof Alert> = {
  title: 'Feedback/Alert',
  component: Alert,
  // React elements, not values to edit.
  argTypes: { icon: { control: false } },
}

export default meta

type Story = StoryObj<typeof Alert>

export const Info: Story = {
  args: {
    color: 'info',
    title: 'Lưu ý',
    icon: <IconInfoCircle size={icon.text.size} stroke={icon.text.stroke} />,
    children: 'Bảng lương tháng 10 sẽ được chốt vào ngày 25.',
  },
}

export const Success: Story = {
  args: {
    color: 'success',
    title: 'Đã lưu',
    icon: <IconCircleCheck size={icon.text.size} stroke={icon.text.stroke} />,
    children: 'Thay đổi của bạn đã được lưu.',
  },
}

export const Warning: Story = {
  args: {
    color: 'warning',
    title: 'Cần kiểm tra',
    icon: <IconAlertTriangle size={icon.text.size} stroke={icon.text.stroke} />,
    children: 'Có 3 nhân viên chưa ký hợp đồng lao động.',
  },
}

export const Danger: Story = {
  args: {
    color: 'danger',
    title: 'Không thành công',
    icon: <IconAlertTriangle size={icon.text.size} stroke={icon.text.stroke} />,
    children: 'Không gửi được yêu cầu. Vui lòng thử lại.',
  },
  play: async ({ canvasElement }) => {
    const alert = within(canvasElement).getByRole('alert')
    expect(alert).toBeInTheDocument()
  },
}

export const Discovery: Story = {
  args: {
    color: 'discovery',
    title: 'Mới',
    icon: <IconSparkles size={icon.text.size} stroke={icon.text.stroke} />,
    children: 'Bạn có thể in phiếu lương thành PDF.',
  },
}
