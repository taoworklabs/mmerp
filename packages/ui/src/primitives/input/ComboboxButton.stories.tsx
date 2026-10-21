import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { Button } from './Button.tsx'
import { ComboboxButton, type ComboboxButtonProps } from './ComboboxButton.tsx'
import type { SelectItem } from './Select.tsx'

const data: SelectItem[] = [
  { value: 'a', label: 'Nguyễn Văn An' },
  { value: 'b', label: 'Đặng Thị Bích' },
  { value: 'c', label: 'Trần Văn C' },
]

const meta: Meta<typeof ComboboxButton> = {
  title: 'Input/ComboboxButton',
  component: ComboboxButton,
  args: { data, nothingFound: 'No people found', onChange: fn() },
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
  args: { value: null, trigger: <Button aria-label="Pick a person">Pick a person</Button> },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const trigger = within(canvasElement).getByRole('combobox', { name: 'Pick a person' })
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

export const Searchable: Story = {
  args: { value: null, trigger: <Button aria-label="Pick a person">Pick a person</Button>, searchable: true, searchLabel: 'Search people' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const trigger = within(canvasElement).getByRole('combobox', { name: 'Pick a person' })
    const body = within(canvasElement.ownerDocument.body)

    await userEvent.click(trigger)
    await waitFor(() => expect(body.getByRole('listbox')).toBeInTheDocument())
    const search = body.getByRole('combobox', { name: 'Search people' })
    await waitFor(() => expect(search).toHaveFocus())

    await userEvent.type(search, 'nguyen')
    const options = body.getAllByRole('option')
    expect(options).toHaveLength(1)
    expect(options[0]).toHaveTextContent('Nguyễn Văn An')

    await userEvent.clear(search)
    await userEvent.type(search, 'zzz')
    expect(body.getByText('No people found')).toBeInTheDocument()
  },
}
