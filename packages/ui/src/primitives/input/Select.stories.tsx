import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { Select, type SelectItem, type SelectProps } from './Select.tsx'

const data: SelectItem[] = [
  { value: 'a', label: 'Nguyễn Văn A' },
  { value: 'b', label: 'Đặng Thị Bích' },
  { value: 'c', label: 'Trần Văn C' },
]

const longLabel = 'A'.repeat(200)
const longData: SelectItem[] = [{ value: 'long', label: longLabel }]

const meta: Meta<typeof Select> = {
  title: 'Input/Select',
  component: Select,
  args: { label: 'Assignee', data, onChange: fn() },
}

export default meta

type Story = StoryObj<typeof Select>

function Controlled(props: SelectProps) {
  const [value, setValue] = useState(props.value)
  return (
    <Select
      {...props}
      value={value}
      onChange={(next) => {
        setValue(next)
        props.onChange(next)
      }}
    />
  )
}

export const Default: Story = {
  args: { value: null, placeholder: 'Choose a person' },
  render: (args) => <Controlled {...args} />,
}

export const WithValue: Story = {
  args: { value: 'a' },
  render: (args) => <Controlled {...args} />,
}

export const Disabled: Story = {
  args: { value: 'a', disabled: true },
  render: (args) => <Controlled {...args} />,
}

export const ReadOnly: Story = {
  args: { value: 'a', readOnly: true },
  render: (args) => <Controlled {...args} />,
}

export const Error: Story = {
  args: { value: null, error: 'Pick someone', placeholder: 'Choose a person' },
  render: (args) => <Controlled {...args} />,
}

export const Clearable: Story = {
  args: { value: 'a', clearable: true, clearLabel: 'Clear selection' },
  render: (args) => <Controlled {...args} />,
}

export const KeyboardSelect: Story = {
  args: { value: null, placeholder: 'Choose a person' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const trigger = within(canvasElement).getByRole('combobox', { name: 'Assignee' })
    trigger.focus()
    await userEvent.keyboard('{ArrowDown}')
    await userEvent.keyboard('{ArrowDown}')
    await userEvent.keyboard('{Enter}')
    expect(args.onChange).toHaveBeenCalledWith('b')
  },
}

export const Long: Story = {
  args: { data: longData, value: 'long', style: { width: 240 } },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const trigger = within(canvasElement).getByRole('combobox', { name: 'Assignee' })
    expect(trigger.getBoundingClientRect().width).toBe(240)
    const value = within(trigger).getByText(longLabel)
    expect(getComputedStyle(value).textOverflow).toBe('ellipsis')
  },
}
