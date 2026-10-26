import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Avatar } from '../data-display/Avatar.tsx'
import { Indicator } from './Indicator.tsx'

const meta: Meta<typeof Indicator> = {
  title: 'Feedback/Indicator',
  component: Indicator,
}

export default meta

type Story = StoryObj<typeof Indicator>

export const Dot: Story = {
  args: { children: <Avatar alt="Nguyễn Văn A">NA</Avatar> },
}

export const Label: Story = {
  args: { label: 3, children: <Avatar alt="Nguyễn Văn A">NA</Avatar> },
  play: async ({ canvasElement }) => {
    const label = within(canvasElement).getByText('3')
    expect(label).toBeInTheDocument()
  },
}

export const Disabled: Story = {
  args: { disabled: true, children: <Avatar alt="Nguyễn Văn A">NA</Avatar> },
}

export const Offset: Story = {
  args: { label: 5, offset: 4, children: <Avatar alt="Nguyễn Văn A">NA</Avatar> },
  play: async ({ canvasElement }) => {
    const badge = within(canvasElement).getByText('5')
    const root = badge.parentElement!.getBoundingClientRect()
    const box = badge.getBoundingClientRect()
    // Centred 4px in from the top-right corner.
    expect(Math.round(box.top + box.height / 2 - root.top)).toBe(4)
    expect(Math.round(root.right - (box.left + box.width / 2))).toBe(4)
  },
}
