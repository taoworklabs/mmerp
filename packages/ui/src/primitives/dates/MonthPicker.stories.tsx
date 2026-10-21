import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { MonthPicker, type MonthPickerProps } from './MonthPicker.tsx'
import { viLocale, yearLabels } from './storySamples.ts'

const meta: Meta<typeof MonthPicker> = {
  title: 'Dates/MonthPicker',
  component: MonthPicker,
  args: { locale: viLocale, labels: yearLabels, onChange: fn() },
}

export default meta

type Story = StoryObj<typeof MonthPicker>

function Controlled(props: MonthPickerProps) {
  const [value, setValue] = useState(props.value)
  return (
    <MonthPicker
      {...props}
      value={value}
      onChange={(month) => {
        setValue(month)
        props.onChange(month)
      }}
    />
  )
}

export const Default: Story = {
  args: { value: '2026-10' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await userEvent.tab()
    await userEvent.tab()
    await userEvent.tab()
    expect(canvas.getByRole('gridcell', { name: 'Tháng 10' })).toHaveFocus()

    await userEvent.keyboard('{ArrowRight}{Enter}')
    expect(args.onChange).toHaveBeenCalledWith('2026-11')

    await userEvent.keyboard('{ArrowUp}')
    expect(canvas.getByRole('gridcell', { name: 'Tháng 8' })).toHaveFocus()
    await userEvent.keyboard('{PageDown}')
    expect(canvas.getByText('2027')).toBeInTheDocument()
  },
}

export const MinMaxDate: Story = {
  args: { value: '2026-10', minDate: '2026-03', maxDate: '2026-11-15' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('gridcell', { name: 'Tháng 2' })).toBeDisabled()
    expect(canvas.getByRole('gridcell', { name: 'Tháng 11' })).toBeEnabled()
    expect(canvas.getByRole('gridcell', { name: 'Tháng 12' })).toBeDisabled()
  },
}
