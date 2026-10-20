import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { Combobox, type ComboboxProps } from './Combobox.tsx'
import type { SelectItem } from './Select.tsx'

const data: SelectItem[] = [
  { value: 'a', label: 'Nguyễn Văn A' },
  { value: 'b', label: 'Đặng Thị Bích' },
  { value: 'c', label: 'Trần Văn C' },
]

const meta: Meta<typeof Combobox> = {
  title: 'Input/Combobox',
  component: Combobox,
  args: { label: 'Assignee', data, nothingFound: 'No people found', onChange: fn() },
}

export default meta

type Story = StoryObj<typeof Combobox>

function Controlled(props: ComboboxProps) {
  const [value, setValue] = useState(props.value)
  return (
    <Combobox
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
  args: { value: null, placeholder: 'Search a person' },
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
  args: { value: null, error: 'Pick someone', placeholder: 'Search a person' },
  render: (args) => <Controlled {...args} />,
}

export const Clearable: Story = {
  args: { value: 'a', clearable: true, clearLabel: 'Clear selection' },
  render: (args) => <Controlled {...args} />,
}

export const TypingMatches: Story = {
  args: { value: null, placeholder: 'Search a person' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const input = within(canvasElement).getByRole('combobox', { name: 'Assignee' })
    await userEvent.type(input, 'nguyen')
    const options = within(canvasElement.ownerDocument.body).getAllByRole('option')
    expect(options).toHaveLength(1)
    expect(options[0]).toHaveTextContent('Nguyễn Văn A')
  },
}

export const NothingFound: Story = {
  args: { value: null, placeholder: 'Search a person' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const input = within(canvasElement).getByRole('combobox', { name: 'Assignee' })
    await userEvent.type(input, 'zzz')
    expect(within(canvasElement.ownerDocument.body).getByText('No people found')).toBeInTheDocument()
  },
}

export const EscapeReverts: Story = {
  args: { value: 'c', placeholder: 'Search a person' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const input = within(canvasElement).getByRole('combobox', { name: 'Assignee' })
    await userEvent.type(input, 'x')
    await userEvent.keyboard('{Escape}')
    expect(input).toHaveValue('Trần Văn C')
    expect(args.onChange).not.toHaveBeenCalled()
  },
}

export const KeyboardSelect: Story = {
  args: { value: null, placeholder: 'Search a person' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const input = within(canvasElement).getByRole('combobox', { name: 'Assignee' })
    input.focus()
    await userEvent.keyboard('{ArrowDown}')
    await userEvent.keyboard('{ArrowDown}')
    await userEvent.keyboard('{Enter}')
    expect(args.onChange).toHaveBeenCalledWith('b')
  },
}
