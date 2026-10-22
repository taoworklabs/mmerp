import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { Checkbox, type CheckboxProps } from './Checkbox.tsx'

const meta: Meta<typeof Checkbox> = {
  title: 'Input/Checkbox',
  component: Checkbox,
  args: { label: 'Đồng ý với điều khoản sử dụng', onChange: fn() },
}

export default meta

type Story = StoryObj<typeof Checkbox>

function Controlled(props: CheckboxProps) {
  const [checked, setChecked] = useState(props.checked ?? false)
  return (
    <Checkbox
      {...props}
      checked={checked}
      onChange={(next) => {
        setChecked(next)
        props.onChange?.(next)
      }}
    />
  )
}

export const Default: Story = {
  args: { checked: false },
  render: (args) => <Controlled {...args} />,
}

export const Filled: Story = {
  args: { checked: true },
  render: (args) => <Controlled {...args} />,
}

export const WithDescription: Story = {
  args: { checked: false, description: 'Bắt buộc để tiếp tục' },
  render: (args) => <Controlled {...args} />,
}

export const Error: Story = {
  args: { checked: false, error: 'Bạn cần đồng ý với điều khoản sử dụng' },
  render: (args) => <Controlled {...args} />,
}

export const Disabled: Story = {
  args: { checked: false, disabled: true },
  render: (args) => <Controlled {...args} />,
}

export const Indeterminate: Story = {
  args: { checked: false, indeterminate: true },
  play: async ({ canvasElement }) => {
    const checkbox = within(canvasElement).getByRole('checkbox')
    expect(checkbox).toHaveAttribute('aria-checked', 'mixed')
  },
}

export const ToggleWithSpace: Story = {
  tags: ['!dev', '!autodocs'],
  args: { checked: false },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const checkbox = within(canvasElement).getByRole('checkbox')
    checkbox.focus()
    expect(checkbox).toHaveAttribute('aria-checked', 'false')
    await userEvent.keyboard(' ')
    expect(checkbox).toHaveAttribute('aria-checked', 'true')
  },
}
