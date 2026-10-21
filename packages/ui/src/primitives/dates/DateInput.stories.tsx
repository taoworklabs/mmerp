import type { Meta, StoryObj } from '@storybook/react-vite'
import { IconCalendar } from '@tabler/icons-react'
import { useState } from 'react'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { DateInput, type DateInputProps } from './DateInput.tsx'
import { formatDate, monthLabels, parseDate, viLocale } from './storySamples.ts'

const meta: Meta<typeof DateInput> = {
  title: 'Dates/DateInput',
  component: DateInput,
  args: {
    label: 'Ngày hiệu lực',
    placeholder: 'dd/mm/yyyy',
    parse: parseDate,
    format: formatDate,
    locale: viLocale,
    labels: monthLabels,
    onChange: fn(),
  },
}

export default meta

type Story = StoryObj<typeof DateInput>

function Controlled(props: DateInputProps) {
  const [value, setValue] = useState(props.value)
  return (
    <DateInput
      {...props}
      value={value}
      onChange={(iso) => {
        setValue(iso)
        props.onChange(iso)
      }}
    />
  )
}

export const Empty: Story = {
  args: { value: null },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const input = within(canvasElement).getByRole('textbox', { name: 'Ngày hiệu lực' })
    await userEvent.type(input, '05/01/2026{Enter}')
    expect(args.onChange).toHaveBeenCalledWith('2026-01-05')
    expect(input).toHaveValue('05/01/2026')
  },
}

export const Filled: Story = {
  args: { value: '2026-01-05' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const input = within(canvasElement).getByRole('textbox', { name: 'Ngày hiệu lực' })
    expect(input).toHaveValue('05/01/2026')
    await userEvent.clear(input)
    await userEvent.type(input, '31/02/2026')
    await userEvent.tab()
    expect(args.onChange).not.toHaveBeenCalled()
    expect(input).toHaveValue('05/01/2026')
  },
}

export const PickFromCalendar: Story = {
  args: { value: '2026-01-05' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const input = within(canvasElement).getByRole('textbox', { name: 'Ngày hiệu lực' })
    const body = within(canvasElement.ownerDocument.body)
    input.focus()
    await userEvent.keyboard('{ArrowDown}')
    const day5 = await body.findByRole('gridcell', { name: '5' })
    await waitFor(() => expect(day5).toHaveFocus())

    await userEvent.keyboard('{ArrowRight}{Enter}')
    expect(args.onChange).toHaveBeenCalledWith('2026-01-06')
    await waitFor(() => expect(body.queryByRole('grid')).not.toBeInTheDocument())
    expect(input).toHaveFocus()
    expect(input).toHaveValue('06/01/2026')
  },
}

export const OpenOnClick: Story = {
  args: { value: null },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const input = within(canvasElement).getByRole('textbox', { name: 'Ngày hiệu lực' })
    await userEvent.click(input)
    expect(await within(canvasElement.ownerDocument.body).findByRole('grid')).toBeInTheDocument()
    // Focus stays in the input so the user can still type.
    expect(input).toHaveFocus()
  },
}

export const MinMaxDate: Story = {
  args: { value: '2026-10-20', minDate: '2026-10-10', maxDate: '2026-10-25' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const input = within(canvasElement).getByRole('textbox', { name: 'Ngày hiệu lực' })
    await userEvent.clear(input)
    await userEvent.type(input, '09/10/2026{Enter}')
    expect(args.onChange).not.toHaveBeenCalled()
    expect(input).toHaveValue('20/10/2026')
  },
}

export const Disabled: Story = {
  args: { value: '2026-01-05', disabled: true, clearable: true, clearLabel: 'Xóa' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('textbox', { name: 'Ngày hiệu lực' })).toBeDisabled()
    expect(canvas.queryByRole('button', { name: 'Xóa' })).not.toBeInTheDocument()
  },
}

export const ReadOnly: Story = {
  args: { value: '2026-01-05', readOnly: true, clearable: true, clearLabel: 'Xóa' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('textbox', { name: 'Ngày hiệu lực' }))
    expect(within(canvasElement.ownerDocument.body).queryByRole('grid')).not.toBeInTheDocument()
    expect(canvas.queryByRole('button', { name: 'Xóa' })).not.toBeInTheDocument()
  },
}

export const Error: Story = {
  args: { value: null, error: 'Bắt buộc nhập ngày' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByRole('textbox', { name: 'Ngày hiệu lực' })).toHaveAttribute('aria-invalid', 'true')
  },
}

export const Clearable: Story = {
  args: { value: '2026-01-05', clearable: true, clearLabel: 'Xóa' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const input = canvas.getByRole('textbox', { name: 'Ngày hiệu lực' })
    await userEvent.click(canvas.getByRole('button', { name: 'Xóa' }))
    expect(args.onChange).toHaveBeenLastCalledWith(null)
    expect(input).toHaveValue('')

    // Emptying the text by hand clears too.
    await userEvent.type(input, '05/01/2026{Enter}')
    await userEvent.clear(input)
    await userEvent.tab()
    expect(args.onChange).toHaveBeenLastCalledWith(null)
  },
}

export const LeftSection: Story = {
  args: { value: null, leftSection: <IconCalendar size={14} /> },
  render: (args) => <Controlled {...args} />,
}

export const Required: Story = {
  args: { value: null, required: true },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByRole('textbox', { name: 'Ngày hiệu lực' })).toHaveAttribute('aria-required', 'true')
  },
}
