import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { NumberInput, type NumberInputProps } from './NumberInput.tsx'

const meta: Meta<typeof NumberInput> = {
  title: 'Input/NumberInput',
  component: NumberInput,
  args: { label: 'Quantity', onChange: fn() },
}

export default meta

type Story = StoryObj<typeof NumberInput>

function Controlled(props: NumberInputProps) {
  const [value, setValue] = useState(props.value)
  return (
    <NumberInput
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
  args: { value: null },
  render: (args) => <Controlled {...args} />,
}

export const Filled: Story = {
  args: { value: 42 },
  render: (args) => <Controlled {...args} />,
}

export const Placeholder: Story = {
  args: { value: null, placeholder: 'Enter a quantity' },
  render: (args) => <Controlled {...args} />,
}

export const WithDescription: Story = {
  args: { value: null, description: 'Whole units only' },
  render: (args) => <Controlled {...args} />,
}

export const Required: Story = {
  args: { value: null, required: true },
  render: (args) => <Controlled {...args} />,
}

export const Error: Story = {
  args: { value: null, error: 'Quantity must be positive' },
  render: (args) => <Controlled {...args} />,
}

export const Disabled: Story = {
  args: { value: 5, disabled: true },
  render: (args) => <Controlled {...args} />,
}

export const ReadOnly: Story = {
  args: { value: 5, readOnly: true },
  render: (args) => <Controlled {...args} />,
}

export const Stepping: Story = {
  args: { value: 5 },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const input = within(canvasElement).getByRole('textbox', { name: 'Quantity' })
    input.focus()
    await userEvent.keyboard('{ArrowUp}')
    expect(input).toHaveValue('6')
    expect(args.onChange).toHaveBeenCalledWith(6)

    await userEvent.clear(input)
    await userEvent.keyboard('{Tab}')
    expect(args.onChange).toHaveBeenCalledWith(null)
  },
}
