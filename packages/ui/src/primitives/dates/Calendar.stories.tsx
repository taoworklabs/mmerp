import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { Calendar, type CalendarProps } from './Calendar.tsx'
import { monthLabels, viLocale } from './storySamples.ts'

const meta: Meta<typeof Calendar> = {
  title: 'Dates/Calendar',
  component: Calendar,
  args: { locale: viLocale, labels: monthLabels, today: '2026-10-20', onChange: fn() },
}

export default meta

type Story = StoryObj<typeof Calendar>

function Controlled(props: CalendarProps) {
  const [value, setValue] = useState(props.value)
  return (
    <Calendar
      {...props}
      value={value}
      onChange={(iso) => {
        setValue(iso)
        props.onChange(iso)
      }}
    />
  )
}

export const Default: Story = {
  args: { value: '2026-10-20' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const day20 = canvas.getByRole('gridcell', { name: '20' })
    expect(day20).toHaveAttribute('aria-selected', 'true')
    expect(day20).toHaveAttribute('aria-current', 'date')
    // Previous and next month, then the grid's single tab stop.
    await userEvent.tab()
    await userEvent.tab()
    await userEvent.tab()
    expect(day20).toHaveFocus()

    await userEvent.keyboard('{ArrowRight}')
    expect(canvas.getByRole('gridcell', { name: '21' })).toHaveFocus()
    await userEvent.keyboard('{Enter}')
    expect(args.onChange).toHaveBeenCalledWith('2026-10-21')

    await userEvent.keyboard('{PageDown}')
    expect(canvas.getByText('Tháng 11 2026')).toBeInTheDocument()
    expect(canvas.getByRole('gridcell', { name: '21' })).toHaveFocus()

    await userEvent.keyboard('{Home}')
    expect(canvas.getByRole('gridcell', { name: '16' })).toHaveFocus()
    await userEvent.keyboard('{End}')
    expect(canvas.getByRole('gridcell', { name: '22' })).toHaveFocus()
  },
}

export const Empty: Story = {
  args: { value: null },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    // With no value, today is the tab stop.
    expect(within(canvasElement).getByRole('gridcell', { name: '20' })).toHaveAttribute('tabindex', '0')
  },
}

export const DefaultMonth: Story = {
  args: { value: null, defaultMonth: '2027-03' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByText('Tháng 3 2027')).toBeInTheDocument()
  },
}

export const MinMaxDate: Story = {
  args: { value: '2026-10-20', minDate: '2026-10-10', maxDate: '2026-10-25' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('gridcell', { name: '9' })).toBeDisabled()
    expect(canvas.getByRole('gridcell', { name: '10' })).toBeEnabled()
    expect(canvas.getByRole('gridcell', { name: '26' })).toBeDisabled()
    expect(canvas.getByRole('button', { name: 'Tháng trước' })).toBeDisabled()
    expect(canvas.getByRole('button', { name: 'Tháng sau' })).toBeDisabled()
  },
}

export const FirstDayOfWeek: Story = {
  args: { value: null, locale: { ...viLocale, firstDayOfWeek: 0 } },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getAllByRole('columnheader')[0]).toHaveTextContent('CN')
  },
}
