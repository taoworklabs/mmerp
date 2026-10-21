import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { FilterChip, type FilterChipProps } from './FilterChip.tsx'

const data = [
  { value: 'open', label: 'Open' },
  { value: 'closed', label: 'Closed' },
  { value: 'long', label: 'A'.repeat(200) },
]

const valueText = (label: string, value: string) => `${label}: ${value}`

const meta: Meta<typeof FilterChip> = {
  title: 'Blocks/FilterChip',
  component: FilterChip,
  args: {
    label: 'Status',
    icon: '●',
    data,
    nothingFound: 'No options',
    clearLabel: 'Clear status filter',
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
    const trigger = canvas.getByRole('combobox', { name: 'Status' })
    const body = within(canvasElement.ownerDocument.body)

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
    const clear = canvas.getByRole('button', { name: 'Clear status filter' })
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
  args: { value: null, searchable: true, searchLabel: 'Search statuses' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('combobox', { name: 'Status' })
    const body = within(canvasElement.ownerDocument.body)

    await userEvent.click(trigger)
    await waitFor(() => expect(body.getByRole('listbox')).toBeInTheDocument())
    const search = body.getByRole('combobox', { name: 'Search statuses' })
    await waitFor(() => expect(search).toHaveFocus())
  },
}
