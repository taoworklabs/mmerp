import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { Combobox, type ComboboxProps } from './Combobox.tsx'
import type { SelectItem } from './Select.tsx'

const data: SelectItem[] = [
  { value: 'a', label: 'Nguyễn Văn An' },
  { value: 'b', label: 'Đặng Thị Bích' },
  { value: 'c', label: 'Trần Văn Cường' },
]

const meta: Meta<typeof Combobox> = {
  title: 'Input/Combobox',
  component: Combobox,
  args: { label: 'Người phụ trách', data, nothingFound: 'Không tìm thấy nhân viên', onChange: fn() },
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
  args: { value: null, placeholder: 'Tìm nhân viên' },
  render: (args) => <Controlled {...args} />,
}

export const WithValue: Story = {
  args: { value: 'a' },
  render: (args) => <Controlled {...args} />,
}

export const Disabled: Story = {
  args: { value: 'a', disabled: true, clearable: true, clearLabel: 'Bỏ chọn' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).queryByRole('button', { name: 'Bỏ chọn' })).not.toBeInTheDocument()
  },
}

export const ReadOnly: Story = {
  args: { value: 'a', readOnly: true, clearable: true, clearLabel: 'Bỏ chọn' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).queryByRole('button', { name: 'Bỏ chọn' })).not.toBeInTheDocument()
  },
}

export const Error: Story = {
  args: { value: null, error: 'Vui lòng chọn người phụ trách', placeholder: 'Tìm nhân viên' },
  render: (args) => <Controlled {...args} />,
}

export const Clearable: Story = {
  args: { value: 'a', clearable: true, clearLabel: 'Bỏ chọn' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const input = canvas.getByRole('combobox', { name: 'Người phụ trách' })
    const clear = canvas.getByRole('button', { name: 'Bỏ chọn' })
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
  tags: ['!dev', '!autodocs'],
  args: { value: 'a' },
  render: (args) => <RemoteResults {...args} />,
  play: async ({ canvasElement }) => {
    const input = within(canvasElement).getByRole('combobox', { name: 'Người phụ trách' })
    expect(input).toHaveValue('Nguyễn Văn An')
    // Replace the selected label by typing over it, as a user starting a new search would.
    await userEvent.tripleClick(input)
    await userEvent.keyboard('ng')
    expect(input).toHaveValue('ng')
  },
}

export const TypingMatches: Story = {
  tags: ['!dev', '!autodocs'],
  args: { value: null, placeholder: 'Tìm nhân viên' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const input = within(canvasElement).getByRole('combobox', { name: 'Người phụ trách' })
    await userEvent.type(input, 'nguyen')
    const options = within(canvasElement.ownerDocument.body).getAllByRole('option')
    expect(options).toHaveLength(1)
    expect(options[0]).toHaveTextContent('Nguyễn Văn An')
  },
}

export const NothingFound: Story = {
  args: { value: null, placeholder: 'Tìm nhân viên' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const input = within(canvasElement).getByRole('combobox', { name: 'Người phụ trách' })
    await userEvent.type(input, 'zzz')
    expect(within(canvasElement.ownerDocument.body).getByText('Không tìm thấy nhân viên')).toBeInTheDocument()
  },
}

export const EscapeReverts: Story = {
  tags: ['!dev', '!autodocs'],
  args: { value: 'c', placeholder: 'Tìm nhân viên' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const input = within(canvasElement).getByRole('combobox', { name: 'Người phụ trách' })
    await userEvent.type(input, 'x')
    await userEvent.keyboard('{Escape}')
    expect(input).toHaveValue('Trần Văn Cường')
    expect(args.onChange).not.toHaveBeenCalled()
  },
}

export const KeyboardSelect: Story = {
  tags: ['!dev', '!autodocs'],
  args: { value: null, placeholder: 'Tìm nhân viên' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const input = within(canvasElement).getByRole('combobox', { name: 'Người phụ trách' })
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
    expect(within(canvasElement).getByRole('combobox', { name: 'Người phụ trách' })).toHaveAttribute('aria-required', 'true')
  },
}

export const OptionAndSection: Story = {
  tags: ['!dev', '!autodocs'],
  args: {
    value: 'a',
    leftSection: <span data-testid="section">●</span>,
    renderOption: (item: SelectItem) => <strong>{item.label.toUpperCase()}</strong>,
  },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByTestId('section')).toBeInTheDocument()
    const input = canvas.getByRole('combobox', { name: 'Người phụ trách' })
    // The chosen value stays the plain label; only the list uses renderOption.
    expect(input).toHaveValue('Nguyễn Văn An')
    await userEvent.click(input)
    expect(await within(canvasElement.ownerDocument.body).findByRole('option', { name: 'ĐẶNG THỊ BÍCH' })).toBeInTheDocument()
  },
}

export const EscapeKeepsValue: Story = {
  tags: ['!dev', '!autodocs'],
  args: { value: 'a' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const input = within(canvasElement).getByRole('combobox', { name: 'Người phụ trách' })
    input.focus()
    await userEvent.keyboard('{Escape}')
    expect(args.onChange).not.toHaveBeenCalled()
    expect(input).toHaveValue('Nguyễn Văn An')
  },
}
