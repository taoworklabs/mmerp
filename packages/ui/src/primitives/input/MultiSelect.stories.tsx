import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { MultiSelect, type MultiSelectProps } from './MultiSelect.tsx'
import type { SelectItem } from './Select.tsx'

const data: SelectItem[] = [
  { value: 'a', label: 'Nguyễn Văn An' },
  { value: 'b', label: 'Đặng Thị Bích' },
  { value: 'c', label: 'Trần Văn Cường' },
]

const removeLabel = (itemLabel: string) => `Bỏ chọn ${itemLabel}`

const meta: Meta<typeof MultiSelect> = {
  title: 'Input/MultiSelect',
  component: MultiSelect,
  args: { label: 'Người duyệt', data, nothingFound: 'Không tìm thấy nhân viên', removeLabel, onChange: fn() },
}

export default meta

type Story = StoryObj<typeof MultiSelect>

function Controlled(props: MultiSelectProps) {
  const [value, setValue] = useState(props.value)
  return (
    <MultiSelect
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
  args: { value: [], placeholder: 'Tìm nhân viên' },
  render: (args) => <Controlled {...args} />,
}

export const WithValue: Story = {
  args: { value: ['a', 'c'] },
  render: (args) => <Controlled {...args} />,
}

export const Disabled: Story = {
  args: { value: ['a'], disabled: true },
  render: (args) => <Controlled {...args} />,
}

export const ReadOnly: Story = {
  args: { value: ['a'], readOnly: true },
  render: (args) => <Controlled {...args} />,
}

export const Error: Story = {
  args: { value: [], error: 'Chọn ít nhất một người duyệt', placeholder: 'Tìm nhân viên' },
  render: (args) => <Controlled {...args} />,
}

export const ChooseTwo: Story = {
  tags: ['!dev', '!autodocs'],
  args: { value: [], placeholder: 'Tìm nhân viên' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const body = within(canvasElement.ownerDocument.body)
    const input = canvas.getByRole('combobox', { name: 'Người duyệt' })

    await userEvent.click(input)
    await userEvent.click(body.getByRole('option', { name: 'Nguyễn Văn An' }))
    await userEvent.click(input)
    await userEvent.click(body.getByRole('option', { name: 'Đặng Thị Bích' }))
    // The popup is modal while open, which makes the rest of the page (the chips) inert.
    await userEvent.keyboard('{Escape}')

    expect(canvas.getByRole('button', { name: 'Bỏ chọn Nguyễn Văn An' })).toBeInTheDocument()
    expect(canvas.getByRole('button', { name: 'Bỏ chọn Đặng Thị Bích' })).toBeInTheDocument()
  },
}

export const RemoveChip: Story = {
  tags: ['!dev', '!autodocs'],
  args: { value: ['a', 'b'] },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Bỏ chọn Nguyễn Văn An' }))
    expect(args.onChange).toHaveBeenCalledWith(['b'])
  },
}

export const Required: Story = {
  args: { value: [], required: true },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByRole('combobox', { name: 'Người duyệt' })).toHaveAttribute('aria-required', 'true')
  },
}

export const EscapeKeepsValues: Story = {
  tags: ['!dev', '!autodocs'],
  args: { value: ['a'] },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const input = within(canvasElement).getByRole('combobox', { name: 'Người duyệt' })
    input.focus()
    await userEvent.keyboard('{Escape}')
    expect(args.onChange).not.toHaveBeenCalled()
    expect(within(canvasElement).getByText(data[0]!.label)).toBeInTheDocument()
  },
}
