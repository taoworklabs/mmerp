import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { viLocale, yearLabels } from '../../primitives/dates/storySamples.ts'
import { MonthFilterChip, type MonthFilterChipProps } from './MonthFilterChip.tsx'

const meta: Meta<typeof MonthFilterChip> = {
  title: 'Blocks/MonthFilterChip',
  component: MonthFilterChip,
  args: {
    label: 'Tháng',
    icon: <span aria-hidden="true">●</span>,
    locale: viLocale,
    labels: yearLabels,
    clearLabel: 'Bỏ lọc tháng',
    valueText: (label, month) => `${label}: ${month.slice(5)}/${month.slice(0, 4)}`,
    onChange: fn(),
  },
}

export default meta

type Story = StoryObj<typeof MonthFilterChip>

function Controlled(props: MonthFilterChipProps) {
  const [value, setValue] = useState(props.value)
  return (
    <MonthFilterChip
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
  play: async ({ canvasElement, args }) => {
    const body = within(canvasElement.ownerDocument.body)
    await userEvent.click(within(canvasElement).getByRole('button', { name: 'Tháng' }))
    await waitFor(() => expect(body.getByRole('grid')).toBeInTheDocument())
    await userEvent.click(body.getByRole('gridcell', { name: 'Tháng 3' }))
    expect(args.onChange).toHaveBeenCalledWith(expect.stringMatching(/-03$/))
    await waitFor(() => expect(body.queryByRole('grid')).not.toBeInTheDocument())
  },
}

export const Set: Story = {
  args: { value: '2026-03' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('button', { name: 'Tháng: 03/2026' })).toBeInTheDocument()
    await userEvent.click(canvas.getByRole('button', { name: 'Bỏ lọc tháng' }))
    expect(args.onChange).toHaveBeenCalledWith(null)
  },
}

export const LongLabel: Story = {
  args: { value: '2026-03', label: 'Tháng bắt đầu kỳ lương của nhân viên '.repeat(4) },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const chip = within(canvasElement).getByRole('group')
    expect(chip.getBoundingClientRect().width).toBeLessThanOrEqual(320)
  },
}
