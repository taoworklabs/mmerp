import type { Meta, StoryObj } from '@storybook/react-vite'
import { Paper } from './Paper.tsx'

const meta: Meta<typeof Paper> = {
  title: 'Layout/Paper',
  component: Paper,
}

export default meta

type Story = StoryObj<typeof Paper>

export const Default: Story = {
  args: { children: 'Surface' },
}

export const WithBorder: Story = {
  args: { withBorder: true, children: 'Bordered surface' },
}

export const Padding: Story = {
  args: { padding: 'lg', children: 'Padded surface' },
}

export const ShadowXs: Story = {
  args: { shadow: 'xs', children: 'Raised surface' },
}

export const ShadowSm: Story = {
  args: { shadow: 'sm', children: 'More raised surface' },
}
