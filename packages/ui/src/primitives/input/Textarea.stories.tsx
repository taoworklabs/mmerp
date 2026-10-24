import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Textarea } from './Textarea.tsx'

const meta: Meta<typeof Textarea> = {
  title: 'Input/Textarea',
  component: Textarea,
  args: { label: 'Ghi chú' },
}

export default meta

type Story = StoryObj<typeof Textarea>

export const Default: Story = {}

export const Filled: Story = {
  args: { defaultValue: 'Khách yêu cầu giao trước 17 giờ, liên hệ chị Hương ở Kho Bình Dương.' },
}

export const Placeholder: Story = {
  args: { placeholder: 'Nhập ghi chú' },
}

export const WithDescription: Story = {
  args: { description: 'Chỉ người duyệt nhìn thấy' },
}

export const Required: Story = {
  args: { required: true },
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByRole('textbox', { name: 'Ghi chú' })).toHaveAttribute('aria-required', 'true')
  },
}

export const Error: Story = {
  args: { error: 'Ghi chú không được để trống' },
}

export const Disabled: Story = {
  args: { disabled: true, defaultValue: 'Khách yêu cầu giao trước 17 giờ, liên hệ chị Hương ở Kho Bình Dương.' },
}

export const ReadOnly: Story = {
  args: { readOnly: true, defaultValue: 'Khách yêu cầu giao trước 17 giờ, liên hệ chị Hương ở Kho Bình Dương.' },
}

export const Autosize: Story = {
  args: { autosize: true, minRows: 2, maxRows: 4, defaultValue: 'Ô nhập tự giãn theo nội dung.' },
  play: async ({ canvasElement }) => {
    const textarea = within(canvasElement).getByRole('textbox', { name: 'Ghi chú' })
    expect(textarea).toHaveAttribute('data-autosize', '')
  },
}
