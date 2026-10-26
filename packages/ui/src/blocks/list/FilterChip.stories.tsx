import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { FilterChip, type FilterChipProps } from './FilterChip.tsx'

const data = [
  { value: 'open', label: 'Đang mở' },
  { value: 'closed', label: 'Đã đóng' },
  { value: 'long', label: 'Chờ duyệt cấp phòng ban, sau đó chờ kế toán trưởng và ban giám đốc phê duyệt trước khi ghi sổ '.repeat(2).trim() },
]

const valueText = (label: string, value: string) => `${label}: ${value}`

const meta: Meta<typeof FilterChip> = {
  title: 'Blocks/FilterChip',
  component: FilterChip,
  args: {
    label: 'Trạng thái',
    icon: '●',
    data,
    nothingFound: 'Không có lựa chọn',
    clearLabel: 'Bỏ lọc trạng thái',
    valueText,
    onChange: fn(),
  },
}

export default meta

type Story = StoryObj<typeof FilterChip>

function Controlled(props: FilterChipProps) {
  const [value, setValue] = useState(props.value)
  return (
    <FilterChip
      {...props}
      value={value}
      onChange={(next) => {
        setValue(next)
        props.onChange(next)
      }}
    />
  )
}

export const Unset: Story = {
  args: { value: null },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('combobox', { name: 'Trạng thái' })
    const body = within(canvasElement.ownerDocument.body)

    // A lone trigger keeps all four corners round, despite the combobox's hidden input beside it.
    const radius = getComputedStyle(trigger)
    expect(radius.borderStartStartRadius).not.toBe('0px')
    expect(radius.borderStartEndRadius).toBe(radius.borderStartStartRadius)
    expect(radius.borderEndEndRadius).toBe(radius.borderStartStartRadius)

    trigger.focus()
    await userEvent.keyboard('{Enter}')
    await waitFor(() => expect(body.getByRole('listbox')).toBeInTheDocument())

    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(body.queryByRole('listbox')).not.toBeInTheDocument())
    await waitFor(() => expect(trigger).toHaveFocus())
  },
}

export const Set: Story = {
  args: { value: 'open' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const clear = canvas.getByRole('button', { name: 'Bỏ lọc trạng thái' })
    const trigger = canvas.getByRole('combobox')
    // Joined edges are square, outer edges round.
    expect(getComputedStyle(trigger).borderStartEndRadius).toBe('0px')
    expect(getComputedStyle(trigger).borderStartStartRadius).not.toBe('0px')
    expect(getComputedStyle(clear).borderStartStartRadius).toBe('0px')
    expect(getComputedStyle(clear).borderStartEndRadius).not.toBe('0px')
    await userEvent.click(clear)
    expect(args.onChange).toHaveBeenCalledWith(null)
  },
}

export const LongValue: Story = {
  args: { value: 'long' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const chip = canvas.getByRole('group')
    expect(chip.getBoundingClientRect().width).toBeLessThanOrEqual(320)

    const trigger = canvas.getByRole('combobox')
    const text = trigger.querySelector('span') as HTMLElement
    expect(text.scrollWidth).toBeGreaterThan(text.clientWidth)
  },
}

export const Searchable: Story = {
  args: { value: null, searchable: true, searchLabel: 'Tìm trạng thái' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('combobox', { name: 'Trạng thái' })
    const body = within(canvasElement.ownerDocument.body)

    await userEvent.click(trigger)
    await waitFor(() => expect(body.getByRole('listbox')).toBeInTheDocument())
    const search = body.getByRole('combobox', { name: 'Tìm trạng thái' })
    await waitFor(() => expect(search).toHaveFocus())
  },
}
