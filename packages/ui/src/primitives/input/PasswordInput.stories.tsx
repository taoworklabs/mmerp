import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'
import { PasswordInput } from './PasswordInput.tsx'

const meta: Meta<typeof PasswordInput> = {
  title: 'Input/PasswordInput',
  component: PasswordInput,
  args: { label: 'Mật khẩu', visibilityLabels: { show: 'Hiện mật khẩu', hide: 'Ẩn mật khẩu' } },
}

export default meta

type Story = StoryObj<typeof PasswordInput>

export const Default: Story = {
  play: async ({ canvasElement }) => {
    const input = canvasElement.querySelector('input')
    expect(input).toHaveAttribute('type', 'password')
  },
}

export const Filled: Story = {
  args: { defaultValue: 'secret-value' },
}

export const Placeholder: Story = {
  args: { placeholder: 'Nhập mật khẩu' },
}

export const WithDescription: Story = {
  args: { description: 'Tối thiểu 8 ký tự' },
}

export const Required: Story = {
  args: { required: true },
  play: async ({ canvasElement }) => {
    expect(canvasElement.querySelector('input')).toHaveAttribute('aria-required', 'true')
  },
}

export const Error: Story = {
  args: { error: 'Mật khẩu quá ngắn' },
}

export const Disabled: Story = {
  args: { disabled: true, defaultValue: 'secret-value' },
}

export const ReadOnly: Story = {
  args: { readOnly: true, defaultValue: 'secret-value' },
}

export const ToggleVisibility: Story = {
  tags: ['!dev', '!autodocs'],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const toggle = canvas.getByRole('button', { name: 'Hiện mật khẩu' })
    await userEvent.click(toggle)
    const input = canvasElement.querySelector('input')
    expect(input).toHaveAttribute('type', 'text')
    expect(canvas.getByRole('button', { name: 'Ẩn mật khẩu' })).toBeInTheDocument()
  },
}
