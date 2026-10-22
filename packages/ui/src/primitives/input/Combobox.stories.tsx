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
  args: { value: null, error: 'Pick someone', placeholder: 'Search a person' },
  render: (args) => <Controlled {...args} />,
}

export const Clearable: Story = {
  args: { value: 'a', clearable: true, clearLabel: 'Clear selection' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const input = canvas.getByRole('combobox', { name: 'Assignee' })
    const clear = canvas.getByRole('button', { name: 'Clear selection' })
    // A control, so it grows to the touch target on small screens.
    expect(clear).toHaveAttribute('data-mmerp-control')
    await userEvent.click(clear)
    expect(args.onChange).toHaveBeenCalledWith(null)
    expect(input).toHaveValue('')
    expect(input).toHaveFocus()
  },
}

// Remote search: the caller swaps `data` for the results, which may not contain the current value.
function RemoteResults(props: ComboboxProps) {
  const [value, setValue] = useState(props.value)
  const [items, setItems] = useState(props.data)
  return (
    <Combobox
      {...props}
      data={items}
      value={value}
      onChange={setValue}
      onSearchChange={(query) => setItems(query === 'ng' ? data.filter((item) => item.value !== 'a') : data)}
    />
  )
}

export const RemoteResultsDropSelection: Story = {
  args: { value: 'a' },
  render: (args) => <RemoteResults {...args} />,
  play: async ({ canvasElement }) => {
    const input = within(canvasElement).getByRole('combobox', { name: 'Assignee' })
    expect(input).toHaveValue('Nguyễn Văn A')
    // Replace the selected label by typing over it, as a user starting a new search would.
    await userEvent.tripleClick(input)
    await userEvent.keyboard('ng')
    expect(input).toHaveValue('ng')
  },
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

export const Required: Story = {
  args: { value: null, required: true },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByRole('combobox', { name: 'Assignee' })).toHaveAttribute('aria-required', 'true')
  },
}
