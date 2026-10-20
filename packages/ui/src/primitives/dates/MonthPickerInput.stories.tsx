import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { MonthPickerInput, type MonthPickerInputProps } from './MonthPickerInput.tsx'
import { formatMonth, viLocale, yearLabels } from './storySamples.ts'

const meta: Meta<typeof MonthPickerInput> = {
  title: 'Dates/MonthPickerInput',
  component: MonthPickerInput,
  args: { label: 'Kỳ lương', placeholder: 'mm/yyyy', format: formatMonth, locale: viLocale, labels: yearLabels, onChange: fn() },
}

export default meta

type Story = StoryObj<typeof MonthPickerInput>

function Controlled(props: MonthPickerInputProps) {
  const [value, setValue] = useState(props.value)
  return (
    <MonthPickerInput
      {...props}
      value={value}
      onChange={(month) => {
        setValue(month)
        props.onChange(month)
      }}
    />
  )
}

export const Empty: Story = {
  args: { value: null },
  render: (args) => <Controlled {...args} />,
}

export const Filled: Story = {
  args: { value: '2026-10' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    // The name carries both the field label and the chosen month.
    const trigger = within(canvasElement).getByRole('button', { name: 'Kỳ lương 10/2026' })
    await userEvent.click(trigger)
    const body = within(canvasElement.ownerDocument.body)
    const october = await body.findByRole('gridcell', { name: 'Tháng 10' })
    await waitFor(() => expect(october).toHaveFocus())

    await userEvent.keyboard('{ArrowRight}{Enter}')
    expect(args.onChange).toHaveBeenCalledWith('2026-11')
    await waitFor(() => expect(body.queryByRole('grid')).not.toBeInTheDocument())
    expect(trigger).toHaveTextContent('11/2026')
    await waitFor(() => expect(trigger).toHaveFocus())
  },
}

export const Disabled: Story = {
  args: { value: '2026-10', disabled: true, clearable: true, clearLabel: 'Xóa' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('button', { name: /Kỳ lương/ })).toBeDisabled()
    expect(canvas.queryByRole('button', { name: 'Xóa' })).not.toBeInTheDocument()
  },
}

export const ReadOnly: Story = {
  args: { value: '2026-10', readOnly: true, clearable: true, clearLabel: 'Xóa' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).queryByRole('button', { name: 'Xóa' })).not.toBeInTheDocument()
  },
}

export const Error: Story = {
  args: { value: null, error: 'Chọn kỳ lương' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByRole('button', { name: /Kỳ lương/ })).toHaveAttribute('aria-invalid', 'true')
  },
}

export const Clearable: Story = {
  args: { value: '2026-10', clearable: true, clearLabel: 'Xóa' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Xóa' }))
    expect(args.onChange).toHaveBeenCalledWith(null)
    expect(canvas.getByRole('button', { name: /Kỳ lương/ })).toHaveTextContent('mm/yyyy')
  },
}
