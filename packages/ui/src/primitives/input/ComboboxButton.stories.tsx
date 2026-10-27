import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { Button } from './Button.tsx'
import { ComboboxButton, type ComboboxButtonProps } from './ComboboxButton.tsx'
import type { SelectItem } from './Select.tsx'

const data: SelectItem[] = [
  { value: 'a', label: 'Nguyễn Văn An' },
  { value: 'b', label: 'Đặng Thị Bích' },
  { value: 'c', label: 'Trần Văn Cường' },
]

const meta: Meta<typeof ComboboxButton> = {
  title: 'Input/ComboboxButton',
  component: ComboboxButton,
  args: { data, nothingFound: 'Không tìm thấy nhân viên', onChange: fn() },
}

export default meta

type Story = StoryObj<typeof ComboboxButton>

function Controlled(props: ComboboxButtonProps) {
  const [value, setValue] = useState(props.value)
  return (
    <ComboboxButton
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
  args: { value: null, trigger: <Button aria-label="Chọn nhân viên">Chọn nhân viên</Button> },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const trigger = within(canvasElement).getByRole('combobox', { name: 'Chọn nhân viên' })
    const body = within(canvasElement.ownerDocument.body)

    await userEvent.click(trigger)
    await waitFor(() => expect(body.getByRole('listbox')).toBeInTheDocument())

    await userEvent.keyboard('{ArrowDown}')
    await userEvent.keyboard('{Enter}')
    expect(args.onChange).toHaveBeenCalledWith('a')
    await waitFor(() => expect(body.queryByRole('listbox')).not.toBeInTheDocument())
    await waitFor(() => expect(trigger).toHaveFocus())

    await userEvent.click(trigger)
    await waitFor(() => expect(body.getByRole('listbox')).toBeInTheDocument())
    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(body.queryByRole('listbox')).not.toBeInTheDocument())
    await waitFor(() => expect(trigger).toHaveFocus())
  },
}

// Pins the hidden input's accessible name: the trigger's own keydown handler moves focus onto it
// for ArrowDown/ArrowUp, so it must carry a name — left open so the a11y check covers the popup too.
export const KeyboardOpen: Story = {
  tags: ['!dev', '!autodocs'],
  args: { value: null, trigger: <Button aria-label="Chọn nhân viên">Chọn nhân viên</Button> },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const trigger = within(canvasElement).getByRole('combobox', { name: 'Chọn nhân viên' })
    const body = within(canvasElement.ownerDocument.body)

    trigger.focus()
    await userEvent.keyboard('{ArrowDown}')
    await waitFor(() => expect(body.getByRole('listbox')).toBeInTheDocument())

    expect(document.activeElement).not.toBe(trigger)
    expect(document.activeElement).toHaveAccessibleName('Chọn nhân viên')
  },
}

export const Searchable: Story = {
  args: { value: null, trigger: <Button aria-label="Chọn nhân viên">Chọn nhân viên</Button>, searchable: true, searchLabel: 'Tìm nhân viên' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const trigger = within(canvasElement).getByRole('combobox', { name: 'Chọn nhân viên' })
    const body = within(canvasElement.ownerDocument.body)

    await userEvent.click(trigger)
    await waitFor(() => expect(body.getByRole('listbox')).toBeInTheDocument())
    const search = body.getByRole('combobox', { name: 'Tìm nhân viên' })
    await waitFor(() => expect(search).toHaveFocus())

    await userEvent.type(search, 'nguyen')
    const options = body.getAllByRole('option')
    expect(options).toHaveLength(1)
    expect(options[0]).toHaveTextContent('Nguyễn Văn An')

    await userEvent.clear(search)
    await userEvent.type(search, 'zzz')
    expect(await body.findByText('Không tìm thấy nhân viên')).toBeInTheDocument()
  },
}

// A row built of parts (an indent, an icon, a truncating name) ends long names with an ellipsis
// inside the popup instead of running past its edge.
export const LongOptions: Story = {
  args: {
    value: null,
    popupWidth: 220,
    data: [{ value: 'a', label: 'Hành chính – Nhân sự và Tổng hợp khu vực phía Nam' }],
    renderOption: (item) => (
      <span style={{ display: 'flex', gap: 8, paddingLeft: 24 }}>
        <span aria-hidden="true">■</span>
        <span data-testid="name" style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {item.label}
        </span>
      </span>
    ),
    trigger: <Button aria-label="Chọn đơn vị">Chọn đơn vị</Button>,
  },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const body = within(canvasElement.ownerDocument.body)
    await userEvent.click(within(canvasElement).getByRole('combobox', { name: 'Chọn đơn vị' }))
    const name = await body.findByTestId('name')
    const option = name.closest('[role="option"]') as HTMLElement
    expect(name.scrollWidth).toBeGreaterThan(name.clientWidth)
    expect(name.getBoundingClientRect().right).toBeLessThanOrEqual(option.getBoundingClientRect().right)
  },
}
