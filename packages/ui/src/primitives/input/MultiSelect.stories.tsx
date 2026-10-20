import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { MultiSelect, type MultiSelectProps } from './MultiSelect.tsx'
import type { SelectItem } from './Select.tsx'

const data: SelectItem[] = [
  { value: 'a', label: 'Nguyễn Văn A' },
  { value: 'b', label: 'Đặng Thị Bích' },
  { value: 'c', label: 'Trần Văn C' },
]

const removeLabel = (itemLabel: string) => `Remove ${itemLabel}`

const meta: Meta<typeof MultiSelect> = {
  title: 'Input/MultiSelect',
  component: MultiSelect,
  args: { label: 'Reviewers', data, nothingFound: 'No people found', removeLabel, onChange: fn() },
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
  args: { value: [], placeholder: 'Search people' },
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
  args: { value: [], error: 'Pick at least one reviewer', placeholder: 'Search people' },
  render: (args) => <Controlled {...args} />,
}

export const ChooseTwo: Story = {
  args: { value: [], placeholder: 'Search people' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const body = within(canvasElement.ownerDocument.body)
    const input = canvas.getByRole('combobox', { name: 'Reviewers' })

    await userEvent.click(input)
    await userEvent.click(body.getByRole('option', { name: 'Nguyễn Văn A' }))
    await userEvent.click(input)
    await userEvent.click(body.getByRole('option', { name: 'Đặng Thị Bích' }))
    // The popup is modal while open, which makes the rest of the page (the chips) inert.
    await userEvent.keyboard('{Escape}')

    expect(canvas.getByRole('button', { name: 'Remove Nguyễn Văn A' })).toBeInTheDocument()
    expect(canvas.getByRole('button', { name: 'Remove Đặng Thị Bích' })).toBeInTheDocument()
  },
}

export const RemoveChip: Story = {
  args: { value: ['a', 'b'] },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Remove Nguyễn Văn A' }))
    expect(args.onChange).toHaveBeenCalledWith(['b'])
  },
}
