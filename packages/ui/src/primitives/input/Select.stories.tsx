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
  args: { value: 'a', disabled: true, clearable: true, clearLabel: 'Clear selection' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).queryByRole('button', { name: 'Clear selection' })).not.toBeInTheDocument()
  },
}

export const ReadOnly: Story = {
  args: { value: 'a', readOnly: true, clearable: true, clearLabel: 'Clear selection' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).queryByRole('button', { name: 'Clear selection' })).not.toBeInTheDocument()
  },
}

export const Error: Story = {
  args: { value: null, error: 'Pick someone', placeholder: 'Choose a person' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const trigger = within(canvasElement).getByRole('combobox', { name: 'Assignee' })
    expect(trigger).toHaveAttribute('aria-invalid', 'true')
    const describedBy = trigger.getAttribute('aria-describedby')
    expect(describedBy).toBeTruthy()
    const errorText = canvasElement.querySelector(`#${describedBy}`)
    expect(errorText).toHaveTextContent('Pick someone')
  },
}

export const Clearable: Story = {
  args: { value: 'a', clearable: true, clearLabel: 'Clear selection' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('combobox', { name: 'Assignee' })
    const clearButton = canvas.getByRole('button', { name: 'Clear selection' })
    // The clear button must be a sibling of the trigger, never nested inside its <button>.
    expect(trigger.contains(clearButton)).toBe(false)

    await userEvent.click(clearButton)
    expect(args.onChange).toHaveBeenCalledWith(null)
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    // The button vanishes with the value; focus must not fall back to the page.
    expect(trigger).toHaveFocus()
  },
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
  args: { data: longData, value: 'long' },
  // Width is constrained on an ancestor, the way a consumer's own layout would — the wrapper
  // box stretches to fill it (Field's column flex defaults to `align-items: stretch`).
  render: (args) => (
    <div style={{ width: 240 }}>
      <Controlled {...args} />
    </div>
  ),
  play: async ({ canvasElement }) => {
    const box = canvasElement.querySelector('[data-mmerp-control]') as HTMLElement
    expect(box.getBoundingClientRect().width).toBe(240)
    const value = within(canvasElement).getByText(longLabel)
    expect(getComputedStyle(value).textOverflow).toBe('ellipsis')
  },
}

export const Required: Story = {
  args: { value: null, required: true },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByRole('combobox', { name: 'Assignee' })).toHaveAttribute('aria-required', 'true')
  },
}
