import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { formatDate, monthLabels, viLocale } from '../../primitives/dates/storySamples.ts'
import { DateFilterChip, type DateFilterChipProps } from './DateFilterChip.tsx'

const meta: Meta<typeof DateFilterChip> = {
  title: 'Blocks/DateFilterChip',
  component: DateFilterChip,
  args: {
    label: 'Hết hạn trước',
    icon: <span aria-hidden="true">●</span>,
    locale: viLocale,
    labels: monthLabels,
    clearLabel: 'Bỏ lọc hết hạn trước',
    valueText: (label, date) => `${label}: ${formatDate(date)}`,
    onChange: fn(),
  },
}

export default meta

type Story = StoryObj<typeof DateFilterChip>

function Controlled(props: DateFilterChipProps) {
  const [value, setValue] = useState(props.value)
  return (
    <DateFilterChip
      {...props}
      value={value}
      onChange={(next) => {
        setValue(next)
        props.onChange?.(next)
      }}
    />
  )
}

export const Unset: Story = {
  args: { value: null },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const body = within(canvasElement.ownerDocument.body)
    await userEvent.click(within(canvasElement).getByRole('button', { name: 'Hết hạn trước' }))
    await waitFor(() => expect(body.getByRole('grid')).toBeInTheDocument())
    await userEvent.click(body.getByRole('gridcell', { name: '15' }))
    expect(args.onChange).toHaveBeenCalledWith(expect.stringMatching(/-15$/))
    await waitFor(() => expect(body.queryByRole('grid')).not.toBeInTheDocument())
  },
}

export const Set: Story = {
  args: { value: '2026-11-15' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('button', { name: 'Hết hạn trước: 15/11/2026' })).toBeInTheDocument()
    await userEvent.click(canvas.getByRole('button', { name: 'Bỏ lọc hết hạn trước' }))
    expect(args.onChange).toHaveBeenCalledWith(null)
  },
}
