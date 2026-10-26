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
