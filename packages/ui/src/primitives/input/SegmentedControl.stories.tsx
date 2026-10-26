import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { SegmentedControl, type SegmentedControlProps } from './SegmentedControl.tsx'

const data = [
  { value: 'day', label: 'Ngày' },
  { value: 'week', label: 'Tuần' },
  { value: 'month', label: 'Tháng' },
]

const meta: Meta<typeof SegmentedControl> = {
  title: 'Input/SegmentedControl',
  component: SegmentedControl,
  args: { data, 'aria-label': 'Kỳ báo cáo', onChange: fn() },
}

export default meta

type Story = StoryObj<typeof SegmentedControl>

function Controlled(props: SegmentedControlProps) {
  const [value, setValue] = useState(props.value)
  return (
    <SegmentedControl
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
  args: { value: 'day' },
  render: (args) => <Controlled {...args} />,
}

export const FullWidth: Story = {
  args: { value: 'day', fullWidth: true },
  render: (args) => (
    <div style={{ width: 320 }}>
      <Controlled {...args} />
    </div>
  ),
}

export const Disabled: Story = {
  args: { value: 'day', disabled: true },
  render: (args) => <Controlled {...args} />,
}

export const MoveWithArrowsAndSpace: Story = {
  tags: ['!dev', '!autodocs'],
  args: { value: 'day' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const day = canvas.getByRole('button', { name: 'Ngày' })
    day.focus()
    await userEvent.keyboard('{ArrowRight}')
    await userEvent.keyboard(' ')
    expect(args.onChange).toHaveBeenCalledWith('week')
    expect(canvas.getByRole('button', { name: 'Tuần' })).toHaveAttribute('data-pressed', '')
  },
}
