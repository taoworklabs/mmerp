import type { Meta, StoryObj } from '@storybook/react-vite'
import type * as React from 'react'
import { expect, userEvent, within } from 'storybook/test'
import { TextInput } from './TextInput.tsx'

const meta: Meta<typeof TextInput> = {
  title: 'Input/TextInput',
  component: TextInput,
}

export default meta

type Story = StoryObj<typeof TextInput>

export const Default: Story = {
  args: { label: 'Họ tên' },
  play: async ({ canvasElement }) => {
    const input = within(canvasElement).getByRole('textbox', { name: 'Họ tên' })
    await userEvent.type(input, 'Lê Thị Hồng')
    expect(input).toHaveValue('Lê Thị Hồng')
  },
}

export const Filled: Story = {
  args: { label: 'Họ tên', defaultValue: 'Nguyễn Thị Thu Hương' },
}

export const Placeholder: Story = {
  args: { label: 'Họ tên', placeholder: 'Nhập họ tên' },
}

export const WithDescription: Story = {
  args: { label: 'Họ tên', description: 'Ghi đúng như trên căn cước công dân' },
}

export const Required: Story = {
  args: { label: 'Họ tên', required: true },
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByRole('textbox', { name: 'Họ tên' })).toHaveAttribute('aria-required', 'true')
  },
}

export const Error: Story = {
  args: { label: 'Họ tên', error: 'Vui lòng nhập họ tên' },
  play: async ({ canvasElement }) => {
    const input = within(canvasElement).getByRole('textbox', { name: 'Họ tên' })
    expect(input).toHaveAttribute('aria-invalid', 'true')
    const describedBy = input.getAttribute('aria-describedby')
    expect(describedBy).toBeTruthy()
    const errorText = canvasElement.querySelector(`#${describedBy}`)
    expect(errorText).toHaveTextContent('Vui lòng nhập họ tên')
  },
}

export const Disabled: Story = {
  args: { label: 'Họ tên', disabled: true, defaultValue: 'Nguyễn Thị Thu Hương' },
}

export const ReadOnly: Story = {
  args: { label: 'Họ tên', readOnly: true, defaultValue: 'Nguyễn Thị Thu Hương' },
}

export const RootClassName: Story = {
  tags: ['!dev', '!autodocs'],
  args: { label: 'Họ tên', className: 'custom-width' },
  play: async ({ canvasElement }) => {
    const input = within(canvasElement).getByRole('textbox', { name: 'Họ tên' })
    expect(input).not.toHaveClass('custom-width')
    const root = canvasElement.querySelector('.custom-width')
    expect(root).not.toBeNull()
    expect(root?.querySelector('label')).not.toBeNull()
  },
}

export const InModal: Story = {
  tags: ['!dev', '!autodocs'],
  args: { label: 'Họ tên' },
  render: (args) => (
    <div style={{ '--mmerp-control-height': 'var(--mmerp-control-height-lg)' } as React.CSSProperties}>
      <TextInput {...args} />
    </div>
  ),
  play: async ({ canvasElement }) => {
    const input = within(canvasElement).getByRole('textbox', { name: 'Họ tên' })
    // The border sits on the wrapper around the borderless input; that's the 40px control box.
    expect(input.parentElement?.getBoundingClientRect().height).toBe(40)
  },
}
