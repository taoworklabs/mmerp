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
